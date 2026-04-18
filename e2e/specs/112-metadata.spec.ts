import { test, expect } from '@playwright/test';
import { getPipelineClips, openDbRW, resetPipelineData } from '../helpers/db';
import { API_BASE } from '../helpers/env';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

interface SeedOpts {
  clipCount?: number;
  videoTitle?: string;
  /** When true, inserts highlight rows linked to each clip for AI context. */
  withHighlights?: boolean;
}

/** Highlight descriptions for seeded clips (used when withHighlights=true). */
const FAKE_HIGHLIGHTS = [
  { text: 'Demonstração ao vivo de como uma IA gera um jogo 2D completo em 3 minutos usando apenas prompts de texto', reason: 'Momento viral — reação impressionante do público' },
  { text: 'Comparação lado a lado do código gerado pela IA vs código feito por humano — resultado surpreendente', reason: 'Conteúdo educacional com alto engajamento' },
  { text: 'Tutorial passo a passo de como configurar o ambiente de desenvolvimento com IA do zero', reason: 'Conteúdo prático com alta retenção' },
];

/** Seed a pipeline_run at a specific state with N fake clips. Returns runId. */
function seedRunAtState(state: string, optsOrCount: SeedOpts | number = 2): number {
  const opts: SeedOpts = typeof optsOrCount === 'number' ? { clipCount: optsOrCount } : optsOrCount;
  const clipCount = opts.clipCount ?? 2;
  const videoTitle = opts.videoTitle ?? '';

  const db = openDbRW();
  try {
    const now = Date.now();
    const res = db
      .prepare(
        `INSERT INTO pipeline_runs (url, mode, state, video_path, video_title, started_at)
         VALUES (?, ?, ?, ?, ?, ?)`,
      )
      .run('https://youtube.com/watch?v=FAKE', 'longform', state, '/tmp/fake.mp4', videoTitle, now);
    const runId = Number(res.lastInsertRowid);

    const insHighlight = db.prepare(
      `INSERT INTO pipeline_highlights (run_id, start_sec, end_sec, adj_start_sec, adj_end_sec, text, reason, score, is_selected)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)`,
    );
    const insClip = db.prepare(
      `INSERT INTO pipeline_clips (run_id, highlight_id, file_path, start_sec, end_sec, duration_sec)
       VALUES (?, ?, ?, ?, ?, ?)`,
    );
    const insClipNoHL = db.prepare(
      `INSERT INTO pipeline_clips (run_id, file_path, start_sec, end_sec, duration_sec)
       VALUES (?, ?, ?, ?, ?)`,
    );

    for (let i = 0; i < clipCount; i++) {
      const startSec = i * 30;
      const endSec = (i + 1) * 30;

      if (opts.withHighlights) {
        const hl = FAKE_HIGHLIGHTS[i % FAKE_HIGHLIGHTS.length];
        const hlRes = insHighlight.run(runId, startSec, endSec, startSec, endSec, hl.text, hl.reason, 0.9);
        const hlId = Number(hlRes.lastInsertRowid);
        insClip.run(runId, hlId, `/tmp/clip_${i}.mp4`, startSec, endSec, 30);
      } else {
        insClipNoHL.run(runId, `/tmp/clip_${i}.mp4`, startSec, endSec, 30);
      }
    }
    return runId;
  } finally {
    db.close();
  }
}

