import { vi, describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { SetupProvider } from '@/components/setup/SetupProvider';

vi.mock('@/hooks/useSetupCheck', () => ({
  useSetupCheck: vi.fn(() => ({
    isReady: false,
    isLoading: false,
    tools: [],
    missingRequired: [],
    error: 'connection refused',
  })),
}));

vi.mock('@/components/setup/SetupGate', () => ({
  SetupGate: () => null,
}));

describe('SetupProvider — error state', () => {
  it('does not render children when backend returns an error and no tools loaded', () => {
    const { queryByTestId } = render(
      <SetupProvider>
        <div data-testid="main-content" />
      </SetupProvider>
    );

    expect(queryByTestId('main-content')).toBeNull();
  });
});
