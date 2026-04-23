import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { FileInputButton } from './file-input-button';

describe('FileInputButton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders a button with default text "Choose file"', () => {
    render(<FileInputButton onFilesSelected={vi.fn()} />);
    expect(screen.getByRole('button', { name: 'Choose file' })).toBeInTheDocument();
  });

  it('renders custom children text', () => {
    render(<FileInputButton onFilesSelected={vi.fn()}>Upload Image</FileInputButton>);
    expect(screen.getByRole('button', { name: 'Upload Image' })).toBeInTheDocument();
  });

  it('button click triggers file input click', () => {
    render(<FileInputButton onFilesSelected={vi.fn()} />);
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    const clickSpy = vi.spyOn(input, 'click');
    fireEvent.click(screen.getByRole('button'));
    expect(clickSpy).toHaveBeenCalledTimes(1);
  });

  it('onFilesSelected is called with FileList when file chosen', () => {
    const onFilesSelected = vi.fn();
    render(<FileInputButton onFilesSelected={onFilesSelected} />);
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['content'], 'test.png', { type: 'image/png' });
    Object.defineProperty(input, 'files', { value: [file], configurable: true });
    fireEvent.change(input);
    expect(onFilesSelected).toHaveBeenCalledTimes(1);
    expect(onFilesSelected).toHaveBeenCalledWith(expect.objectContaining({ 0: file }));
  });

  it('input value is reset after selection so same file can be re-selected', () => {
    const onFilesSelected = vi.fn();
    render(<FileInputButton onFilesSelected={onFilesSelected} />);
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['content'], 'test.png', { type: 'image/png' });
    // We track whether the value setter was called with ''
    let resetCalled = false;
    Object.defineProperty(input, 'value', {
      set(v: string) {
        if (v === '') resetCalled = true;
      },
      configurable: true,
    });
    Object.defineProperty(input, 'files', { value: [file], configurable: true });
    fireEvent.change(input);
    expect(resetCalled).toBe(true);
  });

  it('button is disabled when disabled prop is true', () => {
    render(<FileInputButton onFilesSelected={vi.fn()} disabled />);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('root has data-slot="file-input-button"', () => {
    render(<FileInputButton onFilesSelected={vi.fn()} />);
    expect(document.querySelector('[data-slot="file-input-button"]')).toBeInTheDocument();
  });

  it('accept is forwarded to the hidden input', () => {
    render(<FileInputButton onFilesSelected={vi.fn()} accept="image/*" />);
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    expect(input.accept).toBe('image/*');
  });

  it('multiple is forwarded to the hidden input', () => {
    render(<FileInputButton onFilesSelected={vi.fn()} multiple />);
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    expect(input.multiple).toBe(true);
  });

  it('className merges on root wrapper', () => {
    render(<FileInputButton onFilesSelected={vi.fn()} className="my-custom-class" />);
    const root = document.querySelector('[data-slot="file-input-button"]')!;
    expect(root.className).toContain('my-custom-class');
  });

  it('does not call onFilesSelected when no files selected', () => {
    const onFilesSelected = vi.fn();
    render(<FileInputButton onFilesSelected={onFilesSelected} />);
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    Object.defineProperty(input, 'files', { value: [], configurable: true });
    fireEvent.change(input);
    expect(onFilesSelected).not.toHaveBeenCalled();
  });
});
