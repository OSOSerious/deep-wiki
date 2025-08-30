import fs from 'fs'
import path from 'path'

const IGNORES = new Set(['node_modules', '.git', '.DS_Store'])

function shouldIgnore(name) {
  return IGNORES.has(name)
}

export function collectFiles(localDir, baseInSandbox = '/workspace/app') {
  const out = []
  const root = path.resolve(localDir)

  function walk(dir) {
    const entries = fs.readdirSync(dir, { withFileTypes: true })
    for (const e of entries) {
      if (shouldIgnore(e.name)) continue
      const full = path.join(dir, e.name)
      const rel = path.relative(root, full)
      if (e.isDirectory()) {
        walk(full)
      } else if (e.isFile()) {
        // Read as UTF-8 text. For binaries, extend this to handle base64.
        const data = fs.readFileSync(full, 'utf8')
        const sandboxPath = baseInSandbox + '/' + rel.split(path.sep).join('/')
        out.push({ path: sandboxPath, data })
      }
    }
  }

  walk(root)
  return out
}
