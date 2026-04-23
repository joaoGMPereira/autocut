import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SettingRow } from './setting-row';

describe('SettingRow', () => {
  it('renders label text', () => {
    render(<SettingRow label="Skip Regenerate"><span>control</span></SettingRow>);
    expect(screen.getByText('Skip Regenerate')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(
      <SettingRow label="Skip Regenerate" description="Reuse previous output">
        <span>control</span>
      </SettingRow>,
    );
    expect(screen.getByText('Reuse previous output')).toBeInTheDocument();
  });

  it('omits description slot when not provided', () => {
    render(<SettingRow label="Label"><span>control</span></SettingRow>);
    expect(
      document.querySelector('[data-slot="setting-row-description"]'),
    ).toBeNull();
  });

  it('renders children in control slot', () => {
    render(<SettingRow label="Label"><button>toggle</button></SettingRow>);
    expect(screen.getByRole('button', { name: 'toggle' })).toBeInTheDocument();
  });

  it('root has flex layout classes', () => {
    render(<SettingRow label="L"><span/></SettingRow>);
    const el = document.querySelector('[data-slot="setting-row"]')!;
    expect(el.className).toContain('flex');
    expect(el.className).toContain('items-center');
    expect(el.className).toContain('justify-between');
  });

  it('merges className on root', () => {
    render(<SettingRow label="L" className="extra-class"><span/></SettingRow>);
    const el = document.querySelector('[data-slot="setting-row"]')!;
    expect(el.className).toContain('extra-class');
  });
});
