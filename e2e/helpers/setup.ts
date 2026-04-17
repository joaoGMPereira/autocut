import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'node:child_process';
import { API_BASE, TEST_DATA_DIR } from './env';

export interface ToolStatus {
  name: string;
  installed: boolean;
  required: boolean;
  path?: string;
  version?: string;
  source?: string;
}

export async function getToolsStatus(): Promise<ToolStatus[]> {
  const res = await fetch(`${API_BASE}/api/setup/status`);
  if (!res.ok) throw new Error(`GET /api/setup/status: ${res.status}`);
  const body = (await res.json()) as { tools?: ToolStatus[] };
  return body.tools ?? [];
}

export function removeBinDir(): void {
  const binDir = path.join(TEST_DATA_DIR, 'bin');
  fs.rmSync(binDir, { recursive: true, force: true });
}

export function systemHas(tool: string): string | null {
  try {
    const out = execSync(`which ${tool}`, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] });
    return out.trim() || null;
  } catch {
    return null;
  }
}

export async function syncFromPath(): Promise<ToolStatus[]> {
  const res = await fetch(`${API_BASE}/api/setup/sync`, { method: 'POST' });
  if (!res.ok) throw new Error(`POST /api/setup/sync: ${res.status}`);
  const body = (await res.json()) as { tools?: ToolStatus[] };
  return body.tools ?? [];
}

// Seeds a ~1KB placeholder whisper model so the whisper-cli gate passes for
// tests that never actually run Whisper (e.g. Longform mode).
export function seedFakeWhisperModel(modelName = 'ggml-tiny.bin'): string {
  const dir = path.join(TEST_DATA_DIR, 'models', 'whisper');
  fs.mkdirSync(dir, { recursive: true });
  const file = path.join(dir, modelName);
  const filler = Buffer.alloc(1024, 0);
  fs.writeFileSync(file, filler);
  return file;
}

export async function triggerInstallViaApi(toolName: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/setup/install/${toolName}`, { method: 'POST' });
  if (!res.ok) throw new Error(`POST /api/setup/install/${toolName}: ${res.status}`);
}

export async function setCustomToolPath(toolName: string, customPath: string): Promise<Response> {
  return fetch(`${API_BASE}/api/setup/tools/${toolName}/path`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path: customPath }),
  });
}

export async function waitForToolInstalled(toolName: string, timeoutMs = 90_000): Promise<ToolStatus> {
  const deadline = Date.now() + timeoutMs;
  let last: ToolStatus | undefined;
  while (Date.now() < deadline) {
    const tools = await getToolsStatus();
    last = tools.find((t) => t.name === toolName);
    if (last?.installed) return last;
    await sleep(500);
  }
  throw new Error(
    `timed out waiting for ${toolName} to install; last status: ${JSON.stringify(last)}`,
  );
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}
