import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ColorInputField } from './ColorInputField';

describe('ColorInputField', () => {
  it('renders label and hex value in text input', () => {
    render(<ColorInputField label="Cor do Texto" value="#FFFFFF" onValueChange={vi.fn()} />);
    expect(screen.getByText('Cor do Texto')).toBeDefined();
    expect(screen.getByDisplayValue('#FFFFFF')).toBeDefined();
  });

  it('calls onValueChange when text input changes', async () => {
    const onChange = vi.fn();
    render(<ColorInputField label="Cor" value="#000000" onValueChange={onChange} />);
    // Two inputs share the same display value; pick the visible text input (not the sr-only color picker)
    const inputs = screen.getAllByDisplayValue('#000000');
    const input = inputs.find((el) => (el as HTMLInputElement).type !== 'color')!;
    await userEvent.clear(input);
    await userEvent.type(input, '#FF0000');
    expect(onChange).toHaveBeenCalled();
  });
});
