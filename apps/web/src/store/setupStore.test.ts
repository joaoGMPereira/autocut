import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useSetupStore } from './setupStore';
import type { ToolStatus } from '@/types/setup';

const mockTools: ToolStatus[] = [
  { name: 'yt-dlp', installed: false, required: true, path: '', version: '' },
  {
    name: 'ffmpeg',
    installed: true,
    required: true,
    path: '/usr/local/bin/ffmpeg',
    version: '6.1',
  },
  {
    name: 'TwitchDownloaderCLI',
    installed: false,
    required: false,
    path: '',
    version: '',
  },
];

describe('setupStore', () => {
  beforeEach(() => {
    useSetupStore.setState({
      tools: [],
      loading: false,
      error: null,
      installStates: {},
      installLogs: {},
    });
  });

  it('fetchStatus populates tools correctly', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ tools: mockTools }),
    });

    await useSetupStore.getState().fetchStatus('http://localhost:4071');
    const state = useSetupStore.getState();

    expect(state.tools).toHaveLength(3);
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
  });

  it('allRequiredInstalled returns false when required tool is missing', () => {
    useSetupStore.setState({ tools: mockTools });
    expect(useSetupStore.getState().allRequiredInstalled()).toBe(false);
  });

  it('allRequiredInstalled returns true when all required tools are installed', () => {
    const allInstalled = mockTools.map((t) => ({
      ...t,
      installed: t.required ? true : t.installed,
    }));
    useSetupStore.setState({ tools: allInstalled });
    expect(useSetupStore.getState().allRequiredInstalled()).toBe(true);
  });

  it('missingRequired returns only required tools that are not installed', () => {
    useSetupStore.setState({ tools: mockTools });
    const missing = useSetupStore.getState().missingRequired();
    expect(missing).toHaveLength(1);
    expect(missing[0].name).toBe('yt-dlp');
  });

  it('startInstall sets installState to installing', () => {
    // Mock EventSource
    const mockES = {
      onmessage: null as ((e: MessageEvent) => void) | null,
      onerror: null as (() => void) | null,
      close: vi.fn(),
    };
    global.EventSource = vi.fn().mockImplementation(() => mockES) as unknown as typeof EventSource;
    global.fetch = vi.fn().mockResolvedValue({ ok: true });

    useSetupStore.getState().startInstall('http://localhost:4071', 'yt-dlp');

    expect(useSetupStore.getState().installStates['yt-dlp']).toBe('installing');
  });
});
