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
});
