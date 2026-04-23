'use client';

import { useEffect, useState } from 'react';
import { useDispatcherStore, type DispatcherTask } from '@/store/dispatcherStore';
import { usePipelineStore } from '@/store/pipelineStore';
import { Button } from '@/components/ui/button';

// ─── Per-task pill ────────────────────────────────────────────────────────────

interface TaskPillProps {
  task: DispatcherTask;
  onDone: () => void;
}

function DownloadPill({ task, onDone }: TaskPillProps) {
  useEffect(() => {
    if (task.status === 'done') {
      const t = setTimeout(onDone, 3000);
      return () => clearTimeout(t);
    }
  }, [task.status, onDone]);

  return (
    <div
      className={`flex items-center gap-3 rounded-lg border px-4 py-2 text-xs font-medium ${
        task.status === 'done'
          ? 'border-success-alt/40 bg-success-alt/10 text-success-alt'
          : task.status === 'error'
            ? 'border-destructive/40 bg-destructive/10 text-destructive'
            : 'border-brand/40 bg-brand/10 text-brand'
      }`}
    >
      {task.status === 'done' && <span>✓ Download concluído</span>}
      {task.status === 'error' && <span>✗ Erro no download: {task.error}</span>}
      {task.status === 'running' && (
        <>
          <span>⬇ Baixando vídeo...</span>
          <div className="max-w-[120px] flex-1 overflow-hidden rounded-full bg-brand/20 h-1">
            {task.progress > 0 ? (
              <div
                className="h-full rounded-full bg-brand transition-all duration-300"
                style={{ width: `${task.progress}%` }}
              />
            ) : (
              <div className="h-full w-full animate-pulse rounded-full bg-brand/60" />
            )}
          </div>
          {task.progress > 0 && <span>{Math.round(task.progress)}%</span>}
        </>
      )}
    </div>
  );
}

function UpdatePill({ task }: { task: DispatcherTask }) {
  const label = String(task.data.label ?? 'Atualizando...');
  const action = task.data.action as string | undefined;
  const version = task.data.version as string | undefined;

  const handleRestart = () => {
    const api = (window as unknown as Record<string, unknown>).electronAPI as
      | { updateAndRestart?: () => void }
      | undefined;
    api?.updateAndRestart?.();
  };

  return (
    <div
      className={`flex items-center gap-3 rounded-lg border px-4 py-2 text-xs font-medium ${
        task.status === 'done' && action === 'restart'
          ? 'border-success/40 bg-success/10 text-success'
          : task.status === 'error'
            ? 'border-destructive/40 bg-destructive/10 text-destructive'
            : 'border-violet/40 bg-violet/10 text-violet'
      }`}
    >
      {task.status === 'error' && <span>✗ Erro na atualização: {task.error}</span>}

      {task.status === 'done' && action === 'restart' && (
        <>
          <span>↑ {label}{version ? ` v${version}` : ''}</span>
          <Button
            onClick={handleRestart}
            variant="outline"
            size="xs"
            className="ml-1 border-success/40 bg-success/20 text-success hover:bg-success/30"
          >
            Reiniciar
          </Button>
        </>
      )}

      {task.status === 'running' && (
        <>
          <span>↑ {label}</span>
          <div className="max-w-[120px] flex-1 overflow-hidden rounded-full bg-violet/20 h-1">
            {task.progress > 0 ? (
              <div
                className="h-full rounded-full bg-violet transition-all duration-300"
                style={{ width: `${task.progress}%` }}
              />
            ) : (
              <div className="h-full w-full animate-pulse rounded-full bg-violet/60" />
            )}
          </div>
          {task.progress > 0 && <span>{Math.round(task.progress)}%</span>}
        </>
      )}
    </div>
  );
}

// ─── Tool error pill ──────────────────────────────────────────────────────────

const TOOL_LABELS: Record<string, string> = {
  ffmpeg: 'FFmpeg',
  whisper: 'Whisper',
  ollama: 'Ollama',
};

