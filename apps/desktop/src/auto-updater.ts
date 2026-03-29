import { autoUpdater } from 'electron-updater';
import { BrowserWindow } from 'electron';
import type { UpdateProgress } from '@autocut/shared';

interface AutoUpdaterDeps {
  getMainWindow: () => BrowserWindow | null;
  onStatusChange: (status: UpdateProgress) => void;
}

export function setupAutoUpdater(deps: AutoUpdaterDeps): void {
  autoUpdater.autoDownload = true;
  autoUpdater.autoInstallOnAppQuit = true;
  autoUpdater.logger = null;

  autoUpdater.on('checking-for-update', () => {
    deps.onStatusChange({ status: 'checking' });
  });

  autoUpdater.on('update-available', (info) => {
    deps.onStatusChange({
      status: 'available',
      updateInfo: {
        version: info.version,
        releaseDate: info.releaseDate,
        releaseNotes: typeof info.releaseNotes === 'string' ? info.releaseNotes : undefined,
      },
    });
  });

  autoUpdater.on('update-not-available', () => {
    deps.onStatusChange({ status: 'not-available' });
  });

  autoUpdater.on('download-progress', (progress) => {
    deps.onStatusChange({
      status: 'downloading',
      percent: progress.percent,
      bytesPerSecond: progress.bytesPerSecond,
      transferred: progress.transferred,
      total: progress.total,
    });
  });

  autoUpdater.on('update-downloaded', (info) => {
    deps.onStatusChange({
      status: 'downloaded',
      updateInfo: {
        version: info.version,
        releaseDate: info.releaseDate,
        releaseNotes: typeof info.releaseNotes === 'string' ? info.releaseNotes : undefined,
      },
    });
  });

  autoUpdater.on('error', (err) => {
    deps.onStatusChange({ status: 'error', error: err.message });
  });

  // IPC handlers for update actions
  const { ipcMain } = require('electron');
  ipcMain.handle('check-for-updates', async () => {
    await autoUpdater.checkForUpdates();
  });
  ipcMain.handle('update-and-restart', () => {
    autoUpdater.quitAndInstall();
  });

  // Check after 10 seconds, then every 30 minutes
  setTimeout(() => autoUpdater.checkForUpdates(), 10_000);
  setInterval(() => autoUpdater.checkForUpdates(), 30 * 60 * 1000);
}
