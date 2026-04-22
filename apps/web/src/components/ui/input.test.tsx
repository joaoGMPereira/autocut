import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Input } from './input';

describe('Input', () => {
  it('renders with surface bg and border classes by default', () => {
    render(<Input aria-label="x" />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('bg-surface');
    expect(input.className).toContain('border-border');
    expect(input.className).toContain('rounded-lg');
    expect(input.className).toContain('h-9');
  });

  it('applies brand focus ring class', () => {
    render(<Input aria-label="x" />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('focus-visible:border-brand/60');
    expect(input.className).toContain('focus-visible:ring-brand/30');
  });

  it('applies destructive ring when aria-invalid', () => {
    render(<Input aria-label="x" aria-invalid />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('aria-invalid:border-destructive');
    expect(input.className).toContain('aria-invalid:ring-destructive/30');
  });

  it('applies disabled styles', () => {
    render(<Input aria-label="x" disabled />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('disabled:opacity-50');
    expect(input).toBeDisabled();
  });

  it('passes className through', () => {
    render(<Input aria-label="x" className="font-mono custom-extra" />);
    const input = screen.getByLabelText('x');
    expect(input.className).toContain('font-mono');
    expect(input.className).toContain('custom-extra');
  });
});
