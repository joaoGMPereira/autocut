import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { createRun, getRun } from '../helpers/api';
import { advanceGate, assertClipFilesExist, assertVideoDownloaded, waitForState } from '../helpers/pipeline';
import { getPipelineClips, getPipelineRun, resetPipelineData } from '../helpers/db';
import { seedFakeWhisperModel, systemHas } from '../helpers/setup';
import { API_BASE, TEST_DATA_DIR } from '../helpers/env';

const FIXTURE_URL = 'https://www.youtube.com/watch?v=bfJy1-IRa_k';
let sharedRunId = 0;

test.describe.serial('Pipeline Longform: URL → clips → thumbnail (real)', () => {
  test.skip(process.env.E2E_NETWORK !== 'true', 'requires network (yt-dlp) + FFmpeg on PATH');

  test.beforeAll(() => {
    resetPipelineData();
    seedFakeWhisperModel();
  });

  test('C1: completes longform pipeline end-to-end and produces files on disk', async () => {
    test.setTimeout(1_800_000); // 30 min: yt-dlp + FFmpeg (download + raw cut + effects per clip)

    test.skip(!systemHas('yt-dlp'), 'yt-dlp missing from PATH');
    test.skip(!systemHas('ffmpeg'), 'ffmpeg missing from PATH');

    const runId = await createRun();
    sharedRunId = runId;

    // 1. Submit URL — state → WAITING_MODE; download runs async
    const urlResult = await advanceGate(runId, { url: FIXTURE_URL });
    expect(urlResult.state).toBe('WAITING_MODE');

    // 2. Wait for video_path + duration populated (download done)
    await expect
      .poll(
        () => {
          const row = getPipelineRun(runId);
          return row?.video_path && (row?.duration_sec ?? 0) > 0 ? 'done' : 'pending';
        },
        { timeout: 600_000, intervals: [1000, 2000] },
      )
      .toBe('done');
    assertVideoDownloaded(runId);

    // 3. Submit longform mode — state → EXECUTING → GENERATING_CLIPS (auto for longform)
    const modeResult = await advanceGate(runId, {
      mode_config: { mode: 'longform', segment_secs: 30 },
    });
    expect(['EXECUTING', 'GENERATING_CLIPS']).toContain(modeResult.state);

    // 4. Wait for clip cutting to complete → WAITING_THUMBNAIL_CONFIG
    //    Effects pass runs per-clip and can be slow; allow up to ~20 min.
    const finalState = await waitForState(runId, 'WAITING_THUMBNAIL_CONFIG', 1_200_000);
    expect(finalState).toBe('WAITING_THUMBNAIL_CONFIG');

    // 5. Disk assertions
    const run = getPipelineRun(runId)!;
    expect(run.video_title).toBeTruthy();
    expect(run.duration_sec).toBeGreaterThan(0);

    const clipPaths = assertClipFilesExist(runId);
    expect(clipPaths.length).toBeGreaterThan(0);

    // Clip dir is under {dataDir}/clips/*
    const clipsRoot = path.join(TEST_DATA_DIR, 'clips');
    expect(fs.existsSync(clipsRoot)).toBe(true);
    const clipDirs = fs.readdirSync(clipsRoot).filter((d) => d.includes('bfJy1-IRa_k'));
    expect(clipDirs.length).toBeGreaterThan(0);

    // At least one clip row per segment; all file_paths point to real files
    const clips = getPipelineClips(runId);
    for (const c of clips) {
      expect(c.file_path, `clip ${c.id} should have a file_path`).toBeTruthy();
      expect(fs.existsSync(c.file_path), `clip ${c.id} file should exist on disk`).toBe(true);
    }

    // Sanity on API
    const { run: apiRun } = await getRun(runId);
    expect(apiRun.state).toBe('WAITING_THUMBNAIL_CONFIG');
    expect(apiRun.mode).toBe('longform');
  });

  test('D1: generates a thumbnail for the first clip and persists to DB + disk', async () => {
    test.skip(!systemHas('magick') && !systemHas('convert'), 'ImageMagick (magick/convert) not on PATH');
    test.skip(sharedRunId === 0, 'C1 did not run');

    const clips = getPipelineClips(sharedRunId);
    expect(clips.length).toBeGreaterThan(0);
    const firstClip = clips[0];

    const res = await fetch(
      `${API_BASE}/api/thumbnail/runs/${sharedRunId}/clips/${firstClip.id}/generate`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          config: {
            text: 'TEST',
            font_family: 'Impact',
            text_color: '#FFD700',
            outline_color: '#000000',
            stroke_width: 8,
            blur_amount: 25,
            darken_amount: 0.3,
            center_scale: 0.75,
            corner_radius: 40,
            text_y_offset: 0,
          },
          landscape_config: {
            border_size: 12,
            corner_radius: 40,
            stroke_width: 8,
            gradient_start: '#FF00AA',
            gradient_end: '#00AAFF',
            text_color: '#FFD700',
            outline_color: '#000000',
          },
        }),
      },
    );
    expect(res.ok, `thumbnail generate status ${res.status}: ${await res.clone().text()}`).toBe(true);

    const body = (await res.json()) as { clip_id: number; thumbnail_path: string };
    expect(body.thumbnail_path).toBeTruthy();
    expect(fs.existsSync(body.thumbnail_path)).toBe(true);

    // DB reflects the result
    const after = getPipelineClips(sharedRunId).find((c) => c.id === firstClip.id);
    expect(after?.thumbnail_path).toBe(body.thumbnail_path);
  });
});
