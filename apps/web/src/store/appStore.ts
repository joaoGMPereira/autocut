import { create } from 'zustand';
import type { UpdateProgress } from '@autocut/shared';

interface AppState {
  updateStatus: UpdateProgress;
  goUrl: string;
  setUpdateStatus: (status: UpdateProgress) => void;
  setGoUrl: (url: string) => void;
}

export const useAppStore = create<AppState>((set) => ({
  updateStatus: { status: 'idle' },
  goUrl: process.env.NEXT_PUBLIC_GO_URL ?? 'http://127.0.0.1:4070',
  setUpdateStatus: (status) => set({ updateStatus: status }),
  setGoUrl: (goUrl) => set({ goUrl }),
}));
