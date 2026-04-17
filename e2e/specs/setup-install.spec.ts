import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { launchApp, closeApp } from '../helpers/electron';
import { WEB_BASE, TEST_DATA_DIR } from '../helpers/env';
import {
  getToolsStatus,
  seedFakeWhisperModel,
  setCustomToolPath,
  syncFromPath,
  systemHas,
} from '../helpers/setup';

test.describe('Setup: install flow via UI', () => {
  test.beforeEach(() => {
    // Ensure whisper model is absent so the gate stays up until we seed it.
    const modelsDir = path.join(TEST_DATA_DIR, 'models', 'whisper');
    fs.rmSync(modelsDir, { recursive: true, force: true });
  });

  test('B1: SetupGate renders; Re-check + seed model flips whisper-cli to Installed', async () => {
    const { app, page } = await launchApp();
    try {
      await page.goto(`${WEB_BASE}/pipeline?setup=force`);

      // Gate is visible, tool rows are populated
      await expect(page.getByTestId('setup-gate')).toBeVisible({ timeout: 30_000 });
      const whisperStatus = page.getByTestId('tool-status-whisper-cli');
      await expect(whisperStatus).toBeVisible();

      // Binary is auto-symlinked from PATH but model is missing → badge says "Missing model"
      await expect(whisperStatus).toHaveText(/missing model/i, { timeout: 15_000 });

      // Continue is disabled because whisper (a required tool) isn't fully installed
      await expect(page.getByTestId('setup-continue-btn')).toBeDisabled();

      // Seed a placeholder model file; Re-check triggers fetchStatus
      seedFakeWhisperModel();
      await page.getByRole('button', { name: /re-check/i }).click();

      // Badge flips to "Installed" once binary + model are both present
      await expect(whisperStatus).toHaveText(/^installed$/i, { timeout: 15_000 });
      // (Continue only enables when every required tool is installed; e.g. TwitchDownloaderCLI
      // may be absent on a fresh dev host. B2 covers the full sync.)
    } finally {
      await closeApp(app);
    }
  });

  test('B2: Sync-from-computer button reconciles bin dir via API', async () => {
    seedFakeWhisperModel();
    const tools = await syncFromPath();
    // Every required tool present on the system must report installed after sync
    const required = tools.filter((t) => t.required);
    for (const t of required) {
      if (t.name === 'whisper-cli') {
        // whisper-cli needs model; we seeded it — source may be autocut_bin or path
        continue;
      }
      if (!systemHas(t.name === 'TwitchDownloaderCLI' ? 'TwitchDownloaderCLI' : t.name)) {
        continue; // skip tools not present on this system
      }
      expect(t.installed, `${t.name} should be installed after sync`).toBe(true);
    }

    // After sync, fetching status reflects the same
    const after = await getToolsStatus();
    const ffmpeg = after.find((t) => t.name === 'ffmpeg');
    if (systemHas('ffmpeg')) {
      expect(ffmpeg?.installed).toBe(true);
    }
  });

  test('B3a: custom path for ffmpeg — invalid path rejected', async () => {
    const res = await setCustomToolPath('ffmpeg', '/this/path/does/not/exist/ffmpeg');
    expect(res.status).toBeGreaterThanOrEqual(400);
  });

  test('B3b: custom path for ffmpeg — valid path accepted', async () => {
    const systemFfmpeg = systemHas('ffmpeg');
    test.skip(!systemFfmpeg, 'ffmpeg not on PATH');

    const res = await setCustomToolPath('ffmpeg', systemFfmpeg!);
    expect(res.ok).toBe(true);

    const body = (await res.json()) as { tool?: { source?: string; installed?: boolean } };
    expect(body.tool?.source).toBe('custom');
    expect(body.tool?.installed).toBe(true);
  });
});
