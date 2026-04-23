import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PanelHeader } from './panel-header';

describe('PanelHeader', () => {
  it('renders title as h3', () => {
    render(<PanelHeader title="Preview" />);
    const heading = screen.getByRole('heading', { level: 3, name: 'Preview' });
    expect(heading).toBeInTheDocument();
  });

  it('h3 has text-sm font-medium text-prose classes', () => {
    render(<PanelHeader title="Preview" />);
    const h3 = screen.getByRole('heading', { level: 3 });
    expect(h3.className).toContain('text-sm');
    expect(h3.className).toContain('font-medium');
    expect(h3.className).toContain('text-prose');
  });

  it('renders ReactNode title (not just string)', () => {
    render(<PanelHeader title={<span data-testid="custom-title">Rich</span>} />);
    expect(screen.getByTestId('custom-title')).toBeInTheDocument();
  });

  it('renders actions when provided', () => {
    render(<PanelHeader title="Preview" actions={<button>Click</button>} />);
    expect(screen.getByRole('button', { name: 'Click' })).toBeInTheDocument();
  });

  it('action wrapper absent when actions not provided', () => {
    render(<PanelHeader title="Preview" />);
    expect(document.querySelector('[data-slot="panel-header"]')!.children).toHaveLength(1);
  });

  it('root has data-slot="panel-header"', () => {
    render(<PanelHeader title="Preview" />);
    expect(document.querySelector('[data-slot="panel-header"]')).toBeInTheDocument();
  });

  it('root has flex items-center justify-between classes', () => {
    render(<PanelHeader title="Preview" />);
    const el = document.querySelector('[data-slot="panel-header"]')!;
    expect(el.className).toContain('flex');
    expect(el.className).toContain('items-center');
    expect(el.className).toContain('justify-between');
  });

  it('className merges on root', () => {
    render(<PanelHeader title="Preview" className="extra-class" />);
    expect(document.querySelector('[data-slot="panel-header"]')!.className).toContain('extra-class');
  });
});
