import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from './tabs';

function Demo({ count, defaultValue = 'a' }: { count?: number; defaultValue?: string }) {
  return (
    <Tabs defaultValue={defaultValue}>
      <TabsList>
        <TabsTrigger value="a" count={count}>Alpha</TabsTrigger>
        <TabsTrigger value="b">Beta</TabsTrigger>
      </TabsList>
      <TabsContent value="a">A</TabsContent>
      <TabsContent value="b">B</TabsContent>
    </Tabs>
  );
}

describe('TabsTrigger count slot', () => {
  it('renders count pill when count > 0', () => {
    render(<Demo count={3} />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    const pill = trigger.querySelector('[data-slot="tabs-trigger-count"]');
    expect(pill).not.toBeNull();
    expect(pill).toHaveTextContent('3');
  });

  it('hides pill when count is 0', () => {
    render(<Demo count={0} />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    expect(trigger.querySelector('[data-slot="tabs-trigger-count"]')).toBeNull();
  });

  it('hides pill when count is undefined', () => {
    render(<Demo />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    expect(trigger.querySelector('[data-slot="tabs-trigger-count"]')).toBeNull();
  });

  it('count pill carries the active-state inversion class', () => {
    render(<Demo count={5} defaultValue="a" />);
    const trigger = screen.getByRole('tab', { name: /Alpha/ });
    const pill = trigger.querySelector('[data-slot="tabs-trigger-count"]')!;
    expect(pill.className).toContain('group-data-[state=active]/tabs-trigger:bg-primary-foreground/20');
  });
});
