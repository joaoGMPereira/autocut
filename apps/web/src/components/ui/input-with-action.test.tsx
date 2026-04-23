import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { InputWithAction } from './input-with-action';

const defaultProps = {
  value: 'hello',
  onValueChange: vi.fn(),
  onSubmit: vi.fn(),
  actionLabel: 'Save',
};

describe('InputWithAction', () => {
  it('renders input with current value', () => {
    render(<InputWithAction {...defaultProps} />);
    expect(screen.getByRole('textbox')).toHaveValue('hello');
  });

  it('renders button with actionLabel', () => {
    render(<InputWithAction {...defaultProps} actionLabel="Criar" />);
    expect(screen.getByRole('button', { name: 'Criar' })).toBeInTheDocument();
  });

  it('typing fires onValueChange with new string', () => {
    const onValueChange = vi.fn();
    render(<InputWithAction {...defaultProps} onValueChange={onValueChange} />);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'world' } });
    expect(onValueChange).toHaveBeenCalledWith('world');
  });

  it('pressing Enter fires onSubmit when not disabled', () => {
    const onSubmit = vi.fn();
    render(<InputWithAction {...defaultProps} onSubmit={onSubmit} />);
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' });
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it('pressing Enter does NOT fire onSubmit when actionDisabled=true', () => {
    const onSubmit = vi.fn();
    render(
      <InputWithAction {...defaultProps} onSubmit={onSubmit} actionDisabled />,
    );
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' });
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('pressing non-Enter key does NOT fire onSubmit', () => {
    const onSubmit = vi.fn();
    render(<InputWithAction {...defaultProps} onSubmit={onSubmit} />);
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Escape' });
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'a' });
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('clicking button fires onSubmit', () => {
    const onSubmit = vi.fn();
    render(<InputWithAction {...defaultProps} onSubmit={onSubmit} />);
    fireEvent.click(screen.getByRole('button'));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it('button is disabled when actionDisabled=true', () => {
    render(<InputWithAction {...defaultProps} actionDisabled />);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('placeholder forwarded to input', () => {
    render(<InputWithAction {...defaultProps} placeholder="Type here…" />);
    expect(screen.getByPlaceholderText('Type here…')).toBeInTheDocument();
  });

  it('root has data-slot="input-with-action"', () => {
    render(<InputWithAction {...defaultProps} />);
    expect(document.querySelector('[data-slot="input-with-action"]')).toBeInTheDocument();
  });

  it('className merges on root', () => {
    render(<InputWithAction {...defaultProps} className="extra-class" />);
    expect(
      document.querySelector('[data-slot="input-with-action"]')!.className,
    ).toContain('extra-class');
  });

  it('inputClassName merges on input', () => {
    render(<InputWithAction {...defaultProps} inputClassName="h-8" />);
    expect(screen.getByRole('textbox').className).toContain('h-8');
  });
});
