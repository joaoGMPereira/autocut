import { spawn, ChildProcess } from 'node:child_process';
import path from 'node:path';
import { API_BASE, TEST_WEB_PORT, WEB_BASE } from './env';
import { waitForHealth } from './server';

const REPO_ROOT = path.resolve(__dirname, '..', '..');
const WEB_DIR = path.join(REPO_ROOT, 'apps', 'web');

let nextProc: ChildProcess | null = null;

export async function startNextDev(): Promise<void> {
  console.log('[e2e] starting Next.js dev on port', TEST_WEB_PORT);
  // Invoke `next` directly (not via the package `dev` script which hardcodes --port 3201).
  nextProc = spawn(
    'pnpm',
    ['exec', 'next', 'dev', '--port', String(TEST_WEB_PORT)],
    {
      cwd: WEB_DIR,
      env: {
        ...process.env,
        NEXT_PUBLIC_GO_URL: API_BASE,
        NEXT_PUBLIC_AUTOCUT_TEST_MODE: 'true',
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    },
  );
  nextProc.stdout?.on('data', (d: Buffer) => process.stdout.write(`[next] ${d}`));
  nextProc.stderr?.on('data', (d: Buffer) => process.stderr.write(`[next] ${d}`));
  // Next.js dev takes long on first compile; extend timeout.
  await waitForHealth(`${WEB_BASE}/api/health`, 180_000);
  console.log('[e2e] Next.js healthy');
}

export function stopNextDev(): void {
  if (nextProc && !nextProc.killed) {
    console.log('[e2e] stopping Next.js dev');
    nextProc.kill('SIGTERM');
    nextProc = null;
  }
}
