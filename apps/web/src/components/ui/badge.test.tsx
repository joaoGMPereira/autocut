import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge } from './badge';

describe('Badge', () => {
  it('renders default variant unchanged', () => {
    render(<Badge>default</Badge>);
    const badge = screen.getByText('default');
    expect(badge).toHaveAttribute('data-variant', 'default');
    expect(badge.className).toContain('bg-primary');
  });

  it('renders destructive variant unchanged (solid red)', () => {
    render(<Badge variant="destructive">err</Badge>);
    const badge = screen.getByText('err');
    expect(badge).toHaveAttribute('data-variant', 'destructive');
    expect(badge.className).toContain('bg-destructive');
  });

  describe('soft-fill variants', () => {
    it.each([
      ['success', ['bg-success/10', 'border-success/20', 'text-success']],
      ['warning', ['bg-warning/15', 'border-warning/40', 'text-warning']],
      ['info', ['bg-info/10', 'border-info/20', 'text-info']],
      ['brand', ['bg-brand/10', 'border-brand/40', 'text-brand']],
    ] as const)('renders %s with soft-fill classes', (variant, classes) => {
      render(<Badge variant={variant}>label</Badge>);
      const badge = screen.getByText('label');
      expect(badge).toHaveAttribute('data-variant', variant);
      classes.forEach((c) => expect(badge.className).toContain(c));
    });
  });
});
