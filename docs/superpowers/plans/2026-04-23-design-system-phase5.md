# Design System Phase 5 — Primitive Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract four patterns left inline after Phase 4 into reusable design system primitives and migrate all their call sites.

**Architecture:** Four commits ship primitives (A–D), four commits migrate consumers (E–H). Each commit is independently gated by `cd apps/web && npx vitest run`. Coordinator runs `npx next build` externally after each commit. Never `--no-verify`. `SegmentedControl` gains a `wrap` prop; `ToggleGroup`, `InputWithAction`, and `PanelHeader` are new files.

**Tech Stack:** React 19, TypeScript, Tailwind CSS, Vitest + @testing-library/react, `cn` from `@/lib/utils`.

---

## File Map

| Action | File | Purpose |
|--------|------|---------|
| Modify | `apps/web/src/components/ui/segmented-control.tsx` | Add `wrap` prop |
| Modify | `apps/web/src/components/ui/segmented-control.test.tsx` | New wrap tests |
| Create | `apps/web/src/components/ui/toggle-group.tsx` | Multi-select primitive |
| Create | `apps/web/src/components/ui/toggle-group.test.tsx` | ToggleGroup tests |
| Create | `apps/web/src/components/ui/input-with-action.tsx` | Input + Button primitive |
| Create | `apps/web/src/components/ui/input-with-action.test.tsx` | InputWithAction tests |
| Create | `apps/web/src/components/ui/panel-header.tsx` | Card header primitive |
| Create | `apps/web/src/components/ui/panel-header.test.tsx` | PanelHeader tests |
| Modify | `apps/web/src/components/pipeline/steps/StepMode.tsx` | 9 sites — Commits E, F, G, H |
| Modify | `apps/web/src/components/channels/ChannelConfigSheet.tsx` | 1 site — Commit G |

---

## Task A: Add `wrap` prop to SegmentedControl

**Files:**
- Modify: `apps/web/src/components/ui/segmented-control.tsx`
- Modify: `apps/web/src/components/ui/segmented-control.test.tsx`

- [ ] **Step 1: Write the three new failing tests (append to the existing file)**

Open `apps/web/src/components/ui/segmented-control.test.tsx`. Append this block after the last `describe` block (after line 159):

```typescript
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd apps/web && npx vitest run src/components/ui/segmented-control.test.tsx
```

Expected: 5 failures about `wrap` prop not existing / classes not matching.

- [ ] **Step 3: Update `segmented-control.tsx`**

Replace the interface and function signature to add `wrap`:

```typescript
interface SegmentedControlProps<T extends string> {
  options: SegmentedOption<T>[]
  value: T
  onChange: (value: T) => void
  variant?: 'flat' | 'card'
  wrap?: boolean
  className?: string
}

function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  variant = 'flat',
  wrap = false,
  className,
}: SegmentedControlProps<T>) {
```

Replace the flat variant's `return` block (leave the card block untouched):

```typescript
  return (
    <div
      data-slot="segmented-control"
      data-variant="flat"
      className={cn('flex gap-2', wrap && 'flex-wrap', className)}
    >
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          onClick={() => onChange(opt.value)}
          className={cn(
            !wrap && 'flex-1',
            'rounded px-2 py-1.5 text-xs transition-colors',
            opt.value === value
              ? 'bg-brand text-brand-foreground font-medium'
              : 'bg-surface text-subtle hover:bg-surface/80',
          )}
          aria-pressed={opt.value === value}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
```

Also update the export to include `wrap` in the type export (already included via the interface — no extra change needed).

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd apps/web && npx vitest run src/components/ui/segmented-control.test.tsx
```

Expected: all tests PASS (original tests + 5 new wrap tests).

- [ ] **Step 5: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/ui/segmented-control.tsx apps/web/src/components/ui/segmented-control.test.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
feat(ui): add wrap prop to SegmentedControl

When wrap=true the flat variant container gains flex-wrap and buttons
drop flex-1, letting options reflow across rows for dynamic or large
option sets.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task B: Add `ToggleGroup` primitive

**Files:**
- Create: `apps/web/src/components/ui/toggle-group.tsx`
- Create: `apps/web/src/components/ui/toggle-group.test.tsx`

- [ ] **Step 1: Write the failing test file**

Create `apps/web/src/components/ui/toggle-group.test.tsx`:

```typescript
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ToggleGroup } from './toggle-group';

