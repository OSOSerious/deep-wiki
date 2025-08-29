import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import crypto from 'crypto';

function pad(n) { return n.toString().padStart(2, '0'); }

export function timestampSlug(date = new Date()) {
  const y = date.getFullYear();
  const m = pad(date.getMonth() + 1);
  const d = pad(date.getDate());
  const hh = pad(date.getHours());
  const mm = pad(date.getMinutes());
  const ss = pad(date.getSeconds());
  return `${y}${m}${d}-${hh}${mm}${ss}`;
}

export function shortId(len = 5) {
  // URL-safe short id
  return crypto.randomBytes(Math.ceil((len * 3) / 4)).toString('base64').replace(/[^a-zA-Z0-9]/g, '').slice(0, len).toLowerCase();
}

export function ensureDir(p) {
  fs.mkdirSync(p, { recursive: true });
}

export function writeFile(filePath, content) {
  ensureDir(path.dirname(filePath));
  fs.writeFileSync(filePath, content);
}

export function run(cmd, options = {}) {
  const { cwd, env } = options;
  // Default stdio inherit for better visibility
  return execSync(cmd, { stdio: 'inherit', cwd, env: { ...process.env, ...(env || {}) } });
}

