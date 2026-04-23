import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FormField } from './form-field';
import { Input } from './input';

describe('FormField', () => {
  it('wires label htmlFor to child input id (auto-generated)', () => {
    render(
      <FormField label="Name">
        <Input data-testid="i" />
      </FormField>,
    );
    const label = screen.getByText('Name').closest('label') as HTMLLabelElement;
    const input = screen.getByTestId('i');
    expect(label.htmlFor).toBeTruthy();
    expect(input.id).toBe(label.htmlFor);
  });

  it('respects child-supplied id', () => {
    render(
      <FormField label="Name">
        <Input id="my-id" data-testid="i" />
      </FormField>,
    );
    const label = screen.getByText('Name').closest('label') as HTMLLabelElement;
    expect(label.htmlFor).toBe('my-id');
    expect((screen.getByTestId('i') as HTMLInputElement).id).toBe('my-id');
  });

  it('omits required marker by default', () => {
    render(
      <FormField label="Name">
        <Input data-testid="i" />
      </FormField>,
    );
    expect(screen.queryByText('*')).not.toBeInTheDocument();
  });

  it('renders required marker when required', () => {
    render(
      <FormField label="Name" required>
        <Input data-testid="i" />
      </FormField>,
    );
    expect(screen.getByText('*')).toBeInTheDocument();
  });

  it('wires aria-describedby to description', () => {
    render(
      <FormField label="Name" description="hint text">
        <Input data-testid="i" />
      </FormField>,
    );
    const input = screen.getByTestId('i');
    const describedBy = input.getAttribute('aria-describedby');
    expect(describedBy).toBeTruthy();
    const desc = screen.getByText('hint text');
    expect(desc.id).toBe(describedBy);
  });

  it('error replaces description and marks aria-invalid', () => {
    render(
      <FormField label="Name" description="hint" error="bad value">
        <Input data-testid="i" />
      </FormField>,
    );
    expect(screen.queryByText('hint')).not.toBeInTheDocument();
    expect(screen.getByText('bad value')).toBeInTheDocument();
    const input = screen.getByTestId('i');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    const describedBy = input.getAttribute('aria-describedby');
    expect(describedBy).toBeTruthy();
    expect(screen.getByText('bad value').id).toBe(describedBy);
  });

  describe('counter slot', () => {
    it('renders counter inline-right of label when provided', () => {
      render(
        <FormField label="Title" counter="12/60">
          <Input data-testid="i" />
        </FormField>,
      );
      const counter = screen.getByText('12/60');
      expect(counter.tagName).toBe('SPAN');
      expect(counter.className).toContain('text-xs');
      expect(counter.className).toContain('text-caption');
    });

    it('wraps label + counter in a flex-between row when counter present', () => {
      render(
        <FormField label="Title" counter="5/30">
          <Input data-testid="i" />
        </FormField>,
      );
      const label = screen.getByText('Title').closest('label') as HTMLLabelElement;
      const row = label.parentElement as HTMLDivElement;
      expect(row.className).toContain('flex');
      expect(row.className).toContain('justify-between');
      expect(row).toContainElement(screen.getByText('5/30'));
    });

    it('omits counter when not provided (label is direct child of root)', () => {
      render(
        <FormField label="Title">
          <Input data-testid="i" />
        </FormField>,
      );
      const label = screen.getByText('Title').closest('label') as HTMLLabelElement;
      const parent = label.parentElement as HTMLElement;
      expect(parent.getAttribute('data-slot')).toBe('form-field');
    });

    it('coexists with description below the control', () => {
      render(
        <FormField label="Title" counter="3/30" description="hint">
          <Input data-testid="i" />
        </FormField>,
      );
      expect(screen.getByText('3/30')).toBeInTheDocument();
      expect(screen.getByText('hint')).toBeInTheDocument();
    });

    it('still wires htmlFor when counter is present', () => {
      render(
        <FormField label="Title" counter="0/10">
          <Input data-testid="i" />
        </FormField>,
      );
      const label = screen.getByText('Title').closest('label') as HTMLLabelElement;
      const input = screen.getByTestId('i');
      expect(label.htmlFor).toBeTruthy();
      expect(input.id).toBe(label.htmlFor);
    });
  });
});
