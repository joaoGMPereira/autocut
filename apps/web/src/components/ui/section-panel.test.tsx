import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SectionPanel } from './section-panel';

describe('SectionPanel', () => {
  it('renders title as an eyebrow paragraph', () => {
    render(
      <SectionPanel title="AI Configuration">
        <div>body</div>
      </SectionPanel>,
    );
    const title = screen.getByText('AI Configuration');
    expect(title.tagName).toBe('P');
    expect(title.className).toContain('uppercase');
  });

  it('defaults to md size: p-4 + text-xs eyebrow in text-prose', () => {
    render(
      <SectionPanel title="T">
        <div>body</div>
      </SectionPanel>,
    );
    const root = screen.getByText('T').closest('[data-slot="section-panel"]')!;
    expect(root.className).toContain('p-4');
    expect(root.className).toContain('space-y-4');
    const title = screen.getByText('T');
    expect(title.className).toContain('text-xs');
    expect(title.className).toContain('font-medium');
    expect(title.className).toContain('text-prose');
    expect(title.className).toContain('tracking-wider');
  });

  it('applies sm size: p-3 + text-[10px] eyebrow in text-subtle', () => {
    render(
      <SectionPanel title="T" size="sm">
        <div>body</div>
      </SectionPanel>,
    );
    const root = screen.getByText('T').closest('[data-slot="section-panel"]')!;
    expect(root.className).toContain('p-3');
    expect(root.className).toContain('space-y-3');
    const title = screen.getByText('T');
    expect(title.className).toContain('text-[10px]');
    expect(title.className).toContain('font-semibold');
    expect(title.className).toContain('text-subtle');
  });

  it('omits description when not provided', () => {
    render(
      <SectionPanel title="T">
        <div>body</div>
      </SectionPanel>,
    );
    expect(
      document.querySelector('[data-slot="section-panel-description"]'),
    ).toBeNull();
  });

  it('renders description when provided', () => {
    render(
      <SectionPanel title="T" description="Hint text.">
        <div>body</div>
      </SectionPanel>,
    );
    expect(screen.getByText('Hint text.')).toBeInTheDocument();
  });

  it('omits actions slot when not provided', () => {
    render(
      <SectionPanel title="T">
        <div>body</div>
      </SectionPanel>,
    );
    expect(
      document.querySelector('[data-slot="section-panel-actions"]'),
    ).toBeNull();
  });

  it('renders actions slot when provided', () => {
    render(
      <SectionPanel title="T" actions={<button>Toggle</button>}>
        <div>body</div>
      </SectionPanel>,
    );
    expect(screen.getByRole('button', { name: 'Toggle' })).toBeInTheDocument();
  });

  it('renders children below the header', () => {
    render(
      <SectionPanel title="T">
        <div data-testid="child">body</div>
      </SectionPanel>,
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
  });

  it('anchors: rounded-lg border-border bg-card on wrapper', () => {
    render(
      <SectionPanel title="T">
        <div>body</div>
      </SectionPanel>,
    );
    const root = screen.getByText('T').closest('[data-slot="section-panel"]')!;
    expect(root.className).toContain('rounded-lg');
    expect(root.className).toContain('border-border');
    expect(root.className).toContain('bg-card');
  });

  it('merges className on the root', () => {
    render(
      <SectionPanel title="T" className="extra-class">
        <div>body</div>
      </SectionPanel>,
    );
    const root = screen.getByText('T').closest('[data-slot="section-panel"]')!;
    expect(root.className).toContain('extra-class');
  });
});
