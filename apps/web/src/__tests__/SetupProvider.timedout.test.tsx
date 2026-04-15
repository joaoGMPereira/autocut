import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, act } from '@testing-library/react';
import { SetupProvider } from '@/components/setup/SetupProvider';

vi.mock('@/hooks/useSetupCheck', () => ({
  useSetupCheck: vi.fn(() => ({
    isReady: false,
    isLoading: true,
    tools: [],
    missingRequired: [],
    error: null,
  })),
}));

vi.mock('@/components/setup/SetupGate', () => ({
  SetupGate: () => null,
}));

describe('SetupProvider — timedOut state', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not render children after 3s timeout while backend is still loading', async () => {
    const { queryByTestId } = render(
      <SetupProvider>
        <div data-testid="main-content" />
      </SetupProvider>
    );

    await act(async () => {
      vi.advanceTimersByTime(3100);
    });

    expect(queryByTestId('main-content')).toBeNull();
  });
});
