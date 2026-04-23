import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { InfoBanner } from './info-banner';

describe('InfoBanner', () => {
  it('renders children', () => {
    render(<InfoBanner>Reviewing previous submission.</InfoBanner>);
    expect(screen.getByText('Reviewing previous submission.')).toBeInTheDocument();
  });

  it('has anchor design-system classes', () => {
    render(<InfoBanner data-testid="b">x</InfoBanner>);
    const el = screen.getByTestId('b');
    expect(el.className).toContain('rounded-md');
    expect(el.className).toContain('border-border');
    expect(el.className).toContain('bg-card');
    expect(el.className).toContain('text-prose');
    expect(el.className).toContain('px-4');
    expect(el.className).toContain('py-3');
    expect(el.className).toContain('text-xs');
  });

  it('merges className', () => {
    render(<InfoBanner data-testid="b" className="extra-class">x</InfoBanner>);
    expect(screen.getByTestId('b').className).toContain('extra-class');
  });

  it('forwards HTML attributes', () => {
    render(<InfoBanner data-testid="b" role="status">x</InfoBanner>);
    expect(screen.getByTestId('b')).toHaveAttribute('role', 'status');
  });
});
