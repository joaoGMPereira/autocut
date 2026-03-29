import { ipcMain, dialog, BrowserWindow } from 'electron';
import { ChildProcess } from 'child_process';
import type { UpdateProgress } from '@autocut/shared';

export interface IpcDeps {
  getMainWindow: () => BrowserWindow | null;
  getGoProcess: () => ChildProcess | null;
  getCurrentUpdateStatus: () => UpdateProgress;
  isDev: boolean;
  goUrl: string;
}

export function registerIpcHandlers(deps: IpcDeps): void {
  // Auto-update
  ipcMain.handle('get-update-status', () => deps.getCurrentUpdateStatus());

  // Alpha channel preference (stored in-memory for simplicity at this stage)
  let alphaEnabled = false;
  ipcMain.handle('set-alpha-channel', (_e, enabled: boolean) => {
    alphaEnabled = enabled;
  });
  ipcMain.handle('get-alpha-channel', () => alphaEnabled);

  // Native dialog
  ipcMain.handle('select-directory', async () => {
    const win = deps.getMainWindow();
    const result = await dialog.showOpenDialog(win ?? new BrowserWindow({ show: false }), {
      properties: ['openDirectory'],
    });
    return result.canceled ? null : result.filePaths[0];
  });

  // Service control
  ipcMain.handle('restart-go-server', () => {
    const proc = deps.getGoProcess();
    if (proc && !proc.killed) {
      proc.kill('SIGTERM');
    }
    // main.ts crash handler will auto-restart
  });
}
