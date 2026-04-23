import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Textarea } from './textarea';

describe('Textarea', () => {
  it('renders with surface bg, border, and rounded-lg by default', () => {
    render(<Textarea aria-label="x" />);
    const el = screen.getByLabelText('x');
    expect(el.tagName).toBe('TEXTAREA');
    expect(el.className).toContain('bg-surface');
    expect(el.className).toContain('border-border');
    expect(el.className).toContain('rounded-lg');
  });

  it('applies brand focus ring classes', () => {
    render(<Textarea aria-label="x" />);
    const el = screen.getByLabelText('x');
    expect(el.className).toContain('focus-visible:border-brand/60');
    expect(el.className).toContain('focus-visible:ring-brand/30');
  });

  it('applies destructive ring when aria-invalid', () => {
    render(<Textarea aria-label="x" aria-invalid />);
    const el = screen.getByLabelText('x');
    expect(el.className).toContain('aria-invalid:border-destructive');
    expect(el.className).toContain('aria-invalid:ring-destructive/30');
  });

  it('applies disabled styles', () => {
    render(<Textarea aria-label="x" disabled />);
    const el = screen.getByLabelText('x');
    expect(el.className).toContain('disabled:opacity-50');
    expect(el).toBeDisabled();
  });

  it('passes className through', () => {
    render(<Textarea aria-label="x" className="font-mono custom-extra" />);
    const el = screen.getByLabelText('x');
    expect(el.className).toContain('font-mono');
    expect(el.className).toContain('custom-extra');
  });

  it('forwards rows and value props', () => {
    render(<Textarea aria-label="x" rows={5} defaultValue="hello" />);
    const el = screen.getByLabelText('x') as HTMLTextAreaElement;
    expect(el.rows).toBe(5);
    expect(el.value).toBe('hello');
  });
});
