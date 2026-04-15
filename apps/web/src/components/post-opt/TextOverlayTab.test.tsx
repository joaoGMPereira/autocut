import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TextOverlayTab } from './TextOverlayTab';

describe('TextOverlayTab', () => {
  it('renders master toggle', () => {
    render(<TextOverlayTab />);
    expect(screen.getByText('Adicionar Overlays de Texto')).toBeDefined();
  });

  it('shows overlay list only when enabled', async () => {
    render(<TextOverlayTab />);
    // Initially disabled — no add button
    expect(screen.queryByText('+ Adicionar Texto')).toBeNull();
    // Enable
    const toggle = screen.getByRole('switch', { name: 'Adicionar Overlays de Texto' });
    await userEvent.click(toggle);
    expect(screen.getByText('+ Adicionar Texto')).toBeDefined();
  });

  it('adds a new overlay when clicking Adicionar Texto', async () => {
    render(<TextOverlayTab />);
    const toggle = screen.getByRole('switch', { name: 'Adicionar Overlays de Texto' });
    await userEvent.click(toggle);
    await userEvent.click(screen.getByText('+ Adicionar Texto'));
    // Default text appears in input
    expect(screen.getByDisplayValue('Novo Texto')).toBeDefined();
  });

  it('removes an overlay when clicking delete', async () => {
    render(<TextOverlayTab />);
    const toggle = screen.getByRole('switch', { name: 'Adicionar Overlays de Texto' });
    await userEvent.click(toggle);
    await userEvent.click(screen.getByText('+ Adicionar Texto'));
    // Text input is present
    expect(screen.getByDisplayValue('Novo Texto')).toBeDefined();
    // Click delete
    await userEvent.click(screen.getByRole('button', { name: 'Remover overlay' }));
    expect(screen.queryByDisplayValue('Novo Texto')).toBeNull();
  });
});