function ToolErrorPill({ task, onRemove }: GenericPillProps) {
  const tool = String(task.data.tool ?? '');
  const op = String(task.data.op ?? '');
  const toolLabel = TOOL_LABELS[tool] ?? tool;
  const opLabel = op ? ` · ${op}` : '';

  return (
    <div className="flex items-center gap-3 rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-2 text-xs font-medium text-destructive">
      <span className="shrink-0">✗</span>
      <span className="flex-1 truncate">
        <span className="font-semibold">{toolLabel}</span>
        {opLabel} falhou
        {task.error ? <span className="opacity-70">: {task.error}</span> : null}
      </span>
      <Button
        onClick={() => onRemove(task.key)}
        variant="ghost"
        size="xs"
        className="ml-1 shrink-0 text-destructive/60 hover:bg-destructive/20 hover:text-destructive"
        aria-label="Fechar"
      >
        ×
      </Button>
    </div>
  );
}

// ─── Generic pill router ──────────────────────────────────────────────────────

interface GenericPillProps {
  task: DispatcherTask;
  onRemove: (key: string) => void;
}

function TaskPill({ task, onRemove }: GenericPillProps) {
  const handleDone = () => onRemove(task.key);

  if (task.key.startsWith('update:')) {
    return <UpdatePill task={task} />;
  }

  if (task.key.startsWith('download:')) {
    return <DownloadPill task={task} onDone={handleDone} />;
  }

  if (task.key.startsWith('tool_error:')) {
    return <ToolErrorPill task={task} onRemove={onRemove} />;
  }

  // Generic fallback for future task types
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border/40 bg-muted/10 px-4 py-2 text-xs font-medium text-subtle">
      {task.status === 'error' && <span>✗ {task.error}</span>}
      {task.status === 'done' && <span>✓ {String(task.data.label ?? task.key)}</span>}
      {task.status === 'running' && (
        <>
          <span>{String(task.data.label ?? task.key)}</span>
          {task.progress > 0 && (
            <div className="max-w-[80px] flex-1 overflow-hidden rounded-full bg-muted/40 h-1">
              <div
                className="h-full rounded-full bg-subtle/60 transition-all duration-300"
                style={{ width: `${task.progress}%` }}
              />
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ─── Main bar ─────────────────────────────────────────────────────────────────

/**
 * Generic dispatcher notification bar. Renders all active tasks from the
 * dispatcher store. Each pill is routed by key prefix:
 *   - `download:*` → download progress pill
 *   - `update:*`   → update / restart pill
 *   - anything else → generic label + progress pill
 *
 * Add new task types by dispatching to the store with a new key prefix and
 * rendering them here (or as a generic fallback automatically).
 */
export function DispatcherBar() {
  const tasks = useDispatcherStore((s) => s.tasks);
  const removeTask = useDispatcherStore((s) => s.removeTask);
  const activeRunId = usePipelineStore((s) => s.activeRunId);

  // Determine which tasks to show:
  // - download tasks: only for the active run
  // - update tasks: always show
  // - other tasks: always show
  const [hiddenKeys, setHiddenKeys] = useState<Set<string>>(new Set());

  const hide = (key: string) => setHiddenKeys((prev) => new Set([...prev, key]));
  const handleRemove = (key: string) => {
    hide(key);
    // Clean up from store after animation would finish
    setTimeout(() => removeTask(key), 300);
  };

  const visibleTasks = Object.values(tasks).filter((task) => {
    if (hiddenKeys.has(task.key)) return false;
    if (task.key.startsWith('download:')) {
      // Only show download task for the currently active run
      if (activeRunId == null) return false;
      if (task.key !== `download:${activeRunId}`) return false;
      return task.status === 'running' || task.status === 'error' || task.status === 'done';
    }
    // update:* and other types: show when running, error, or downloaded-with-action
    if (task.status === 'pending') return false;
    if (task.status === 'done' && task.key.startsWith('update:') && task.data.action === 'restart') {
      return true; // keep restart prompt visible until user acts
    }
    return task.status === 'running' || task.status === 'error';
  });

  if (visibleTasks.length === 0) return null;

  return (
    <div className="flex flex-col gap-2">
      {visibleTasks.map((task) => (
        <TaskPill key={task.key} task={task} onRemove={handleRemove} />
      ))}
    </div>
  );
}
