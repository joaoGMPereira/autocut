import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ToolRow } from './ToolRow';
import { useSetupStore } from '@/store/setupStore';
import type { ToolStatus } from '@/types/setup';

// Mock stores
vi.mock('@/store/appStore', () => ({
  useAppStore: vi.fn((selector) =>
    selector({ goUrl: 'http://localhost:4071' }),
  ),
}));

describe('ToolRow', () => {
  beforeEach(() => {
    useSetupStore.setState({
      installStates: {},
      installLogs: {},
    });
  });

  it('shows "Installed" badge when tool is installed', () => {
    const tool: ToolStatus = {
      name: 'ffmpeg',
      installed: true,
      required: true,
      path: '/usr/local/bin/ffmpeg',
    };
    render(<ToolRow tool={tool} />);
    expect(screen.getByText('Installed')).toBeDefined();
  });

  it('shows "Missing" badge when required tool is not installed', () => {
    const tool: ToolStatus = {
      name: 'yt-dlp',
      installed: false,
      required: true,
    };
    render(<ToolRow tool={tool} />);
    expect(screen.getByText('Missing')).toBeDefined();
  });

  it('shows "Optional" badge when optional tool is not installed', () => {
    const tool: ToolStatus = {
      name: 'ollama',
      installed: false,
      required: false,
    };
    render(<ToolRow tool={tool} />);
    expect(screen.getByText('Optional')).toBeDefined();
  });

  it('shows Install button for auto-installable tools', () => {
    const tool: ToolStatus = {
      name: 'yt-dlp',
      installed: false,
      required: true,
    };
    render(<ToolRow tool={tool} />);
    expect(screen.getByText('Install')).toBeDefined();
  });

  it('shows log viewer when installing with logs', () => {
    useSetupStore.setState({
      installStates: { 'yt-dlp': 'installing' },
      installLogs: { 'yt-dlp': ['Downloading...', 'Extracting...'] },
    });
    const tool: ToolStatus = {
      name: 'yt-dlp',
      installed: false,
      required: true,
    };
    render(<ToolRow tool={tool} />);
    expect(screen.getByText('Downloading...')).toBeDefined();
    expect(screen.getByText('Extracting...')).toBeDefined();
  });
});
