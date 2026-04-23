import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SliderRow } from './slider-row';

describe('SliderRow', () => {
  it('renders label text', () => {
    render(
      <SliderRow label="Volume" min={0} max={100} value={50} onChange={() => {}} />,
    );
    expect(screen.getByText('Volume')).toBeInTheDocument();
  });

  it('range input has correct min, max, value attributes', () => {
    render(
      <SliderRow label="Volume" min={0} max={100} value={50} onChange={() => {}} />,
    );
    const input = screen.getByRole('slider');
    expect(input).toHaveAttribute('min', '0');
    expect(input).toHaveAttribute('max', '100');
    expect(input).toHaveAttribute('value', '50');
  });

  it('default format shows raw value as string', () => {
    render(
      <SliderRow label="Volume" min={0} max={100} value={42} onChange={() => {}} />,
    );
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('custom format function is applied to display value', () => {
    render(
      <SliderRow
        label="Volume"
        min={0}
        max={100}
        value={50}
        onChange={() => {}}
        format={(v) => `${v}%`}
      />,
    );
    expect(screen.getByText('50%')).toBeInTheDocument();
  });

  it('onChange fires with numeric value on input change', () => {
    const onChange = vi.fn();
    render(
      <SliderRow label="Volume" min={0} max={100} value={50} onChange={onChange} />,
    );
    fireEvent.change(screen.getByRole('slider'), { target: { value: '75' } });
    expect(onChange).toHaveBeenCalledWith(75);
  });

  it('step attribute forwarded to input when provided', () => {
    render(
      <SliderRow label="Volume" min={0} max={100} value={50} onChange={() => {}} step={5} />,
    );
    expect(screen.getByRole('slider')).toHaveAttribute('step', '5');
  });

  it('input has accent-brand class', () => {
    render(
      <SliderRow label="Volume" min={0} max={100} value={50} onChange={() => {}} />,
    );
    expect(screen.getByRole('slider').className).toContain('accent-brand');
  });

  it('merges className on root', () => {
    render(
      <SliderRow
        label="Volume"
        min={0}
        max={100}
        value={50}
        onChange={() => {}}
        className="extra-class"
      />,
    );
    const el = document.querySelector('[data-slot="slider-row"]')!;
    expect(el.className).toContain('extra-class');
  });
});
