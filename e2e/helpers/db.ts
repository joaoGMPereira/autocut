import Database from 'better-sqlite3';
import { TEST_DB_PATH } from './env';

export function openDb(): Database.Database {
  // read-only is fine; the Go server owns writes
  return new Database(TEST_DB_PATH, { readonly: true, fileMustExist: true });
}

export function openDbRW(): Database.Database {
  return new Database(TEST_DB_PATH);
}

export function getSetting(key: string): string | null {
  const db = openDb();
  try {
    const row = db
      .prepare<[string], { value: string }>('SELECT value FROM app_settings WHERE key = ?')
      .get(key);
    return row?.value ?? null;
  } finally {
    db.close();
  }
}

export function countPipelineRuns(): number {
  const db = openDb();
  try {
    const row = db
      .prepare<[], { n: number }>('SELECT COUNT(*) AS n FROM pipeline_runs')
      .get();
    return row?.n ?? 0;
  } finally {
    db.close();
  }
}

export function getPipelineRun(id: number):
  | {
      id: number;
      state: string;
      mode: string;
      url: string;
      video_path: string;
      video_title: string | null;
      duration_sec: number | null;
    }
  | null {
  const db = openDb();
  try {
    const row = db
      .prepare<
        [number],
        {
          id: number;
          state: string;
          mode: string;
          url: string;
          video_path: string;
          video_title: string | null;
          duration_sec: number | null;
        }
      >(
        'SELECT id, state, mode, url, video_path, video_title, duration_sec FROM pipeline_runs WHERE id = ?',
      )
      .get(id);
    return row ?? null;
  } finally {
    db.close();
  }
}

export function getPipelineClips(runId: number):
  | Array<{
      id: number;
      run_id: number;
      thumbnail_style: string;
      thumbnail_path: string;
      file_path: string;
    }> {
  const db = openDb();
  try {
    return db
      .prepare<
        [number],
        {
          id: number;
          run_id: number;
          thumbnail_style: string;
          thumbnail_path: string;
          file_path: string;
        }
      >(
        'SELECT id, run_id, thumbnail_style, thumbnail_path, file_path FROM pipeline_clips WHERE run_id = ? ORDER BY id',
      )
      .all(runId);
  } finally {
    db.close();
  }
}

/**
 * Wipe pipeline runs + clips + highlights between tests that need a clean slate.
 * Do NOT touch app_settings (tests may rely on seeded values).
 */
export function resetPipelineData(): void {
  const db = openDbRW();
  try {
    db.exec('DELETE FROM pipeline_clips');
    db.exec('DELETE FROM pipeline_highlights');
    db.exec('DELETE FROM pipeline_runs');
  } finally {
    db.close();
  }
}

/**
 * Seed a minimal authenticated channel row so UI pipeline flows can proceed.
 * YouTube OAuth is not feasible in E2E — fake the token fields.
 */
export function seedFakeChannel(name = 'E2E Test Channel'): number {
  const db = openDbRW();
  try {
    const now = Date.now();
    const res = db
      .prepare(
        `INSERT INTO channels (Name, YouTubeChannelID, IsFavorite, AccessToken, RefreshToken, TokenExpiresAt, CreatedAt, UpdatedAt)
         VALUES (?, ?, 1, ?, ?, ?, ?, ?)`,
      )
      .run(name, 'UCtest1234567890', 'fake-access', 'fake-refresh', now + 3_600_000, now, now);
    return Number(res.lastInsertRowid);
  } finally {
    db.close();
  }
}

export function deleteAllSettings(): void {
  const db = openDbRW();
  try {
    db.exec('DELETE FROM app_settings');
  } finally {
    db.close();
  }
}
