import { create } from 'zustand';
import { createLogger } from '@/lib/logger';
import { useAppStore } from '@/store/appStore';

const log = createLogger('thumbnailStore');

export interface ThumbnailBackground {
  id: number;
  channel_id: number;
  name: string;
  file_path: string;
  is_default: boolean;
  created_at: number;
}

interface ThumbnailStore {
  backgrounds: ThumbnailBackground[];
  selectedBgId: number | null;
  loadingBackgrounds: boolean;
  fetchBackgrounds: (channelId: number) => Promise<void>;
  uploadBackground: (channelId: number, file: File, name: string) => Promise<void>;
  deleteBackground: (channelId: number, bgId: number) => Promise<void>;
  selectBackground: (id: number | null) => void;
}

export const useThumbnailStore = create<ThumbnailStore>((set) => ({
  backgrounds: [],
  selectedBgId: null,
  loadingBackgrounds: false,

  fetchBackgrounds: async (channelId) => {
    const { goUrl } = useAppStore.getState();
    set({ loadingBackgrounds: true });
    try {
      const res = await fetch(`${goUrl}/api/channels/${channelId}/backgrounds`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as ThumbnailBackground[];
      log.info('backgrounds fetched', { channelId, count: data.length });
      set({ backgrounds: data, loadingBackgrounds: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch backgrounds failed', { err: message });
      set({ loadingBackgrounds: false });
    }
  },

  uploadBackground: async (channelId, file, name) => {
    const { goUrl } = useAppStore.getState();
    const form = new FormData();
    form.append('file', file);
    form.append('name', name);
    const res = await fetch(`${goUrl}/api/channels/${channelId}/backgrounds`, {
      method: 'POST',
      body: form,
    });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    log.info('background uploaded', { channelId, name });
    const { fetchBackgrounds } = useThumbnailStore.getState();
    await fetchBackgrounds(channelId);
  },

  deleteBackground: async (channelId, bgId) => {
    const { goUrl } = useAppStore.getState();
    const res = await fetch(`${goUrl}/api/channels/${channelId}/backgrounds/${bgId}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    log.info('background deleted', { channelId, bgId });
    set((state) => ({
      backgrounds: state.backgrounds.filter((bg) => bg.id !== bgId),
      selectedBgId: state.selectedBgId === bgId ? null : state.selectedBgId,
    }));
  },

  selectBackground: (id) => {
    set({ selectedBgId: id });
  },
}));
