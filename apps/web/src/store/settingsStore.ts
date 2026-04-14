import { create } from 'zustand';
import { createLogger } from '@/lib/logger';

const log = createLogger('settingsStore');

// Keys used in the key/value settings table
export const SETTINGS_KEYS = {
  VIDEOS_DIR: 'videos_dir',
  CACHE_DIR: 'cache_dir',
  DEFAULT_QUALITY: 'default_quality',
  DEFAULT_LANGUAGE: 'default_language',
  UPLOAD_STRATEGY: 'upload_strategy',
  UPLOAD_PARALLEL_COUNT: 'upload_parallel_count',
} as const;

export interface AppSettingsFormData {
  videos_dir: string;
  cache_dir: string;
  default_quality: string;
  default_language: string;
}

// Raw row from GET /api/settings
interface AppSettingRow {
  id: number;
  key: string;
  value: string;
  updated_at: number;
}

interface SettingsState {
  settings: Record<string, string>;
  loading: boolean;
  error: string | null;
  saving: boolean;
  saveError: string | null;
  saveSuccess: boolean;

  fetchSettings: (goUrl: string) => Promise<void>;
  saveSetting: (goUrl: string, key: string, value: string) => Promise<void>;
  saveAllSettings: (goUrl: string, data: AppSettingsFormData) => Promise<void>;
}

export const useSettingsStore = create<SettingsState>((set, get) => ({
  settings: {},
  loading: false,
  error: null,
  saving: false,
  saveError: null,
  saveSuccess: false,

  fetchSettings: async (goUrl: string) => {
    set({ loading: true, error: null });
    try {
      const res = await fetch(`${goUrl}/api/settings`);
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      const rows = (await res.json()) as AppSettingRow[];
      const record: Record<string, string> = {};
      for (const row of rows) {
        record[row.key] = row.value;
      }
      log.info('settings fetched', { count: rows.length });
      set({ settings: record, loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('fetch settings failed', { err: message });
      set({ loading: false, error: message });
    }
  },

  saveSetting: async (goUrl: string, key: string, value: string) => {
    const res = await fetch(`${goUrl}/api/settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key, value }),
    });
    if (!res.ok) throw new Error(`Failed: ${res.status}`);
    log.info('setting saved', { key });
    set((state) => ({
      settings: { ...state.settings, [key]: value },
    }));
  },

  saveAllSettings: async (goUrl: string, data: AppSettingsFormData) => {
    set({ saving: true, saveError: null, saveSuccess: false });
    try {
      const { saveSetting } = get();
      const entries: [string, string][] = [
        [SETTINGS_KEYS.VIDEOS_DIR, data.videos_dir],
        [SETTINGS_KEYS.CACHE_DIR, data.cache_dir],
        [SETTINGS_KEYS.DEFAULT_QUALITY, data.default_quality],
        [SETTINGS_KEYS.DEFAULT_LANGUAGE, data.default_language],
      ];
      for (const [key, value] of entries) {
        await saveSetting(goUrl, key, value);
      }
      log.info('all settings saved');
      set({ saving: false, saveSuccess: true });
      // Clear success indicator after 3s
      setTimeout(() => set({ saveSuccess: false }), 3000);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      log.error('save all settings failed', { err: message });
      set({ saving: false, saveError: message });
    }
  },
}));
