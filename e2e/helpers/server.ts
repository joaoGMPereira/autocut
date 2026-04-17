import { spawn, ChildProcess, spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import http from 'node:http';
import { API_BASE, TEST_API_PORT, TEST_DATA_DIR, TEST_SERVER_BIN } from './env';

const REPO_ROOT = path.resolve(__dirname, '..', '..');
const SERVER_CMD_DIR = path.join(REPO_ROOT, 'server', 'cmd', 'server');
const ASSETS_DIR = path.join(REPO_ROOT, 'assets');

let goProc: ChildProcess | null = null;

export function compileGoServer(): void {
  console.log('[e2e] compiling Go server →', TEST_SERVER_BIN);
  const res = spawnSync(
    'go',
    ['build', '-o', TEST_SERVER_BIN, './cmd/server'],
    {
      cwd: path.join(REPO_ROOT, 'server'),
      env: { ...process.env, CGO_ENABLED: '0' },
      stdio: 'inherit',
    },
  );
  if (res.status !== 0) {
    throw new Error(`go build failed with exit code ${res.status}`);
  }
}

export async function startGoServer(): Promise<void> {
  fs.mkdirSync(TEST_DATA_DIR, { recursive: true });
  console.log('[e2e] starting Go server on port', TEST_API_PORT, 'dir', TEST_DATA_DIR);
  goProc = spawn(
    TEST_SERVER_BIN,
    [
      '-host', '127.0.0.1',
      '-port', String(TEST_API_PORT),
      '-dir', TEST_DATA_DIR,
      '-assets-dir', ASSETS_DIR,
    ],
    { stdio: ['ignore', 'pipe', 'pipe'] },
  );
  goProc.stdout?.on('data', (d: Buffer) => process.stdout.write(`[go] ${d}`));
  goProc.stderr?.on('data', (d: Buffer) => process.stderr.write(`[go] ${d}`));
  goProc.on('exit', (code) => {
    if (code !== 0 && code !== null) {
      console.error(`[e2e] Go server exited with code ${code}`);
    }
  });
  await waitForHealth(`${API_BASE}/health`, 30_000);
  console.log('[e2e] Go server healthy');
}

export function stopGoServer(): void {
  if (goProc && !goProc.killed) {
    console.log('[e2e] stopping Go server');
    goProc.kill('SIGTERM');
    goProc = null;
  }
}

export function waitForHealth(url: string, timeoutMs = 15_000): Promise<void> {
  return new Promise((resolve, reject) => {
    const start = Date.now();
    const tick = () => {
      http
        .get(url, (res) => {
          if (res.statusCode === 200) return resolve();
          retry();
        })
        .on('error', retry);
    };
    const retry = () => {
      if (Date.now() - start > timeoutMs) {
        return reject(new Error(`timeout waiting for ${url}`));
      }
      setTimeout(tick, 300);
    };
    tick();
  });
}
