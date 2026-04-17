import { _electron as electron, ElectronApplication, Page } from '@playwright/test';
import path from 'node:path';
import { TEST_API_PORT, TEST_WEB_PORT, WEB_BASE } from './env';

const REPO_ROOT = path.resolve(__dirname, '..', '..');
const DESKTOP_DIR = path.join(REPO_ROOT, 'apps', 'desktop');
const ELECTRON_ENTRY = path.join(DESKTOP_DIR, 'compiled', 'main.js');

export async function launchApp(): Promise<{ app: ElectronApplication; page: Page }> {
  const app = await electron.launch({
    args: [ELECTRON_ENTRY],
    cwd: DESKTOP_DIR,
    env: {
      ...process.env,
      AUTOCUT_TEST_MODE: 'true',
      AUTOCUT_API_PORT: String(TEST_API_PORT),
      AUTOCUT_WEB_PORT: String(TEST_WEB_PORT),
      AUTOCUT_SKIP_NEXT: 'true',
      NODE_ENV: 'development',
    },
    timeout: 60_000,
  });
  const page = await app.firstWindow();
  // Wait for the Next.js URL to load (the main window's loadURL points at WEB_BASE).
  await page.waitForURL(url => url.toString().startsWith(WEB_BASE), { timeout: 60_000 });
  return { app, page };
}

export async function closeApp(app: ElectronApplication): Promise<void> {
  try {
    await app.close();
  } catch {
    /* ignore */
  }
}
