import { app, BrowserWindow, shell } from 'electron';
import { spawn, ChildProcess } from 'child_process';
import path from 'path';
import http from 'http';
import { registerIpcHandlers } from './ipc-handlers';
import { setupAutoUpdater } from './auto-updater';
import type { UpdateProgress } from '@autocut/shared';

// ─── Constants ────────────────────────────────────────────────────────────────

const IS_DEV = !app.isPackaged;
const GO_PORT = IS_DEV ? 4071 : 4070;
const WEB_PORT = IS_DEV ? 3201 : 3200;
const GO_URL = `http://127.0.0.1:${GO_PORT}`;
const WEB_URL = `http://127.0.0.1:${WEB_PORT}`;
const MAX_RESTARTS = 3;

// ─── State ────────────────────────────────────────────────────────────────────

let mainWindow: BrowserWindow | null = null;
let goProcess: ChildProcess | null = null;
let nextProcess: ChildProcess | null = null;
let goRestartCount = 0;
let currentUpdateStatus: UpdateProgress = { status: 'idle' };

// ─── Path helpers ─────────────────────────────────────────────────────────────

function getGoBinPath(): string {
  if (IS_DEV) {
    return path.join(__dirname, '..', 'bin', 'server');
  }
  const ext = process.platform === 'win32' ? '.exe' : '';
  return path.join(process.resourcesPath, `server${ext}`);
}

function getAppDataDir(): string {
  const name = IS_DEV ? '.autocut-dev' : '.autocut';
  return path.join(app.getPath('home'), name);
}

// ─── Service wait ─────────────────────────────────────────────────────────────

function waitForService(url: string, timeoutMs = 30000): Promise<void> {
  return new Promise((resolve, reject) => {
    const start = Date.now();
    function check() {
      http.get(`${url}/health`, (res) => {
        if (res.statusCode === 200) return resolve();
        retry();
      }).on('error', retry);
    }
    function retry() {
      if (Date.now() - start > timeoutMs) {
        return reject(new Error(`Timeout waiting for ${url}`));
      }
      setTimeout(check, 500);
    }
    check();
  });
}

// ─── Go server ────────────────────────────────────────────────────────────────

function startGoServer(): void {
  const bin = getGoBinPath();
  const dir = getAppDataDir();

  goProcess = spawn(bin, [
    '-host', '127.0.0.1',
    '-port', String(GO_PORT),
    '-dir', dir,
  ], {
    detached: false,
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  goProcess.stdout?.on('data', (d: Buffer) => process.stdout.write(`[go] ${d}`));
  goProcess.stderr?.on('data', (d: Buffer) => process.stderr.write(`[go] ${d}`));

  goProcess.on('exit', (code) => {
    if (code !== 0 && goRestartCount < MAX_RESTARTS) {
      goRestartCount++;
      const delay = 1500 * Math.pow(2, goRestartCount - 1);
      console.log(`[go] crashed (code ${code}), restarting in ${delay}ms...`);
      setTimeout(startGoServer, delay);
    }
  });
}

function killGoServer(): void {
  if (goProcess && !goProcess.killed) {
    goProcess.kill('SIGTERM');
    goProcess = null;
  }
}

// ─── Next.js server ───────────────────────────────────────────────────────────

function startNextProcess(): void {
  if (!IS_DEV) return; // prod: standalone server started separately

  nextProcess = spawn('pnpm', ['--filter', '@autocut/web', 'dev', '--port', String(WEB_PORT)], {
    cwd: path.join(__dirname, '..', '..', '..'),
    detached: false,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: {
      ...process.env,
      NEXT_PUBLIC_GO_URL: GO_URL,
    },
  });

  nextProcess.stdout?.on('data', (d: Buffer) => process.stdout.write(`[next] ${d}`));
  nextProcess.stderr?.on('data', (d: Buffer) => process.stderr.write(`[next] ${d}`));
}

function killNextProcess(): void {
  if (nextProcess && !nextProcess.killed) {
    nextProcess.kill('SIGTERM');
    nextProcess = null;
  }
}

// ─── Window ───────────────────────────────────────────────────────────────────

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    show: false,
    backgroundColor: '#09090b', // zinc-950
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  });

  mainWindow.loadURL(WEB_URL);

  mainWindow.once('ready-to-show', () => {
    mainWindow?.show();
  });

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// ─── App lifecycle ────────────────────────────────────────────────────────────

app.whenReady().then(async () => {
  startGoServer();
  startNextProcess();

  try {
    await waitForService(GO_URL);
    if (IS_DEV) await waitForService(WEB_URL);
  } catch (e) {
    console.error('Services failed to start:', e);
  }

  createWindow();

  registerIpcHandlers({
    getMainWindow: () => mainWindow,
    getGoProcess: () => goProcess,
    getCurrentUpdateStatus: () => currentUpdateStatus,
    isDev: IS_DEV,
    goUrl: GO_URL,
  });

  setupAutoUpdater({
    getMainWindow: () => mainWindow,
    onStatusChange: (status) => {
      currentUpdateStatus = status;
      mainWindow?.webContents.send('update-status', status);
    },
  });
});

app.on('window-all-closed', () => {
  killGoServer();
  killNextProcess();
  if (process.platform !== 'darwin') app.quit();
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) createWindow();
});

app.on('before-quit', () => {
  killGoServer();
  killNextProcess();
});
