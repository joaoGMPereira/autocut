import { API_BASE } from './env';

async function req(method: string, path: string, body?: unknown): Promise<Response> {
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  return res;
}

export async function putSetting(key: string, value: string): Promise<void> {
  const res = await req('PUT', '/api/settings', { key, value });
  if (!res.ok) throw new Error(`PUT /api/settings ${key}: ${res.status} ${await res.text()}`);
}

export async function getSettings(): Promise<Record<string, string>> {
  const res = await req('GET', '/api/settings');
  if (!res.ok) throw new Error(`GET /api/settings: ${res.status}`);
  const rows = (await res.json()) as Array<{ key: string; value: string }>;
  const out: Record<string, string> = {};
  for (const r of rows ?? []) out[r.key] = r.value;
  return out;
}

export async function createRun(): Promise<number> {
  const res = await req('POST', '/api/pipeline/runs');
  if (!res.ok) throw new Error(`POST /api/pipeline/runs: ${res.status}`);
  const body = (await res.json()) as { run_id: number };
  return body.run_id;
}

export async function getRun(id: number): Promise<{ run: { id: number; state: string; mode: string } }> {
  const res = await req('GET', `/api/pipeline/runs/${id}`);
  if (!res.ok) throw new Error(`GET /api/pipeline/runs/${id}: ${res.status}`);
  return (await res.json()) as { run: { id: number; state: string; mode: string } };
}

export async function cancelRun(id: number): Promise<void> {
  const res = await req('POST', `/api/pipeline/runs/${id}/cancel`);
  if (!res.ok) throw new Error(`POST /cancel: ${res.status}`);
}

export async function apiHealth(): Promise<boolean> {
  try {
    const res = await req('GET', '/api/health');
    return res.ok;
  } catch {
    return false;
  }
}
