import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TextStyleEditorPanel } from './TextStyleEditorPanel';
import { DEFAULT_STYLE } from '@/types/text-overlay';

describe('TextStyleEditorPanel', () => {
  it('renders Tipografia section', () => {
    render(<TextStyleEditorPanel config={DEFAULT_STYLE} onConfigChange={vi.fn()} />);
    expect(screen.getByText('Tipografia')).toBeDefined();
    expect(screen.getByText('Negrito')).toBeDefined();
    expect(screen.getByText('Itálico')).toBeDefined();
    expect(screen.getByText('CAIXA ALTA')).toBeDefined();
  });

  it('renders Cores e Bordas section', () => {
    render(<TextStyleEditorPanel config={DEFAULT_STYLE} onConfigChange={vi.fn()} />);
    expect(screen.getByText('Cores e Bordas')).toBeDefined();
    expect(screen.getByText('Cor do Texto')).toBeDefined();
    expect(screen.getByText('Borda (Outline)')).toBeDefined();
    expect(screen.getByText('Sombra')).toBeDefined();
  });

  it('renders Ajuste Fino section', () => {
    render(<TextStyleEditorPanel config={DEFAULT_STYLE} onConfigChange={vi.fn()} />);
    expect(screen.getByText('Ajuste Fino de Posição')).toBeDefined();
    expect(screen.getByText(/Offset Vertical/)).toBeDefined();
  });

  it('calls onConfigChange with isBold=true when Negrito is toggled', async () => {
    const onChange = vi.fn();
    render(<TextStyleEditorPanel config={DEFAULT_STYLE} onConfigChange={onChange} />);
    const boldSwitch = screen.getByRole('switch', { name: 'Negrito' });
    await userEvent.click(boldSwitch);
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ isBold: true }));
  });
});
