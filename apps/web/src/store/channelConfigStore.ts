import { create } from 'zustand';
import { createLogger } from '@/lib/logger';
import { useAppStore } from '@/store/appStore';

const log = createLogger('channelConfigStore');

export interface ChannelConfig {
  id?: number;
  channel_id: number;
  gradient_color_start: string;
  gradient_color_end: string;
  thumbnail_font_family: string;
  thumbnail_text_color_1: string;
  default_category_id: number;
  default_tags: string; // comma-separated
  made_for_kids: boolean;
  default_playlist_id_cortes?: string | null;
  default_playlist_id_shorts?: string | null;
  shorts_title_hashtags?: string;
  video_description_hashtags?: string;
  anti_duplicate_enabled: boolean;
  anti_duplicate_mode?: string;
  speed_enabled?: boolean;
  speed_factor?: number;
  visual_color_grading_enabled?: boolean;
  visual_crop_enabled?: boolean;
  visual_zoom_enabled?: boolean;
  branding_logo_enabled: boolean;
  branding_logo_path?: string;
  branding_intro_enabled?: boolean;
  branding_outro_enabled?: boolean;
  audio_music_enabled?: boolean;
  audio_music_volume?: number;
  preview_captions_enabled?: boolean;
  preview_caption_style?: string;
  pinned_comment_template?: string;
  max_highlights?: number;
}

export interface MusicBlacklistItem {
  id: number;
  channel_id: number;
  pattern: string;
  created_at: string;
}

interface ChannelConfigStore {
  configs: Record<number, ChannelConfig>;
  blacklists: Record<number, MusicBlacklistItem[]>;
  fetchConfig: (channelId: number) => Promise<void>;
  saveConfig: (channelId: number, config: Partial<ChannelConfig>) => Promise<void>;
  uploadLogo: (channelId: number, file: File) => Promise<void>;
  fetchBlacklist: (channelId: number) => Promise<void>;
  addBlacklistPattern: (channelId: number, pattern: string) => Promise<void>;
  removeBlacklistPattern: (channelId: number, patternId: number) => Promise<void>;
}

export const useChannelConfigStore = create<ChannelConfigStore>((set, get) => ({
  configs: {},
  blacklists: {},

  fetchConfig: async (channelId) => {
    const { goUrl } = useAppStore.getState();
    try {
      const res = await fetch(`${goUrl}/api/channels/${channelId}/config`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as ChannelConfig;
      log.info('channel config fetched', { channelId });
      set((state) => ({
        configs: { ...state.configs, [channelId]: data },
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch channel config failed', { channelId, err: message });
    }
  },

  saveConfig: async (channelId, config) => {
    const { goUrl } = useAppStore.getState();
    const existing = get().configs[channelId] ?? {};
    const merged = { ...existing, ...config, channel_id: channelId };
    try {
      const res = await fetch(`${goUrl}/api/channels/${channelId}/config`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(merged),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const updated = (await res.json()) as ChannelConfig;
      log.info('channel config saved', { channelId });
      set((state) => ({
        configs: { ...state.configs, [channelId]: updated },
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('save channel config failed', { channelId, err: message });
      throw err;
    }
  },

  uploadLogo: async (channelId, file) => {
    const { goUrl } = useAppStore.getState();
    try {
      const form = new FormData();
      form.append('logo', file);
      const res = await fetch(`${goUrl}/api/channels/${channelId}/config/logo`, {
        method: 'POST',
        body: form,
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const updated = (await res.json()) as ChannelConfig;
      log.info('logo uploaded', { channelId });
      set((state) => ({
        configs: { ...state.configs, [channelId]: updated },
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('upload logo failed', { channelId, err: message });
      throw err;
    }
  },

  fetchBlacklist: async (channelId) => {
    const { goUrl } = useAppStore.getState();
    try {
      const res = await fetch(`${goUrl}/api/channels/${channelId}/music-blacklist`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as MusicBlacklistItem[];
      log.info('music blacklist fetched', { channelId, count: data.length });
      set((state) => ({
        blacklists: { ...state.blacklists, [channelId]: data },
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch blacklist failed', { channelId, err: message });
    }
  },

  addBlacklistPattern: async (channelId, pattern) => {
    const { goUrl } = useAppStore.getState();
    try {
      const res = await fetch(`${goUrl}/api/channels/${channelId}/music-blacklist`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pattern }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('blacklist pattern added', { channelId, pattern });
      await get().fetchBlacklist(channelId);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('add blacklist pattern failed', { channelId, err: message });
      throw err;
    }
  },

  removeBlacklistPattern: async (channelId, patternId) => {
    const { goUrl } = useAppStore.getState();
    try {
      const res = await fetch(
        `${goUrl}/api/channels/${channelId}/music-blacklist/${patternId}`,
        { method: 'DELETE' }
      );
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      log.info('blacklist pattern removed', { channelId, patternId });
      set((state) => ({
        blacklists: {
          ...state.blacklists,
          [channelId]: (state.blacklists[channelId] ?? []).filter(
            (item) => item.id !== patternId
          ),
        },
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('remove blacklist pattern failed', { channelId, err: message });
      throw err;
    }
  },
}));
