import 'dotenv/config';
import express from 'express';
import { Sandbox } from '@e2b/code-interpreter';
import { timestampSlug, shortId } from './lib/util.js';
import { collectFiles } from './lib/collect.js';

const app = express();
app.use(express.json());

function required(name) {
  const v = process.env[name];
  if (!v) console.warn(`[warn] ${name} is not set`);
  return v;
}

function assertEnv(...names) {
  const missing = names.filter(n => !process.env[n]);
  if (missing.length) {
    throw new Error(`Missing required env vars: ${missing.join(', ')}`);
  }
}

async function writeTemplateToSandbox(sandbox, files) {
  await sandbox.files.write(files);
}

async function run(moduleDir) {
  assertEnv('E2B_API_KEY');

  const TEMPLATE = process.env.TEMPLATE || 'basic-express';
  const SERVICE_ENV = process.env.SERVICE_ENV || 'prod';
  const MODULE_DIR = moduleDir;

  const nowSlug = timestampSlug(new Date())
  const repoName = `proj-${nowSlug}-${shortId()}`
  const serviceName = `svc-${repoName}-${SERVICE_ENV}`

  const SANDBOX_APP_DIR = '/tmp/app'
  const REPO_META_PATH = '/tmp/repo_meta.json'

  const ghToken = required('GITHUB_TOKEN')
  const ghOwner = process.env.GITHUB_OWNER || ''
  const renderToken = process.env.RENDER_API_TOKEN || ''
  const renderOwnerId = process.env.RENDER_OWNER_ID || ''
  const renderRegion = process.env.RENDER_REGION || 'oregon'
  const renderPlan = process.env.RENDER_PLAN || 'starter'
  const renderRootDir = process.env.RENDER_ROOT_DIR || '.'
  const nodeVersion = process.env.NODE_VERSION || '18'

  console.log(`Creating E2B sandbox...`)
  const sandbox = await Sandbox.create({
    // Inject tokens and config into the sandbox as env vars
    envs: {
      GH_TOKEN: ghToken || '',
      GH_OWNER: ghOwner || '',
      GH_REPO_PRIVATE: (process.env.GITHUB_REPO_PRIVATE ?? 'false'),
      RENDER_TOKEN: renderToken || '',
      RENDER_OWNER_ID: renderOwnerId || '',
      RENDER_REGION: renderRegion,
      RENDER_PLAN: renderPlan,
      RENDER_ROOT_DIR: renderRootDir,
      NODE_VERSION: nodeVersion,
      SERVICE_ENV,
      REPO_NAME: repoName,
      SERVICE_NAME: serviceName,
    },
  })

  try {
    console.log(`Sandbox started: ${sandbox.id}`)
    console.log(`Writing code into sandbox at ${SANDBOX_APP_DIR} ...`)
    await sandbox.commands.run(`mkdir -p ${SANDBOX_APP_DIR}`)

    let files = []
    if (!MODULE_DIR) {
        throw new Error('Module directory is not specified');
    }
    console.log(`Uploading local module from '${MODULE_DIR}' into sandbox...`);
    files = collectFiles(MODULE_DIR, SANDBOX_APP_DIR);

    await writeTemplateToSandbox(sandbox, files)

    console.log(`Template/files written. Listing ${SANDBOX_APP_DIR}:`)
    const list = await sandbox.files.list(SANDBOX_APP_DIR)
    console.log(list)

    console.log(`Ensuring Python 'requests' is available in the sandbox...`)
    try {
      await sandbox.commands.run('python -c "import requests"')
    } catch (e) {
      console.log('Installing requests via pip...')
      await sandbox.commands.run('python -m pip install -q requests || pip install -q requests')
    }

    console.log(`Creating GitHub repo from within sandbox (Python + GitHub API)...`)

    const py = `
import os, json, base64, pathlib, requests

GH_TOKEN = os.environ.get('GH_TOKEN')
GH_OWNER = os.environ.get('GH_OWNER')
GH_REPO_PRIVATE = os.environ.get('GH_REPO_PRIVATE', 'false')
REPO_NAME = os.environ.get('REPO_NAME')
SERVICE_NAME = os.environ.get('SERVICE_NAME')
RENDER_TOKEN = os.environ.get('RENDER_TOKEN')
RENDER_OWNER_ID = os.environ.get('RENDER_OWNER_ID')
RENDER_REGION = os.environ.get('RENDER_REGION', 'oregon')
RENDER_PLAN = os.environ.get('RENDER_PLAN', 'starter')

if not GH_TOKEN:
    raise RuntimeError('GH_TOKEN is not set in sandbox envs')

headers = {
    'Authorization': f'Bearer {GH_TOKEN}',
    'Accept': 'application/vnd.github+json',
    'X-GitHub-Api-Version': '2022-11-28',
    'User-Agent': 'e2b-sandbox'
}

# Create repo
if GH_OWNER:
    create_url = f'https://api.github.com/orgs/{GH_OWNER}/repos'
else:
    create_url = 'https://api.github.com/user/repos'

def parse_bool(s):
    return str(s).strip().lower() in ('1','true','yes','y','on')

payload = {
    'name': REPO_NAME,
    'private': parse_bool(GH_REPO_PRIVATE),
    'description': 'Repo created from E2B sandbox',
    'auto_init': False
}
resp = requests.post(create_url, headers=headers, json=payload)
print('Create repo status:', resp.status_code)
try:
    repo_json = resp.json()
except Exception:
    repo_json = {'raw': resp.text}
print('Create repo resp:', json.dumps(repo_json, indent=2))
if resp.status_code not in (201, 202):
    raise RuntimeError(f'Failed to create repo: {resp.status_code} {resp.text}')

owner_login = repo_json['owner']['login']
repo_name = repo_json['name']

# Write repo metadata for the outer orchestrator to read
pathlib.Path('${REPO_META_PATH}').write_text(json.dumps({
    'owner': owner_login,
    'name': repo_name
}))
`

    const exec = await sandbox.runCode(py)
    console.log(exec.logs)
    if (exec.error) {
      throw new Error(`Python error while creating repo: ${exec.error}`)
    }

    // Read repo metadata
    const metaRaw = await sandbox.files.read(REPO_META_PATH)
    const meta = JSON.parse(metaRaw)
    console.log('Repo created:', meta)

    // Try to push via git inside sandbox
    console.log('Attempting git push from within sandbox...')
    try {
      await sandbox.commands.run(
        [
          `cd ${SANDBOX_APP_DIR}`,
          'git init',
          'git checkout -b main || git branch -M main',
          'git config user.email "sandbox-bot@example.com"',
          'git config user.name "Sandbox Bot"',
          'git add .',
          'git commit -m "Initial commit"',
          `git remote add origin https://x-access-token:$GH_TOKEN@github.com/${meta.owner}/${meta.name}.git`,
          'git push -u origin main'
        ].join(' && ')
      )
      console.log('Git push succeeded.')
    } catch (gitErr) {
      console.warn('Git push failed in sandbox, falling back to GitHub Contents API...', gitErr?.message || gitErr)
      const pyUpload = `
import os, json, base64, pathlib, requests
GH_TOKEN = os.environ.get('GH_TOKEN')
meta = json.loads(pathlib.Path('${REPO_META_PATH}').read_text())
owner_login = meta['owner']
repo_name = meta['name']
headers = {
    'Authorization': f'Bearer {GH_TOKEN}',
    'Accept': 'application/vnd.github+json',
    'X-GitHub-Api-Version': '2022-11-28',
    'User-Agent': 'e2b-sandbox'
}
app_root = pathlib.Path('${SANDBOX_APP_DIR}')
for p in app_root.rglob('*'):
    if p.is_file():
        rel_path = str(p.relative_to(app_root)).replace('\\\\', '/')
        with open(p, 'rb') as f:
            content_b64 = base64.b64encode(f.read()).decode()
        api_url = f'https://api.github.com/repos/{owner_login}/{repo_name}/contents/{rel_path}'
        sha = None
        get_resp = requests.get(api_url, headers=headers, params={'ref': 'main'})
        if get_resp.status_code == 200:
            try:
                sha = get_resp.json().get('sha')
            except Exception:
                sha = None
        put_payload = {
            'message': f'Add/Update {rel_path}',
            'content': content_b64,
            'branch': 'main'
        }
        if sha:
            put_payload['sha'] = sha
        put_resp = requests.put(api_url, headers=headers, json=put_payload)
        print(f'PUT {rel_path} ->', put_resp.status_code)
        if put_resp.status_code not in (201, 200):
            print('Error body:', put_resp.text)
            raise RuntimeError(f'Failed to upload {rel_path}')
print('Fallback upload completed.')
`
    const up = await sandbox.runCode(pyUpload)
    console.log(up.logs)
    if (up.error) {
      throw new Error(`Fallback upload error: ${up.error}`)
    }
  }

    // Render provisioning (optional)
    if (renderToken) {
      console.log('Provisioning Render service via API...')
      console.log('Render config:', {
        ownerId: renderOwnerId || '(auto-resolve)',
        region: renderRegion,
        plan: renderPlan,
        serviceName,
      })
      const pyRender = `
import os, json, requests, time

RENDER_TOKEN = os.environ.get('RENDER_TOKEN')
RENDER_OWNER_ID = os.environ.get('RENDER_OWNER_ID')
RENDER_REGION = os.environ.get('RENDER_REGION', 'oregon')
RENDER_PLAN = os.environ.get('RENDER_PLAN', 'starter')
RENDER_ROOT_DIR = os.environ.get('RENDER_ROOT_DIR', '.')
NODE_VERSION = os.environ.get('NODE_VERSION', '18')
SERVICE_NAME = os.environ.get('SERVICE_NAME')

meta = json.loads(open('${REPO_META_PATH}').read())
owner_login = meta['owner']
repo_name = meta['name']

headers = {'Authorization': f'Bearer {RENDER_TOKEN}', 'Content-Type': 'application/json'}

if not RENDER_OWNER_ID:
    owners_resp = requests.get('https://api.render.com/v1/owners', headers=headers)
    if owners_resp.status_code != 200:
        raise RuntimeError(f'Failed to list owners: {owners_resp.status_code} {owners_resp.text}')
    try:
        owners_raw = owners_resp.json()
    except Exception:
        raise RuntimeError(f'Owners response not JSON: {owners_resp.text}')
    # Print full owners response for diagnostics
    try:
        print('Owners response:', json.dumps(owners_raw, indent=2))
    except Exception:
        print('Owners raw response (non-JSON):', owners_resp.text)

    owners_list = []
    if isinstance(owners_raw, list):
        owners_list = owners_raw
    elif isinstance(owners_raw, dict):
        if 'owners' in owners_raw and isinstance(owners_raw['owners'], list):
            owners_list = owners_raw['owners']
        elif 'items' in owners_raw and isinstance(owners_raw['items'], list):
            owners_list = owners_raw['items']
        else:
            owners_list = []

    def get_owner_id(o):
        if isinstance(o, dict):
            return o.get('id') or o.get('ownerId') or (o.get('owner') or {}).get('id')
        return None

    user_owner = next((o for o in owners_list if (isinstance(o, dict) and (o.get('type') == 'user' or o.get('ownerType') == 'user'))), None)
    chosen = user_owner or (owners_list[0] if owners_list else None)
    oid = get_owner_id(chosen) if chosen else None
    if not oid:
        raise RuntimeError('Could not determine Render owner id from response')
    RENDER_OWNER_ID = oid

base = {
    'type': 'web_service',
    'name': SERVICE_NAME,
    'ownerId': RENDER_OWNER_ID,
    'repo': f'https://github.com/{owner_login}/{repo_name}',
    'branch': 'main',
    'region': RENDER_REGION,
    'plan': RENDER_PLAN,
    # Be explicit when using monorepos located at repo root
    'rootDir': RENDER_ROOT_DIR,
}

# Three shapes to satisfy differing API expectations
shape_a = {
    **base,
    'serviceDetails': {
        'env': 'node',
        'runtime': 'node',
        'buildCommand': 'npm ci && npm run build',
        'startCommand': 'npm start',
        'autoDeploy': True,
    },
}

shape_b = {
    **base,
    'serviceDetails': {
        'env': 'node',
        'runtime': 'node',
        'autoDeploy': True,
        'envSpecificDetails': {
            'node': {
                'installCommand': 'npm ci',
                'buildCommand': 'npm run build',
                'startCommand': 'npm start',
                'nodeVersion': NODE_VERSION
            }
        }
    },
}

shape_c = {
    **base,
    'serviceDetails': {
        'env': 'node',
        'runtime': 'node',
        'buildCommand': 'npm ci && npm run build',
        'startCommand': 'npm start',
        'autoDeploy': True,
        'envSpecificDetails': {
            'node': {
                'installCommand': 'npm ci',
                'buildCommand': 'npm run build',
                'startCommand': 'npm start',
                'nodeVersion': NODE_VERSION
            }
        }
    },
}

# Some accounts expect top-level build/start in addition to serviceDetails
shape_d = {
    **base,
    'buildCommand': 'npm ci && npm run build',
    'startCommand': 'npm start',
    'serviceDetails': {
        'env': 'node',
        'runtime': 'node',
        'autoDeploy': True,
    },
}

# shape_e: top-level build/start + serviceDetails with envSpecificDetails and both env/runtime
shape_e = {
    **base,
    'buildCommand': 'npm ci && npm run build',
    'startCommand': 'npm start',
    'serviceDetails': {
        'env': 'node',
        'runtime': 'node',
        'autoDeploy': True,
        'envSpecificDetails': {
            'node': {
                'installCommand': 'npm ci',
                'buildCommand': 'npm run build',
                'startCommand': 'npm start',
                'nodeVersion': NODE_VERSION
            }
        }
    },
}

# shape_f: use runtime only instead of env
shape_f = {
    **base,
    'serviceDetails': {
        'runtime': 'node',
        'buildCommand': 'npm ci && npm run build',
        'startCommand': 'npm start',
        'autoDeploy': True,
        'envSpecificDetails': {
            'node': {
                'installCommand': 'npm ci',
                'buildCommand': 'npm run build',
                'startCommand': 'npm start',
                'nodeVersion': NODE_VERSION
            }
        }
    },
}

payloads = [shape_e, shape_d, shape_f, shape_c, shape_b, shape_a]

max_attempts = 5
backoff = 3
svc_resp = None
last_body = None
for idx, payload in enumerate(payloads, start=1):
    print(f'Trying payload shape #{idx}:', json.dumps(payload, indent=2))
    for attempt in range(1, max_attempts + 1):
        svc_resp = requests.post('https://api.render.com/v1/services', headers=headers, json=payload)
        print('Create service status:', svc_resp.status_code)
        try:
            body = svc_resp.json()
            last_body = body
            print('Create service resp:', json.dumps(body, indent=2))
        except Exception:
            body = None
            print('Create service raw resp:', svc_resp.text)
        if svc_resp.status_code in (201, 200):
            break
        msg = (body or {}).get('message', '') if isinstance(body, dict) else ''
        if ('unfetchable' in msg) or ('repository URL is invalid or unfetchable' in msg):
            if attempt < max_attempts:
                sleep_s = backoff * attempt
                print(f'Retrying after {sleep_s}s due to unfetchable repo (attempt {attempt}/{max_attempts})...')
                time.sleep(sleep_s)
                continue
        break
    if svc_resp.status_code in (201, 200):
        break

if svc_resp is None or svc_resp.status_code not in (201, 200):
    try:
        print('Last error body:', json.dumps(last_body, indent=2))
    except Exception:
        pass
    raise RuntimeError(f'Failed to create Render service: {svc_resp.status_code}')

svc = svc_resp.json()
service_id = svc.get('id')
if service_id:
    dep_resp = requests.post(f'https://api.render.com/v1/services/{service_id}/deploys', headers=headers)
    print('Trigger deploy status:', dep_resp.status_code)
    try:
        print('Trigger deploy resp:', json.dumps(dep_resp.json(), indent=2))
    except Exception:
        print('Trigger deploy raw resp:', dep_resp.text)
      `
      const rd = await sandbox.runCode(pyRender)
      console.log(rd.logs)
      if (rd.error) {
        // Surface richer diagnostics instead of [object Object]
        const errStr = typeof rd.error === 'string' ? rd.error : (rd.error?.message || JSON.stringify(rd.error, null, 2))
        console.error('Render provisioning error details:', {
          error: errStr,
          stdout: rd.logs?.stdout,
          stderr: rd.logs?.stderr,
        })
        throw new Error(`Render provisioning error: ${errStr}`)
      }
    } else {
      console.log('RENDER_API_TOKEN not set; skipping Render provisioning.')
    }
  } finally {
    // Allow sandbox TTL to expire naturally; explicit close not required
    console.log('All steps attempted. Sandbox will shut down automatically.')
  }
}

app.post('/', async (req, res) => {
  const { path } = req.body;

  if (!path) {
    return res.status(400).send({ error: 'Missing path in request body' });
  }

  try {
    await run(path);
    res.status(200).send({ message: 'Process completed successfully' });
  } catch (error) {
    console.error(error);
    res.status(500).send({ error: 'An error occurred' });
  }
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Server listening on port ${port}`);
});
