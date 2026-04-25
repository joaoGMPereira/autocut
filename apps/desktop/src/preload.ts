import { contextBridge, ipcRenderer, app } from 'electron';
import type { UpdateProgress } from '@autocut/shared';

contextBridge.exposeInMainWorld('electronAPI', {
  // Platform info
  platform: process.platform,
  arch: process.arch,
  appVersion: app.getVersion(),

  // Auto-update
  checkForUpdates: () => ipcRenderer.invoke('check-for-updates'),
  updateAndRestart: () => ipcRenderer.invoke('update-and-restart'),
  getUpdateStatus: (): Promise<UpdateProgress> => ipcRenderer.invoke('get-update-status'),
  onUpdateStatus: (cb: (status: UpdateProgress) => void) => {
    ipcRenderer.on('update-status', (_e, status) => cb(status));
  },
  setAlphaChannel: (enabled: boolean) => ipcRenderer.invoke('set-alpha-channel', enabled),
  getAlphaChannel: (): Promise<boolean> => ipcRenderer.invoke('get-alpha-channel'),

  // Native dialogs
  selectDirectory: (): Promise<string | null> => ipcRenderer.invoke('select-directory'),
  selectFile: (): Promise<string | null> => ipcRenderer.invoke('select-file'),

  // Services
  restartGoServer: () => ipcRenderer.invoke('restart-go-server'),
});
