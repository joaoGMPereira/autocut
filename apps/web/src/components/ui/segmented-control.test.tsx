import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SegmentedControl } from './segmented-control';

// ── Flat variant ─────────────────────────────────────────────────────────────

const flatOptions = [
  { value: 'liberal' as const, label: 'Liberal' },
  { value: 'balanced' as const, label: 'Balanced' },
  { value: 'selective' as const, label: 'Selective' },
];

describe('SegmentedControl — flat (default)', () => {
  it('renders all option labels', () => {
    render(
      <SegmentedControl options={flatOptions} value="liberal" onChange={() => {}} />,
    );
    expect(screen.getByText('Liberal')).toBeInTheDocument();
    expect(screen.getByText('Balanced')).toBeInTheDocument();
    expect(screen.getByText('Selective')).toBeInTheDocument();
  });

  it('active option has bg-brand class', () => {
    render(
      <SegmentedControl options={flatOptions} value="balanced" onChange={() => {}} />,
    );
    const btn = screen.getByText('Balanced').closest('button')!;
    expect(btn.className).toContain('bg-brand');
  });

  it('inactive options have bg-surface class', () => {
    render(
      <SegmentedControl options={flatOptions} value="liberal" onChange={() => {}} />,
    );
    const btn = screen.getByText('Balanced').closest('button')!;
    expect(btn.className).toContain('bg-surface');
  });

  it('calls onChange with correct value on click', () => {
    const onChange = vi.fn();
    render(
      <SegmentedControl options={flatOptions} value="liberal" onChange={onChange} />,
    );
    fireEvent.click(screen.getByText('Balanced'));
    expect(onChange).toHaveBeenCalledWith('balanced');
  });

  it('aria-pressed reflects active state', () => {
    render(
      <SegmentedControl options={flatOptions} value="liberal" onChange={() => {}} />,
    );
    expect(
      screen.getByText('Liberal').closest('button'),
    ).toHaveAttribute('aria-pressed', 'true');
    expect(
      screen.getByText('Balanced').closest('button'),
    ).toHaveAttribute('aria-pressed', 'false');
  });

  it('container has data-variant="flat"', () => {
    render(
      <SegmentedControl options={flatOptions} value="liberal" onChange={() => {}} />,
    );
    expect(
      document.querySelector('[data-variant="flat"]'),
    ).toBeInTheDocument();
  });

  it('merges className on container', () => {
    render(
      <SegmentedControl
        options={flatOptions}
        value="liberal"
        onChange={() => {}}
        className="extra-class"
      />,
    );
    const el = document.querySelector('[data-slot="segmented-control"]')!;
    expect(el.className).toContain('extra-class');
  });
});

// ── Card variant ──────────────────────────────────────────────────────────────

const cardOptions = [
  { value: 'subtle' as const, label: 'Subtle', description: '1–2% changes' },
  { value: 'aggressive' as const, label: 'Aggressive', description: '5–10% changes' },
];

describe('SegmentedControl — card variant', () => {
  it('renders labels and descriptions', () => {
    render(
      <SegmentedControl
        variant="card"
        options={cardOptions}
        value="subtle"
        onChange={() => {}}
      />,
    );
    expect(screen.getByText('Subtle')).toBeInTheDocument();
    expect(screen.getByText('1–2% changes')).toBeInTheDocument();
    expect(screen.getByText('Aggressive')).toBeInTheDocument();
    expect(screen.getByText('5–10% changes')).toBeInTheDocument();
  });

  it('active card has border-brand class', () => {
    render(
      <SegmentedControl
        variant="card"
        options={cardOptions}
        value="subtle"
        onChange={() => {}}
      />,
    );
    const btn = screen.getByText('Subtle').closest('button')!;
    expect(btn.className).toContain('border-brand');
  });

  it('inactive card has border-border class', () => {
    render(
      <SegmentedControl
        variant="card"
        options={cardOptions}
        value="subtle"
        onChange={() => {}}
      />,
    );
    const btn = screen.getByText('Aggressive').closest('button')!;
    expect(btn.className).toContain('border-border');
  });

  it('calls onChange on card click', () => {
    const onChange = vi.fn();
    render(
      <SegmentedControl
        variant="card"
        options={cardOptions}
        value="subtle"
        onChange={onChange}
      />,
    );
    fireEvent.click(screen.getByText('Aggressive'));
    expect(onChange).toHaveBeenCalledWith('aggressive');
  });

  it('container has data-variant="card"', () => {
    render(
      <SegmentedControl
        variant="card"
        options={cardOptions}
        value="subtle"
        onChange={() => {}}
      />,
    );
    expect(
      document.querySelector('[data-variant="card"]'),
    ).toBeInTheDocument();
  });
});

// ── wrap prop ────────────────────────────────────────────────────────────────

describe('SegmentedControl — flat with wrap=true', () => {
  it('container carries flex-wrap class', () => {
    render(
      <SegmentedControl
        options={flatOptions}
        value="liberal"
        onChange={() => {}}
        wrap
      />,
    );
    const el = document.querySelector('[data-slot="segmented-control"]')!;
    expect(el.className).toContain('flex-wrap');
  });

  it('buttons do NOT carry flex-1 when wrap=true', () => {
    render(
      <SegmentedControl
        options={flatOptions}
        value="liberal"
        onChange={() => {}}
        wrap
      />,
    );
    const btn = screen.getByText('Liberal').closest('button')!;
    expect(btn.className).not.toContain('flex-1');
  });

  it('wrap=false (default) container does NOT carry flex-wrap', () => {
    render(
      <SegmentedControl options={flatOptions} value="liberal" onChange={() => {}} />,
    );
    const el = document.querySelector('[data-slot="segmented-control"]')!;
    expect(el.className).not.toContain('flex-wrap');
  });

  it('wrap=false (default) buttons carry flex-1', () => {
    render(
      <SegmentedControl options={flatOptions} value="liberal" onChange={() => {}} />,
    );
    const btn = screen.getByText('Liberal').closest('button')!;
    expect(btn.className).toContain('flex-1');
  });

  it('wrap prop has no effect on card variant (container stays grid)', () => {
    render(
      <SegmentedControl
        variant="card"
        options={cardOptions}
        value="subtle"
        onChange={() => {}}
        wrap
      />,
    );
    const el = document.querySelector('[data-slot="segmented-control"]')!;
    expect(el.className).toContain('grid');
    expect(el.className).not.toContain('flex-wrap');
  });
});