const options = [
  { key: 'bold' as const, label: 'Negrito' },
  { key: 'italic' as const, label: 'Itálico' },
  { key: 'uppercase' as const, label: 'Maiúsculas' },
] as const;

type K = 'bold' | 'italic' | 'uppercase';

const allOff: Record<K, boolean> = { bold: false, italic: false, uppercase: false };

describe('ToggleGroup', () => {
  it('renders every option label', () => {
    render(
      <ToggleGroup options={options} value={allOff} onChange={() => {}} />,
    );
    expect(screen.getByText('Negrito')).toBeInTheDocument();
    expect(screen.getByText('Itálico')).toBeInTheDocument();
    expect(screen.getByText('Maiúsculas')).toBeInTheDocument();
  });

  it('active option has bg-brand class', () => {
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, bold: true }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText('Negrito').closest('button')!.className).toContain('bg-brand');
  });

  it('inactive option has bg-surface class', () => {
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, bold: true }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText('Itálico').closest('button')!.className).toContain('bg-surface');
  });

  it('aria-pressed tracks value map per key', () => {
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, italic: true }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText('Negrito').closest('button')).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByText('Itálico').closest('button')).toHaveAttribute('aria-pressed', 'true');
  });

  it('clicking active option calls onChange(key, false)', () => {
    const onChange = vi.fn();
    render(
      <ToggleGroup
        options={options}
        value={{ ...allOff, bold: true }}
        onChange={onChange}
      />,
    );
    fireEvent.click(screen.getByText('Negrito'));
    expect(onChange).toHaveBeenCalledWith('bold', false);
  });

  it('clicking inactive option calls onChange(key, true)', () => {
    const onChange = vi.fn();
    render(
      <ToggleGroup options={options} value={allOff} onChange={onChange} />,
    );
    fireEvent.click(screen.getByText('Itálico'));
    expect(onChange).toHaveBeenCalledWith('italic', true);
  });

  it('onChange only fires once per click (not for other keys)', () => {
    const onChange = vi.fn();
    render(
      <ToggleGroup options={options} value={allOff} onChange={onChange} />,
    );
    fireEvent.click(screen.getByText('Maiúsculas'));
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith('uppercase', true);
  });

  it('container has data-slot="toggle-group"', () => {
    render(<ToggleGroup options={options} value={allOff} onChange={() => {}} />);
    expect(document.querySelector('[data-slot="toggle-group"]')).toBeInTheDocument();
  });

  it('merges className on root', () => {
    render(
      <ToggleGroup
        options={options}
        value={allOff}
        onChange={() => {}}
        className="extra-class"
      />,
    );
    expect(document.querySelector('[data-slot="toggle-group"]')!.className).toContain('extra-class');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/toggle-group.test.tsx
```

Expected: FAIL — `Cannot find module './toggle-group'`.

- [ ] **Step 3: Write the implementation**

Create `apps/web/src/components/ui/toggle-group.tsx`:

```typescript
import * as React from 'react'
import { cn } from '@/lib/utils'

interface ToggleGroupOption<K extends string> {
  key: K
  label: string
}

interface ToggleGroupProps<K extends string> {
  options: ToggleGroupOption<K>[]
  value: Record<K, boolean>
  onChange: (key: K, next: boolean) => void
  className?: string
}

function ToggleGroup<K extends string>({
  options,
  value,
  onChange,
  className,
}: ToggleGroupProps<K>) {
  return (
    <div
      data-slot="toggle-group"
      className={cn('flex gap-2', className)}
    >
      {options.map((opt) => {
        const active = value[opt.key]
        return (
          <button
            key={opt.key}
            type="button"
            onClick={() => onChange(opt.key, !active)}
            aria-pressed={active}
            className={cn(
              'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
              active
                ? 'bg-brand text-brand-foreground font-medium'
                : 'bg-surface text-subtle hover:bg-surface/80',
            )}
          >
            {opt.label}
          </button>
        )
      })}
    </div>
  )
}

export { ToggleGroup }
export type { ToggleGroupProps, ToggleGroupOption }
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd apps/web && npx vitest run src/components/ui/toggle-group.test.tsx
```

Expected: all 9 tests PASS.

- [ ] **Step 5: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/ui/toggle-group.tsx apps/web/src/components/ui/toggle-group.test.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
feat(ui): add ToggleGroup primitive

Independent multi-select toggle group: each option is an independently
toggleable boolean keyed by generic string. Mirrors SegmentedControl
flat styling but with Record<K,boolean> value contract.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task C: Add `InputWithAction` primitive

**Files:**
- Create: `apps/web/src/components/ui/input-with-action.tsx`
- Create: `apps/web/src/components/ui/input-with-action.test.tsx`

- [ ] **Step 1: Write the failing test file**

Create `apps/web/src/components/ui/input-with-action.test.tsx`:

```typescript
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/input-with-action.test.tsx
```

Expected: FAIL — `Cannot find module './input-with-action'`.

- [ ] **Step 3: Write the implementation**

Create `apps/web/src/components/ui/input-with-action.tsx`:

```typescript
import * as React from 'react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface InputWithActionProps {
  value: string
  onValueChange: (value: string) => void
  onSubmit: () => void
  actionLabel: React.ReactNode
  actionDisabled?: boolean
  placeholder?: string
  inputClassName?: string
  actionVariant?: React.ComponentProps<typeof Button>['variant']
  actionSize?: React.ComponentProps<typeof Button>['size']
  className?: string
}

function InputWithAction({
  value,
  onValueChange,
  onSubmit,
  actionLabel,
  actionDisabled,
  placeholder,
  inputClassName,
  actionVariant = 'outline',
  actionSize = 'sm',
  className,
}: InputWithActionProps) {
  return (
    <div
      data-slot="input-with-action"
      className={cn('flex gap-2', className)}
    >
      <Input
        type="text"
        value={value}
        onChange={(e) => onValueChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !actionDisabled) {
            onSubmit()
          }
        }}
        placeholder={placeholder}
        className={cn('flex-1', inputClassName)}
      />
      <Button
        type="button"
        onClick={onSubmit}
        disabled={actionDisabled}
        variant={actionVariant}
        size={actionSize}
        className="text-xs"
      >
        {actionLabel}
      </Button>
    </div>
  )
}

