import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('oauthStore');

export interface OAuthProfile {
  id: number;
  name: string;
  project_id: string;
  is_default: boolean;
  has_credentials: boolean;
}

interface OAuthStore {
  profiles: OAuthProfile[];
  loading: boolean;
  fetchProfiles: (goUrl: string) => Promise<void>;
  uploadProfile: (goUrl: string, name: string, file: File) => Promise<void>;
  patchProfile: (goUrl: string, id: number, file: File) => Promise<void>;
  deleteProfile: (goUrl: string, id: number) => Promise<void>;
  setDefault: (goUrl: string, id: number) => Promise<void>;
  initOAuth: (goUrl: string, channelId: number) => Promise<{ auth_url: string }>;
  submitCode: (goUrl: string, channelId: number, code: string) => Promise<void>;
}

export const useOAuthStore = create<OAuthStore>((set) => ({
  profiles: [],
  loading: false,

  fetchProfiles: async (goUrl) => {
    set({ loading: true });
    try {
      const res = await fetch(`${goUrl}/api/oauth/profiles`);
      if (!res.ok) throw new Error(`fetch profiles failed: ${res.status}`);
      const data = (await res.json()) as { profiles: OAuthProfile[] };
      log.info('oauth profiles fetched', { count: data.profiles?.length ?? 0 });
      set({ profiles: data.profiles ?? [], loading: false });
    } catch (err) {
      log.error('fetchProfiles failed', { err });
      set({ loading: false });
    }
  },

  uploadProfile: async (goUrl, name, file) => {
    const form = new FormData();
    form.append('name', name);
    form.append('file', file);
    const res = await fetch(`${goUrl}/api/oauth/profiles`, {
      method: 'POST',
      body: form,
    });
    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(body.error ?? `upload failed: ${res.status}`);
    }
    log.info('oauth profile uploaded', { name });
  },

  patchProfile: async (goUrl, id, file) => {
    const form = new FormData();
    form.append('file', file);
    const res = await fetch(`${goUrl}/api/oauth/profiles/${id}`, {
      method: 'PATCH',
      body: form,
    });
    if (!res.ok) {
      const text = await res.text().catch(() => '');
      throw new Error(text || `patch failed: ${res.status}`);
    }
    log.info('oauth profile patched', { id });
  },

  deleteProfile: async (goUrl, id) => {
    const res = await fetch(`${goUrl}/api/oauth/profiles/${id}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error(`delete profile failed: ${res.status}`);
    log.info('oauth profile deleted', { id });
  },

  setDefault: async (goUrl, id) => {
    const res = await fetch(`${goUrl}/api/oauth/profiles/${id}/set-default`, {
      method: 'POST',
    });
    if (!res.ok) throw new Error(`set default failed: ${res.status}`);
    log.info('oauth profile set as default', { id });
  },

  initOAuth: async (goUrl, channelId) => {
    const res = await fetch(`${goUrl}/api/channels/${channelId}/auth`, {
      method: 'POST',
    });
    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as { code?: string; error?: string };
      throw new Error(body.code ?? body.error ?? `init oauth failed: ${res.status}`);
    }
    const data = (await res.json()) as { auth_url: string };
    log.info('oauth flow initiated', { channelId });
    return data;
  },

  submitCode: async (goUrl, channelId, code) => {
    const res = await fetch(`${goUrl}/api/channels/${channelId}/auth/callback`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    });
    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(body.error ?? `submit code failed: ${res.status}`);
    }
    log.info('oauth code submitted', { channelId });
  },
}));
