import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PageHeader } from './page-header';

describe('PageHeader', () => {
  it('renders title as an h1 with display font classes', () => {
    render(<PageHeader title="Settings" />);
    const heading = screen.getByRole('heading', { level: 1 });
    expect(heading).toHaveTextContent('Settings');
    expect(heading.className).toContain('font-display');
    expect(heading.className).toContain('text-[32px]');
    expect(heading.className).toContain('text-heading');
  });

  it('omits description when not provided', () => {
    render(<PageHeader title="X" />);
    expect(document.querySelector('p[data-slot="page-header-description"]')).toBeNull();
  });

  it('renders description when provided', () => {
    render(<PageHeader title="X" description="Manage things" />);
    expect(screen.getByText('Manage things')).toBeInTheDocument();
  });

  it('omits actions slot when not provided', () => {
    render(<PageHeader title="X" />);
    expect(document.querySelector('[data-slot="page-header-actions"]')).toBeNull();
  });

  it('renders actions slot when provided', () => {
    render(<PageHeader title="X" actions={<button>Add</button>} />);
    expect(screen.getByRole('button', { name: 'Add' })).toBeInTheDocument();
  });
});
