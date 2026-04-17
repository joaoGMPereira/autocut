import fs from 'node:fs';
import path from 'node:path';
import { API_BASE, TEST_DATA_DIR } from './env';
import { getPipelineRun, getPipelineClips } from './db';

export interface AdvanceBody {
  url?: string;
  channel_id?: number;
  mode?: 'ai' | 'longform';
  mode_config?: Record<string, unknown>;
}

export async function advanceGate(runId: number, body: AdvanceBody): Promise<{ state: string; video_reused: boolean }> {
  const res = await fetch(`${API_BASE}/api/pipeline/runs/${runId}/advance`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`POST /api/pipeline/runs/${runId}/advance: ${res.status} ${text}`);
  }
  return (await res.json()) as { state: string; video_reused: boolean };
}

export async function waitForState(
  runId: number,
  target: string | string[],
  timeoutMs = 120_000,
  pollMs = 500,
): Promise<string> {
  const targets = Array.isArray(target) ? target : [target];
  const deadline = Date.now() + timeoutMs;
  let last: string | undefined;
  while (Date.now() < deadline) {
    const row = getPipelineRun(runId);
    last = row?.state;
    if (last === 'ERROR') {
      throw new Error(`run ${runId} transitioned to ERROR while waiting for ${targets.join('|')}`);
    }
    if (last && targets.includes(last)) return last;
    await sleep(pollMs);
  }
  throw new Error(
    `timed out waiting for run ${runId} to reach ${targets.join('|')}; last state: ${last}`,
  );
}

export function getDownloadedVideoPath(runId: number): string | null {
  const row = getPipelineRun(runId);
  if (!row?.video_path) return null;
  if (!fs.existsSync(row.video_path)) return null;
  return row.video_path;
}

export function assertVideoDownloaded(runId: number): void {
  const row = getPipelineRun(runId);
  if (!row) throw new Error(`run ${runId} not found`);
  if (!row.video_path) throw new Error(`run ${runId} has no video_path`);
  if (!fs.existsSync(row.video_path)) {
    throw new Error(`run ${runId} video_path points to missing file: ${row.video_path}`);
  }
  const stat = fs.statSync(row.video_path);
  if (stat.size === 0) {
    throw new Error(`run ${runId} video file is empty: ${row.video_path}`);
  }
}

export function assertClipFilesExist(runId: number): string[] {
  const clips = getPipelineClips(runId);
  if (clips.length === 0) throw new Error(`run ${runId} has no clips rows in DB`);
  const missing: string[] = [];
  const found: string[] = [];
  for (const c of clips) {
    if (!c.file_path) {
      missing.push(`clip ${c.id}: empty file_path`);
      continue;
    }
    if (!fs.existsSync(c.file_path)) {
      missing.push(`clip ${c.id}: ${c.file_path} does not exist`);
      continue;
    }
    found.push(c.file_path);
  }
  if (missing.length > 0) {
    throw new Error(`clip files missing:\n${missing.join('\n')}`);
  }
  return found;
}

export function clipsDirExistsForVideo(videoIdOrPattern: string): string[] {
  const clipsRoot = path.join(TEST_DATA_DIR, 'clips');
  if (!fs.existsSync(clipsRoot)) return [];
  return fs
    .readdirSync(clipsRoot, { withFileTypes: true })
    .filter((e) => e.isDirectory() && e.name.startsWith(videoIdOrPattern))
    .map((e) => path.join(clipsRoot, e.name));
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}