export { InputWithAction }
export type { InputWithActionProps }
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd apps/web && npx vitest run src/components/ui/input-with-action.test.tsx
```

Expected: all 12 tests PASS.

- [ ] **Step 5: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/ui/input-with-action.tsx apps/web/src/components/ui/input-with-action.test.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
feat(ui): add InputWithAction primitive

Paired Input + trailing Button with Enter-to-submit semantics.
actionDisabled guards both the button click and the Enter key.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task D: Add `PanelHeader` primitive

**Files:**
- Create: `apps/web/src/components/ui/panel-header.tsx`
- Create: `apps/web/src/components/ui/panel-header.test.tsx`

- [ ] **Step 1: Write the failing test file**

Create `apps/web/src/components/ui/panel-header.test.tsx`:

```typescript
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PanelHeader } from './panel-header';

describe('PanelHeader', () => {
  it('renders title as h3', () => {
    render(<PanelHeader title="Preview" />);
    const heading = screen.getByRole('heading', { level: 3, name: 'Preview' });
    expect(heading).toBeInTheDocument();
  });

  it('h3 has text-sm font-medium text-prose classes', () => {
    render(<PanelHeader title="Preview" />);
    const h3 = screen.getByRole('heading', { level: 3 });
    expect(h3.className).toContain('text-sm');
    expect(h3.className).toContain('font-medium');
    expect(h3.className).toContain('text-prose');
  });

  it('renders ReactNode title (not just string)', () => {
    render(<PanelHeader title={<span data-testid="custom-title">Rich</span>} />);
    expect(screen.getByTestId('custom-title')).toBeInTheDocument();
  });

  it('renders actions when provided', () => {
    render(<PanelHeader title="Preview" actions={<button>Click</button>} />);
    expect(screen.getByRole('button', { name: 'Click' })).toBeInTheDocument();
  });

  it('action wrapper absent when actions not provided', () => {
    render(<PanelHeader title="Preview" />);
    expect(document.querySelector('[data-slot="panel-header"]')!.children).toHaveLength(1);
  });

  it('root has data-slot="panel-header"', () => {
    render(<PanelHeader title="Preview" />);
    expect(document.querySelector('[data-slot="panel-header"]')).toBeInTheDocument();
  });

  it('root has flex items-center justify-between classes', () => {
    render(<PanelHeader title="Preview" />);
    const el = document.querySelector('[data-slot="panel-header"]')!;
    expect(el.className).toContain('flex');
    expect(el.className).toContain('items-center');
    expect(el.className).toContain('justify-between');
  });

  it('className merges on root', () => {
    render(<PanelHeader title="Preview" className="extra-class" />);
    expect(document.querySelector('[data-slot="panel-header"]')!.className).toContain('extra-class');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/web && npx vitest run src/components/ui/panel-header.test.tsx
```

Expected: FAIL — `Cannot find module './panel-header'`.

- [ ] **Step 3: Write the implementation**

Create `apps/web/src/components/ui/panel-header.tsx`:

```typescript
import * as React from 'react'
import { cn } from '@/lib/utils'

interface PanelHeaderProps {
  title: React.ReactNode
  actions?: React.ReactNode
  className?: string
}

function PanelHeader({ title, actions, className }: PanelHeaderProps) {
  return (
    <div
      data-slot="panel-header"
      className={cn('flex items-center justify-between', className)}
    >
      <h3 className="text-sm font-medium text-prose">{title}</h3>
      {actions && <div className="shrink-0">{actions}</div>}
    </div>
  )
}

export { PanelHeader }
export type { PanelHeaderProps }
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd apps/web && npx vitest run src/components/ui/panel-header.test.tsx
```

Expected: all 8 tests PASS.

- [ ] **Step 5: Run full test suite (pre-migration baseline)**

```bash
cd apps/web && npx vitest run
```

Expected: all tests PASS. This is the green baseline before consumer migrations.

- [ ] **Step 6: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/ui/panel-header.tsx apps/web/src/components/ui/panel-header.test.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
feat(ui): add PanelHeader primitive

Title-left / actions-right header row for inline cards. Wraps
<h3 class="text-sm font-medium text-prose"> + optional shrink-0 action
slot. Not SectionPanel — zero container semantics.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task E: Adopt `SegmentedControl wrap` in StepMode (5 sites)

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`

These 5 sites each share the same structure:
```tsx
<div className="flex gap-2 flex-wrap">  {/* or flex flex-wrap gap-2 */}
  {/* button map */}
</div>
```
Replace each with `<SegmentedControl wrap ... />`. `ToggleGroup` import is NOT needed yet (Task F). No new imports required — `SegmentedControl` is already imported in `StepMode.tsx`.

### Site 1 — Target Clip Duration (AI mode, search for `Target Clip Duration`)

**Find this block** (inside the AI Configuration section):

```tsx
<div className="flex gap-2 flex-wrap">
  {(() => {
    const MIN_PART = 60; // 1 min (AI mode allows shorter clips than Long Form's 8 min)
    const totalSec = videoInfo?.durationSec ?? 0;
    const suggestions: [number, string][] = [];
    for (const parts of [3, 2, 1]) {
      if (totalSec === 0) break;
      const partSec = Math.round(totalSec / parts / 60) * 60;
      if (partSec >= MIN_PART) {
        const mins = Math.round(partSec / 60);
        suggestions.push([partSec, `${mins} min`]);
      }
    }
    const options: [number, string][] = suggestions.length > 0
      ? suggestions
      : [[1020, '17 min'], [1560, '26 min'], [3120, '52 min']];
    return options.map(([secs, label]) => (
      <button
        key={secs}
        onClick={() => setClipDurationSecs(secs)}
        className={[
          'rounded px-2 py-1.5 text-xs transition-colors',
          clipDurationSecs === secs
            ? 'bg-brand text-brand-foreground font-medium'
            : 'bg-surface text-subtle hover:bg-surface/80',
        ].join(' ')}
      >
        {label}
      </button>
    ));
  })()}
</div>
```

**Replace with:**

```tsx
{(() => {
  const MIN_PART = 60; // 1 min (AI mode allows shorter clips than Long Form's 8 min)
  const totalSec = videoInfo?.durationSec ?? 0;
  const suggestions: Array<[number, string]> = [];
  for (const parts of [3, 2, 1]) {
    if (totalSec === 0) break;
    const partSec = Math.round(totalSec / parts / 60) * 60;
    if (partSec >= MIN_PART) {
      const mins = Math.round(partSec / 60);
      suggestions.push([partSec, `${mins} min`]);
    }
  }
  const raw: Array<[number, string]> = suggestions.length > 0
    ? suggestions
    : [[1020, '17 min'], [1560, '26 min'], [3120, '52 min']];
  return (
    <SegmentedControl<string>
      wrap
      value={String(clipDurationSecs)}
      onChange={(v) => setClipDurationSecs(Number(v))}
      options={raw.map(([secs, label]) => ({ value: String(secs), label }))}
    />
  );
})()}
```

### Site 2 — Minimum Clip Duration (AI mode, search for `Minimum Clip Duration`)

**Find:**

```tsx
<div className="flex gap-2 flex-wrap">
  {([60, 120, 300, 480, 600, 900] as const).map((secs) => (
    <button
      key={secs}
      onClick={() => setMinDurationSecs(secs)}
      className={[
        'rounded px-2 py-1.5 text-xs transition-colors',
        minDurationSecs === secs
          ? 'bg-brand text-brand-foreground font-medium'
          : 'bg-surface text-subtle hover:bg-surface/80',
      ].join(' ')}
    >
      {secs / 60} min
    </button>
  ))}
</div>
```

**Replace with:**

```tsx
<SegmentedControl<string>
  wrap
  value={String(minDurationSecs)}
  onChange={(v) => setMinDurationSecs(Number(v))}
  options={[
    { value: '60',  label: '1 min' },
    { value: '120', label: '2 min' },
    { value: '300', label: '5 min' },
    { value: '480', label: '8 min' },
    { value: '600', label: '10 min' },
    { value: '900', label: '15 min' },
  ]}
/>
```

### Site 3 — Segment Duration (Longform mode, search for `Segment Duration`)

**Find** (inside Long Form Configuration section):

```tsx
<div className="flex gap-2 flex-wrap">
  {(() => {
    const MIN_PART = 480; // 8 min
    const totalSec = videoInfo?.durationSec ?? 0;
    const suggestions: [number, string][] = [];
    for (const parts of [3, 2, 1]) {
      if (totalSec === 0) break;
      const partSec = Math.round(totalSec / parts / 60) * 60;
      if (partSec >= MIN_PART) {
        const mins = Math.round(partSec / 60);
        suggestions.push([partSec, `${mins} min`]);
      }
    }
    const options: [number, string][] = suggestions.length > 0
      ? suggestions
      : [[1020, '17 min'], [1560, '26 min'], [3120, '52 min']];
    return options.map(([secs, label]) => (
      <button
        key={secs}
        onClick={() => setSegmentSecs(secs)}
        className={[
          'rounded px-2 py-1.5 text-xs transition-colors',
          segmentSecs === secs
            ? 'bg-brand text-brand-foreground font-medium'
            : 'bg-surface text-subtle hover:bg-surface/80',
        ].join(' ')}
      >
        {label}
      </button>
    ));
  })()}
</div>
```

**Replace with:**

```tsx
{(() => {
  const MIN_PART = 480; // 8 min
  const totalSec = videoInfo?.durationSec ?? 0;
  const suggestions: Array<[number, string]> = [];
  for (const parts of [3, 2, 1]) {
    if (totalSec === 0) break;
    const partSec = Math.round(totalSec / parts / 60) * 60;
    if (partSec >= MIN_PART) {
      const mins = Math.round(partSec / 60);
      suggestions.push([partSec, `${mins} min`]);
    }
  }
  const raw: Array<[number, string]> = suggestions.length > 0
    ? suggestions
    : [[1020, '17 min'], [1560, '26 min'], [3120, '52 min']];
  return (
    <SegmentedControl<string>
      wrap
      value={String(segmentSecs)}
      onChange={(v) => setSegmentSecs(Number(v))}
      options={raw.map(([secs, label]) => ({ value: String(secs), label }))}
    />
  );
})()}
```

### Site 4 — Min Part Duration (Longform mode, search for `Min Part Duration`)

**Find:**

```tsx
<div className="flex flex-wrap gap-2">
  {([
    [0, 'Off'],
    [60, '1 min'],
    [120, '2 min'],
    [300, '5 min'],
    [480, '8 min'],
    [600, '10 min'],
    [900, '15 min'],
  ] as [number, string][]).map(([secs, label]) => (
    <button
      key={secs}
      onClick={() => setMinPartSecs(secs)}
      className={[
        'rounded px-2 py-1.5 text-xs transition-colors',
        minPartSecs === secs
          ? 'bg-brand text-brand-foreground font-medium'
          : 'bg-surface text-subtle hover:bg-surface/80',
      ].join(' ')}
    >
      {label}
    </button>
  ))}
</div>
```

**Replace with:**

```tsx
<SegmentedControl<string>
  wrap
  value={String(minPartSecs)}
  onChange={(v) => setMinPartSecs(Number(v))}
  options={[
    { value: '0',   label: 'Off' },
    { value: '60',  label: '1 min' },
    { value: '120', label: '2 min' },
    { value: '300', label: '5 min' },
    { value: '480', label: '8 min' },
    { value: '600', label: '10 min' },
    { value: '900', label: '15 min' },
  ]}
/>
```

### Site 5 — Chroma Key (search for `Chroma Key`)

**Find:**

```tsx
<div className="flex flex-wrap gap-2">
  {(['none', 'green', 'black', 'white', 'blue'] as const).map((key) => (
    <button
      key={key}
      onClick={() => setOverlayChromaKey(key)}
      className={[
        'rounded px-3 py-1.5 text-xs capitalize transition-colors',
        overlayChromaKey === key
          ? 'bg-brand text-brand-foreground font-medium'
          : 'bg-surface text-subtle hover:bg-surface/80',
      ].join(' ')}
    >
      {key === 'none' ? 'Nenhum' : key}
    </button>
  ))}
</div>
```

**Replace with:**

```tsx
<SegmentedControl<'none' | 'green' | 'black' | 'white' | 'blue'>
  wrap
  value={overlayChromaKey}
  onChange={setOverlayChromaKey}
  options={[
    { value: 'none',  label: 'Nenhum' },
    { value: 'green', label: 'Green' },
    { value: 'black', label: 'Black' },
    { value: 'white', label: 'White' },
    { value: 'blue',  label: 'Blue' },
  ]}
/>
```

*(Visual drift accepted: `px-3`→`px-2`; `capitalize` removed; labels pre-capitalized.)*

- [ ] **Step 1: Apply all 5 replacements in `StepMode.tsx` as described above**

- [ ] **Step 2: Run tests**

```bash
cd apps/web && npx vitest run
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/pipeline/steps/StepMode.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
refactor(ui): adopt SegmentedControl wrap in StepMode

Replace 5 hand-rolled flex-wrap button groups with SegmentedControl
wrap=true: Target/Min Clip Duration (AI), Segment/Min Part Duration
(Longform), Chroma Key. Chroma Key labels pre-capitalized; px-3→px-2
drift accepted.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task F: Adopt `ToggleGroup` in StepMode captions (1 site)

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`

- [ ] **Step 1: Add `ToggleGroup` to the import block in `StepMode.tsx`**

Find the existing import line:
```typescript
import { SegmentedControl } from '@/components/ui/segmented-control';
```

Add the line immediately after it:
```typescript
import { ToggleGroup } from '@/components/ui/toggle-group';
```

- [ ] **Step 2: Replace the captions bold/italic/uppercase group**

Search for `captionsBold` in `StepMode.tsx`. Find this block (near `'Negrito'`):

```tsx
<div className="flex gap-2">
  {([
    ['bold', captionsBold, setCaptionsBold, 'Negrito'],
    ['italic', captionsItalic, setCaptionsItalic, 'Itálico'],
    ['uppercase', captionsUppercase, setCaptionsUppercase, 'Maiúsculas'],
  ] as [string, boolean, (v: boolean) => void, string][]).map(([key, val, setter, label]) => (
    <button
      key={key}
      onClick={() => setter(!val)}
      className={[
        'flex-1 rounded px-2 py-1.5 text-xs transition-colors',
        val ? 'bg-brand text-brand-foreground font-medium' : 'bg-surface text-subtle hover:bg-surface/80',
      ].join(' ')}
    >
      {label}
    </button>
  ))}
</div>
```

**Replace with:**

```tsx
<ToggleGroup<'bold' | 'italic' | 'uppercase'>
  options={[
    { key: 'bold',      label: 'Negrito' },
    { key: 'italic',    label: 'Itálico' },
    { key: 'uppercase', label: 'Maiúsculas' },
  ]}
  value={{ bold: captionsBold, italic: captionsItalic, uppercase: captionsUppercase }}
  onChange={(k, next) => {
    if (k === 'bold') setCaptionsBold(next);
    else if (k === 'italic') setCaptionsItalic(next);
    else setCaptionsUppercase(next);
  }}
/>
```

- [ ] **Step 3: Run tests**

```bash
cd apps/web && npx vitest run
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/pipeline/steps/StepMode.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
refactor(ui): adopt ToggleGroup in StepMode captions

Replace hand-rolled bold/italic/uppercase button group in the captions
section with <ToggleGroup>. Three independent boolean setters move into
a single onChange handler.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task G: Adopt `InputWithAction` in StepMode preset save + ChannelConfigSheet pattern add

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`
- Modify: `apps/web/src/components/channels/ChannelConfigSheet.tsx`

> **Behavior note:** In `ChannelConfigSheet`, the original `onKeyDown` fired `handleAdd()` even when `adding=true`. `InputWithAction` normalizes this: Enter respects `actionDisabled` (same as the button). This is a minor intentional normalization — double-submit while `adding=true` was unintentional in the original.

- [ ] **Step 1: Add `InputWithAction` import to `StepMode.tsx`**

Find the existing import line:
```typescript
import { Input } from '@/components/ui/input';
```

Add immediately after it:
```typescript
import { InputWithAction } from '@/components/ui/input-with-action';
```

- [ ] **Step 2: Replace the preset save row in `StepMode.tsx`**

Search for `Nome do preset`. Find:

```tsx
{(() => {
  const nameExists = presets.some((p) => p.name === presetName.trim());
  return (
    <div className="flex gap-2">
      <Input
        type="text"
        value={presetName}
        onChange={(e) => {
          setPresetName(e.target.value);
          setActivePresetName(presets.find((p) => p.name === e.target.value.trim())?.name ?? null);
        }}
        onKeyDown={(e) => { if (e.key === 'Enter' && presetName.trim()) void handleSavePreset(); }}
        placeholder="Nome do preset…"
        className="flex-1 h-8 text-xs"
      />
      <Button
        onClick={() => void handleSavePreset()}
        disabled={!presetName.trim()}
        variant="outline"
        size="sm"
        className="text-xs"
      >
        {nameExists ? 'Atualizar' : 'Criar'}
      </Button>
    </div>
  );
})()}
```

**Replace with:**

```tsx
{(() => {
  const nameExists = presets.some((p) => p.name === presetName.trim());
  return (
    <InputWithAction
      value={presetName}
      onValueChange={(v) => {
        setPresetName(v);
        setActivePresetName(presets.find((p) => p.name === v.trim())?.name ?? null);
      }}
      onSubmit={() => void handleSavePreset()}
      actionDisabled={!presetName.trim()}
      actionLabel={nameExists ? 'Atualizar' : 'Criar'}
      placeholder="Nome do preset…"
      inputClassName="h-8 text-xs"
    />
  );
})()}
```

- [ ] **Step 3: Add `InputWithAction` import to `ChannelConfigSheet.tsx`**

Find the existing import line:
```typescript
import { Input } from '@/components/ui/input';
```

Add immediately after it:
```typescript
import { InputWithAction } from '@/components/ui/input-with-action';
```

- [ ] **Step 4: Replace the pattern add row in `ChannelConfigSheet.tsx`**

Search for `artist name or pattern`. Find:

```tsx
<div className="flex gap-2">
  <Input
    value={newPattern}
    onChange={(e) => setNewPattern(e.target.value)}
    placeholder="artist name or pattern…"
    onKeyDown={(e) => {
      if (e.key === 'Enter') void handleAdd();
    }}
  />
  <Button
    size="sm"
    onClick={() => void handleAdd()}
    disabled={adding || !newPattern.trim()}
  >
    {adding ? 'Adding…' : 'Add'}
  </Button>
</div>
```

**Replace with:**

```tsx
<InputWithAction
  value={newPattern}
  onValueChange={setNewPattern}
  onSubmit={() => void handleAdd()}
  actionDisabled={adding || !newPattern.trim()}
  actionLabel={adding ? 'Adding…' : 'Add'}
  placeholder="artist name or pattern…"
/>
```

- [ ] **Step 5: Run tests**

```bash
cd apps/web && npx vitest run
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/pipeline/steps/StepMode.tsx apps/web/src/components/channels/ChannelConfigSheet.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
refactor(ui,channels): adopt InputWithAction

Preset save row (StepMode) and pattern add row (ChannelConfigSheet)
both migrated to <InputWithAction>. Enter-to-submit now consistently
respects actionDisabled (minor normalization of original ChannelConfig
behavior where Enter fired even while adding=true).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task H: Adopt `PanelHeader` in StepMode Preview (2 sites)

**Files:**
- Modify: `apps/web/src/components/pipeline/steps/StepMode.tsx`

- [ ] **Step 1: Add `PanelHeader` import to `StepMode.tsx`**

Find the import line (add after any existing `@/components/ui/` import):
```typescript
import { Label } from '@/components/ui/label';
```

Add immediately after it:
```typescript
import { PanelHeader } from '@/components/ui/panel-header';
```

- [ ] **Step 2: Replace the downloading-state Preview header (Site 1)**

Search for `Aguardando download...`. Find the `<>` fragment that opens the downloading state — look for:

```tsx
<div className="flex items-center justify-between">
  <h3 className="text-sm font-medium text-prose">Preview</h3>
  <span className="text-xs text-subtle">Aguardando download...</span>
</div>
```

**Replace with:**

```tsx
<PanelHeader
  title="Preview"
  actions={<span className="text-xs text-subtle">Aguardando download...</span>}
/>
```

- [ ] **Step 3: Replace the ready-state Preview header (Site 2)**

Search for `downloadComplete && (` to find the ready state block. Inside it, find:

```tsx
<div className="flex items-center justify-between">
  <h3 className="text-sm font-medium text-prose">Preview</h3>
  <Button
    onClick={() => {
      if (!activeRunId) return;
      const cfg = buildModeConfig();
      void generatePreview(goUrl, activeRunId, cfg);
    }}
    disabled={previewStatus === 'generating'}
    variant="secondary"
    size="sm"
    className="text-xs"
  >
    {previewStatus === 'generating' ? 'Generating…' : previewStatus === 'ready' ? 'Regenerate Preview' : 'Generate Preview'}
  </Button>
</div>
```

**Replace with:**

```tsx
<PanelHeader
  title="Preview"
  actions={
    <Button
      onClick={() => {
        if (!activeRunId) return;
        const cfg = buildModeConfig();
        void generatePreview(goUrl, activeRunId, cfg);
      }}
      disabled={previewStatus === 'generating'}
      variant="secondary"
      size="sm"
      className="text-xs"
    >
      {previewStatus === 'generating' ? 'Generating…' : previewStatus === 'ready' ? 'Regenerate Preview' : 'Generate Preview'}
    </Button>
  }
/>
```

- [ ] **Step 4: Run full test suite**

```bash
cd apps/web && npx vitest run
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut add apps/web/src/components/pipeline/steps/StepMode.tsx
git -C /Users/joaogabriel/Projects/YoutubeProjects/AutoCut commit -m "$(cat <<'EOF'
refactor(ui): adopt PanelHeader in StepMode Preview

Replace both Preview card header rows (download-waiting and
download-ready states) with <PanelHeader title="Preview" actions=…/>.
Card wrapper and all nested content unchanged.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Final Verification

- [ ] **Run full test suite one last time**

```bash
cd apps/web && npx vitest run
```

Expected: all tests PASS.

- [ ] **Coordinator: run `npx next build` after each commit** (external step — not done by the agent)

- [ ] **Grep check — expect 0 hits outside primitives and tests:**

```bash
cd apps/web && grep -rn \
  'flex gap-2 flex-wrap\|flex flex-wrap gap-2\|flex-wrap gap-2' \
  src/components/pipeline/steps/StepMode.tsx
```

Expected: 0 hits.

```bash
cd apps/web && grep -rn \
  "'flex-1 rounded px-2 py-1.5 text-xs" \
  src/components/pipeline/steps/StepMode.tsx
```

Expected: 0 hits.

- [ ] **Manual spot-check (coordinator):**

Run `npm run dev` from `AutoCut/`. Open the pipeline in the browser and verify:
1. **AI mode** — Target Clip Duration and Minimum Clip Duration option buttons wrap and reflect selection.
2. **Longform mode** — Segment Duration and Min Part Duration buttons wrap and reflect selection.
3. **Overlay config** — Chroma Key options render with correct capitalized labels; "Nenhum" appears for `none`.
4. **Captions section** — Negrito/Itálico/Maiúsculas toggle independently; all three can be on simultaneously.
5. **Preset save row** — typing + Enter saves; button label toggles Criar/Atualizar; `h-8 text-xs` input height preserved visually.
6. **Preview card** — "Aguardando download..." shows span on right; once download completes, button appears on right with correct Generate/Regenerate label.
7. **ChannelConfigSheet** — blacklist pattern add row works; button shows "Adding…" during submit.
