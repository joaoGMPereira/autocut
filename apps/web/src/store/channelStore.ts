import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('channelStore');

export interface Channel {
  ID: number;
  Name: string;
  YouTubeChannelID: string;
  avatar_url?: string;
  subscriber_count?: number;
}

export interface ChannelConfig {
  id: number;
  channel_id: number;
  output_dir: string;
  default_quality: string;
  auto_upload: boolean;
  default_privacy: string;
}

interface ChannelState {
  channels: Channel[];
  loading: boolean;
  error: string | null;
  selectedChannelId: number | null;
  channelConfig: ChannelConfig | null;
  fetchChannels: (goUrl: string) => Promise<void>;
  addChannel: (goUrl: string, payload: { name: string; youtube_channel_id: string }) => Promise<void>;
  removeChannel: (goUrl: string, id: number) => Promise<void>;
  selectChannel: (id: number | null) => void;
  loadChannelConfig: (goUrl: string, channelId: number) => Promise<void>;
}

export const useChannelStore = create<ChannelState>((set, get) => ({
  channels: [],
  loading: false,
  error: null,
  selectedChannelId: null,
  channelConfig: null,

  fetchChannels: async (goUrl) => {
    set({ loading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/channels`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as Channel[];
      log.info('channels fetched', { count: data.length });
      set({ channels: data, loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch channels failed', { err: message });
      set({ loading: false, error: message });
    }
  },

  addChannel: async (goUrl, payload) => {
    const res = await fetch(`${goUrl}/api/channels`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    log.info('channel added');
    await get().fetchChannels(goUrl);
  },

  removeChannel: async (goUrl, id) => {
    const res = await fetch(`${goUrl}/api/channels/${id}`, { method: 'DELETE' });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    log.info('channel removed', { id });
    await get().fetchChannels(goUrl);
  },

  selectChannel: (id) => {
    set({ selectedChannelId: id });
  },

  loadChannelConfig: async (goUrl, channelId) => {
    try {
      const res = await fetch(`${goUrl}/api/channels/${channelId}/config`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const data = (await res.json()) as ChannelConfig;
      log.info('channel config loaded', { channelId });
      set({ channelConfig: data });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('load channel config failed', { err: message });
      set({ channelConfig: null });
    }
  },
}));
