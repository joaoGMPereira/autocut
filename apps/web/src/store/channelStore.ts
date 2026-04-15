import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('channelStore');

export interface Channel {
  ID: number;
  Name: string;
  ChannelID: string;        // YouTube channel ID (backend field name)
  YouTubeChannelID?: string; // Kept for backward compat (not returned by backend)
  ChannelTitle: string;
  AvatarURL: string;
  AccessToken: string;
  RefreshToken: string;
  ExpiresAt: number;
  OAuthClientSecretID: { Int64: number; Valid: boolean } | null;
  IsFavorite: boolean;
  CreatedAt: number;
  UpdatedAt: number;
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
  getChannel: (goUrl: string, id: number) => Promise<Channel>;
  addChannel: (goUrl: string, payload: { name: string; youtube_channel_id: string }) => Promise<void>;
  removeChannel: (goUrl: string, id: number) => Promise<void>;
  selectChannel: (id: number | null) => void;
  loadChannelConfig: (goUrl: string, channelId: number) => Promise<void>;
  refreshConnectedChannels: (goUrl: string) => Promise<void>;
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

  getChannel: async (goUrl, id) => {
    const res = await fetch(`${goUrl}/api/channels/${id}`);
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    const data = (await res.json()) as Channel;
    log.info('channel fetched', { id });
    return data;
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

  refreshConnectedChannels: async (goUrl) => {
    const channels = get().channels.length > 0
      ? get().channels
      : await (async () => {
          const res = await fetch(`${goUrl}/api/channels`);
          if (!res.ok) return [] as Channel[];
          return (await res.json()) as Channel[];
        })();

    const connected = channels.filter((ch) => !!ch.RefreshToken);
    if (connected.length === 0) return;

    log.info('refreshing tokens', { count: connected.length });
    await Promise.allSettled(
      connected.map((ch) =>
        fetch(`${goUrl}/api/channels/${ch.ID}/auth/refresh`, { method: 'POST' })
          .then((r) => { if (!r.ok) throw new Error(`${r.status}`); })
          .then(() => log.info('token refreshed', { id: ch.ID }))
          .catch((err) => log.warn('token refresh failed', { id: ch.ID, err: String(err) }))
      )
    );
    await get().fetchChannels(goUrl);
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