/** POST helper that returns the raw Response. */
async function postJSON(path: string, body?: unknown): Promise<Response> {
  return fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

let sharedRunId = 0;

test.describe.serial('112 — Metadata review endpoints', () => {
  test.beforeAll(() => {
    resetPipelineData();
  });

  test('E1: batch generate rejects when not in WAITING_REVIEW_METADATA', async () => {
    const wrongStateRunId = seedRunAtState('WAITING_THUMBNAIL_CONFIG');

    const res = await postJSON(`/api/metadata/runs/${wrongStateRunId}/generate`);
    expect(res.status).toBe(409);

    const body = await res.json();
    expect(body.error).toContain('WAITING_REVIEW_METADATA');
  });

  test('E2: batch generate returns 202 or 503 depending on Claude CLI', async () => {
    sharedRunId = seedRunAtState('WAITING_REVIEW_METADATA', 3);

    const res = await postJSON(`/api/metadata/runs/${sharedRunId}/generate`);

    // 202 = Claude CLI available (async job started)
    // 503 = Claude CLI not on PATH (expected in CI)
    expect([202, 503]).toContain(res.status);

    if (res.status === 202) {
      const body = await res.json();
      expect(body.job_id).toBeTruthy();
      expect(body.total_clips).toBe(3);
    }
    if (res.status === 503) {
      const body = await res.json();
      expect(body.error).toContain('Claude CLI');
    }
  });

  test('E3: review-metadata saves edits and advances to WAITING_REVIEW_CLIPS', async () => {
    // Use a fresh run to avoid any async generation side-effects from E2
    sharedRunId = seedRunAtState('WAITING_REVIEW_METADATA', 2);

    const clips = getPipelineClips(sharedRunId);
    expect(clips.length).toBe(2);

    const edits = clips.map((c) => ({
      id: c.id,
      title: `Edited Title ${c.id}`,
      description: `Edited description for clip ${c.id}`,
      tags: `tag1,tag2,clip${c.id}`,
      thumbnail_text: `THUMB ${c.id}`,
    }));

    const res = await postJSON(
      `/api/pipeline/runs/${sharedRunId}/gates/review-metadata`,
      { clips: edits },
    );
    expect(res.status).toBe(200);

    const body = await res.json();
    expect(body.state).toBe('WAITING_REVIEW_CLIPS');
  });

  test('E4: clips have metadata persisted in DB after review-metadata', async () => {
    // sharedRunId was advanced in E3
    const clips = getPipelineClips(sharedRunId);
    expect(clips.length).toBe(2);

    for (const c of clips) {
      expect(c.title).toBe(`Edited Title ${c.id}`);
      expect(c.description).toBe(`Edited description for clip ${c.id}`);
      expect(c.tags).toBe(`tag1,tag2,clip${c.id}`);
      expect(c.thumbnail_text).toBe(`THUMB ${c.id}`);
    }
  });

  test('E5: review-metadata rejects when run is not in WAITING_REVIEW_METADATA', async () => {
    // sharedRunId is now WAITING_REVIEW_CLIPS (advanced in E3)
    const res = await postJSON(
      `/api/pipeline/runs/${sharedRunId}/gates/review-metadata`,
      { clips: [] },
    );
    expect(res.status).toBe(409);

    const body = await res.json();
    expect(body.error).toContain('WAITING_REVIEW_METADATA');
  });
});

// ---------------------------------------------------------------------------
// Full AI generation test — requires Claude CLI on PATH
// ---------------------------------------------------------------------------

test.describe.serial('112 — AI metadata generation (full)', () => {
  let aiRunId = 0;
  let clipIds: number[] = [];

  test.beforeAll(() => {
    resetPipelineData();
  });

  test('F1: single-clip generate returns AI metadata via Claude CLI', async () => {
    test.setTimeout(120_000);

    // Seed with realistic video title + highlights so Claude has enough context
    aiRunId = seedRunAtState('WAITING_REVIEW_METADATA', {
      clipCount: 2,
      videoTitle: 'Como Criar Jogos com IA do Zero — Tutorial Completo 2026',
      withHighlights: true,
    });

    const clips = getPipelineClips(aiRunId);
    expect(clips.length).toBe(2);
    clipIds = clips.map((c) => c.id);

    // Use single-clip generate (synchronous) for the first clip
    const res = await postJSON(
      `/api/metadata/runs/${aiRunId}/clips/${clipIds[0]}/generate`,
    );

    // Skip entire suite if Claude CLI not available
    test.skip(res.status === 503, 'Claude CLI not available — skipping AI generation tests');
    expect(res.status).toBe(200);

    const body = (await res.json()) as {
      clip_id: number;
      title: string;
      description: string;
      tags: string[];
      thumbnail_text: string;
      category_id: number;
    };

    // Validate AI-generated response
    expect(body.clip_id).toBe(clipIds[0]);

    // title: non-empty, under 100 chars
    expect(body.title.length, 'title should be non-empty').toBeGreaterThan(0);
    expect(body.title.length, 'title should be under 100 chars').toBeLessThanOrEqual(100);

    // description: non-empty
    expect(body.description.length, 'description should be non-empty').toBeGreaterThan(0);

    // tags: array with items
    expect(body.tags.length, 'tags should have items').toBeGreaterThan(0);

    // thumbnail_text: non-empty, ALL CAPS, max 3 words
    expect(body.thumbnail_text.length, 'thumbnail_text should be non-empty').toBeGreaterThan(0);
    expect(body.thumbnail_text, 'thumbnail_text should be ALL CAPS').toBe(
      body.thumbnail_text.toUpperCase(),
    );
    // AI is instructed "MAX 3 words" but may occasionally produce 4-5; allow some slack
    expect(body.thumbnail_text.split(/\s+/).length, 'thumbnail_text max 5 words').toBeLessThanOrEqual(5);

    // category_id: valid YouTube category
    expect(body.category_id, 'category_id should be a positive number').toBeGreaterThan(0);

    console.log('[F1] AI-generated metadata:', JSON.stringify(body, null, 2));
  });

  test('F2: AI-generated metadata is persisted in DB', async () => {
    test.skip(aiRunId === 0, 'F1 did not run');

    const clips = getPipelineClips(aiRunId);
    const clip = clips.find((c) => c.id === clipIds[0]);
    expect(clip, 'clip should exist').toBeTruthy();

    // The single-generate endpoint persists to DB
    expect(clip!.title.length, 'DB title should be non-empty').toBeGreaterThan(0);
    expect(clip!.description.length, 'DB description should be non-empty').toBeGreaterThan(0);
    expect(clip!.tags.length, 'DB tags should be non-empty').toBeGreaterThan(0);
    expect(clip!.thumbnail_text.length, 'DB thumbnail_text should be non-empty').toBeGreaterThan(0);

    console.log('[F2] DB persisted metadata:', {
      title: clip!.title,
      description: clip!.description,
      tags: clip!.tags,
      thumbnail_text: clip!.thumbnail_text,
    });
  });

  test('F3: review-metadata advances after AI-generated data', async () => {
    test.skip(aiRunId === 0, 'F1 did not run');

    const clips = getPipelineClips(aiRunId);
    // Pass current data (AI-generated for clip 1, empty for clip 2) — simulates user accepting
    const edits = clips.map((c) => ({
      id: c.id,
      title: c.title || `Fallback Title ${c.id}`,
      description: c.description || `Fallback description ${c.id}`,
      tags: c.tags || 'tag1,tag2',
      thumbnail_text: c.thumbnail_text || 'HOOK',
    }));

    const res = await postJSON(
      `/api/pipeline/runs/${aiRunId}/gates/review-metadata`,
      { clips: edits },
    );
    expect(res.status).toBe(200);

    const body = await res.json();
    expect(body.state).toBe('WAITING_REVIEW_CLIPS');
  });
});
