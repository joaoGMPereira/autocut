'use client';

import { useEffect, useState } from 'react';
import { usePipelineStore } from '@/store/pipelineStore';
import { useDispatcherStore } from '@/store/dispatcherStore';

export function DownloadProgressBar() {
  const activeRunId = usePipelineStore((s) => s.activeRunId);
  const task = useDispatcherStore((s) =>
    activeRunId != null ? s.tasks[`download:${activeRunId}`] : undefined,
  );
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (task?.status === 'done') {
      setVisible(true);
      const timer = setTimeout(() => setVisible(false), 3000);
      return () => clearTimeout(timer);
    } else if (task?.status === 'running' || task?.status === 'error') {
      setVisible(true);
    } else {
      setVisible(false);
    }
  }, [task?.status]);

  if (!visible || !task) return null;

  return (
    <div
      className={`flex items-center gap-3 rounded-lg border px-4 py-2 text-xs font-medium ${
        task.status === 'done'
          ? 'border-[#6BCB8B]/40 bg-[#6BCB8B]/10 text-[#6BCB8B]'
          : task.status === 'error'
            ? 'border-red-500/40 bg-red-500/10 text-red-400'
            : 'border-[#00D4FF]/40 bg-[#00D4FF]/10 text-[#00D4FF]'
      }`}
    >
      {task.status === 'done' && <span>✓ Download concluído</span>}
      {task.status === 'error' && <span>✗ Erro no download: {task.error}</span>}
      {task.status === 'running' && (
        <>
          <span>⬇ Baixando vídeo...</span>
          <div className="max-w-[120px] flex-1 overflow-hidden rounded-full bg-[#00D4FF]/20 h-1">
            {task.progress > 0 ? (
              <div
                className="h-full rounded-full bg-[#00D4FF] transition-all duration-300"
                style={{ width: `${task.progress}%` }}
              />
            ) : (
              <div className="h-full w-full animate-pulse rounded-full bg-[#00D4FF]/60" />
            )}
          </div>
          {task.progress > 0 && <span>{Math.round(task.progress)}%</span>}
        </>
      )}
    </div>
  );
}
