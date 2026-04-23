import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ToggleGroup } from './toggle-group';

const options = [
  { key: 'bold' as const, label: 'Negrito' },
  { key: 'italic' as const, label: 'Itálico' },
  { key: 'uppercase' as const, label: 'Maiúsculas' },
] as const;

type K = 'bold' | 'italic' | 'uppercase';

const allOff: Record<K, boolean> = { bold: false, italic: false, uppercase: false };

describe('ToggleGroup', () => {
  it('renders every option label', () => {
    render(
      <ToggleGroup options={options} value={allOff} onChange={() => {}} />,
    );
    expect(screen.getByText('Negrito')).toBeInTheDocument();
    expect(screen.getByText('Itálico')).toBeInTheDocument();
    expect(screen.getByText('Maiúsculas')).toBeInTheDocument();
  });

  it('active option has bg-brand class', () => {
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, bold: true }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText('Negrito').closest('button')!.className).toContain('bg-brand');
  });

  it('inactive option has bg-surface class', () => {
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, bold: true }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText('Itálico').closest('button')!.className).toContain('bg-surface');
  });

  it('aria-pressed tracks value map per key', () => {
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, italic: true }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText('Negrito').closest('button')).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByText('Itálico').closest('button')).toHaveAttribute('aria-pressed', 'true');
  });

  it('clicking active option calls onChange(key, false)', () => {
    const onChange = vi.fn();
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, bold: true }}
        onChange={onChange}
      />,
    );
    fireEvent.click(screen.getByText('Negrito'));
    expect(onChange).toHaveBeenCalledWith('bold', false);
  });

  it('clicking inactive option calls onChange(key, true)', () => {
    const onChange = vi.fn();
    render(
      <ToggleGroup options={options} value={allOff} onChange={onChange} />,
    );
    fireEvent.click(screen.getByText('Itálico'));
    expect(onChange).toHaveBeenCalledWith('italic', true);
  });

  it('onChange only fires once per click (not for other keys)', () => {
    const onChange = vi.fn();
    render(
      <ToggleGroup options={options} value={allOff} onChange={onChange} />,
    );
    fireEvent.click(screen.getByText('Maiúsculas'));
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith('uppercase', true);
  });

  it('container has data-slot="toggle-group"', () => {
    render(<ToggleGroup options={options} value={allOff} onChange={() => {}} />);
    expect(document.querySelector('[data-slot="toggle-group"]')).toBeInTheDocument();
  });

  it('merges className on root', () => {
    render(
      <ToggleGroup
        options={options}
        value={allOff}
        onChange={() => {}}
        className="extra-class"
      />,
    );
    expect(document.querySelector('[data-slot="toggle-group"]')!.className).toContain('extra-class');
  });
});
