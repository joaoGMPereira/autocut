import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NumberStepper } from './number-stepper';

describe('NumberStepper', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders current value in the input', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={5} onValueChange={onValueChange} />);
    expect(screen.getByRole('spinbutton')).toHaveValue(5);
  });

  it('+ button increments by step (default 1)', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={5} onValueChange={onValueChange} />);
    fireEvent.click(screen.getByRole('button', { name: /increase/i }));
    expect(onValueChange).toHaveBeenCalledWith(6);
  });

  it('+ button increments by custom step', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={5} onValueChange={onValueChange} step={5} />);
    fireEvent.click(screen.getByRole('button', { name: /increase/i }));
    expect(onValueChange).toHaveBeenCalledWith(10);
  });

  it('− button decrements by step', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={5} onValueChange={onValueChange} />);
    fireEvent.click(screen.getByRole('button', { name: /decrease/i }));
    expect(onValueChange).toHaveBeenCalledWith(4);
  });

  it('+ button is disabled at max', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={10} onValueChange={onValueChange} max={10} />);
    expect(screen.getByRole('button', { name: /increase/i })).toBeDisabled();
  });

  it('− button is disabled at min', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={0} onValueChange={onValueChange} min={0} />);
    expect(screen.getByRole('button', { name: /decrease/i })).toBeDisabled();
  });

  it('incrementing past max clamps to max', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={9} onValueChange={onValueChange} max={10} step={5} />);
    fireEvent.click(screen.getByRole('button', { name: /increase/i }));
    expect(onValueChange).toHaveBeenCalledWith(10);
  });

  it('decrementing past min clamps to min', () => {
    const onValueChange = vi.fn();
    render(<NumberStepper value={3} onValueChange={onValueChange} min={1} step={5} />);
    fireEvent.click(screen.getByRole('button', { name: /decrease/i }));
    expect(onValueChange).toHaveBeenCalledWith(1);
  });

  it('with allowEmpty=true: empty input emits null', () => {
    const onValueChange = vi.fn();
    render(
      <NumberStepper
        allowEmpty
        value={5}
        onValueChange={onValueChange}
        placeholder="unlimited"
      />,
    );
    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '' } });
    expect(onValueChange).toHaveBeenCalledWith(null);
  });

  it('with allowEmpty=true and value=null: input shows empty', () => {
    const onValueChange = vi.fn();
    render(
      <NumberStepper
        allowEmpty
        value={null}
        onValueChange={onValueChange}
        placeholder="unlimited"
      />,
    );
    expect(screen.getByRole('spinbutton')).toHaveValue(null);
  });

  it('with allowEmpty=true and value=null: placeholder is shown', () => {
    const onValueChange = vi.fn();
    render(
      <NumberStepper
        allowEmpty
        value={null}
        onValueChange={onValueChange}
        placeholder="unlimited"
      />,
    );
    expect(screen.getByPlaceholderText('unlimited')).toBeInTheDocument();
  });

  it('root has data-slot="number-stepper"', () => {
    render(<NumberStepper value={0} onValueChange={vi.fn()} />);
    expect(document.querySelector('[data-slot="number-stepper"]')).toBeInTheDocument();
  });

  it('className merges on root', () => {
    render(<NumberStepper value={0} onValueChange={vi.fn()} className="extra-class" />);
    expect(
      document.querySelector('[data-slot="number-stepper"]')!.className,
    ).toContain('extra-class');
  });

  it('both buttons are disabled when disabled prop is set', () => {
    render(<NumberStepper value={5} onValueChange={vi.fn()} disabled />);
    const buttons = screen.getAllByRole('button');
    buttons.forEach((btn) => expect(btn).toBeDisabled());
  });
});
