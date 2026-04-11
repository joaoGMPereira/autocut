'use client';

import { useEffect } from 'react';
import { useAppStore } from '@/store/appStore';
import { useDispatcherStore } from '@/store/dispatcherStore';

const UPDATE_KEY = 'update:latest';

/**
 * Bridges appStore.updateStatus into the generic dispatcherStore as an
 * `update:latest` task, so DispatcherBar can render update notifications
 * alongside download tasks without any coupling between the two.
 */
export function useUpdateDispatcher() {
  const updateStatus = useAppStore((s) => s.updateStatus);
  const { upsertTask, setProgress, markDone, markError, removeTask } =
    useDispatcherStore();

  useEffect(() => {
    switch (updateStatus.status) {
      case 'idle':
      case 'not-available':
        removeTask(UPDATE_KEY);
        break;

      case 'checking':
        upsertTask(UPDATE_KEY, {
          status: 'running',
          progress: 0,
          data: { label: 'Verificando atualizações...' },
        });
        break;

      case 'available':
        upsertTask(UPDATE_KEY, {
          status: 'running',
          progress: 0,
          data: { label: 'Baixando atualização...' },
        });
        break;

      case 'downloading':
        upsertTask(UPDATE_KEY, {
          status: 'running',
          progress: updateStatus.percent ?? 0,
          data: {
            label: 'Baixando atualização...',
            bytesPerSecond: updateStatus.bytesPerSecond,
          },
        });
        if (updateStatus.percent != null) {
          setProgress(UPDATE_KEY, updateStatus.percent);
        }
        break;

      case 'downloaded':
        markDone(UPDATE_KEY, {
          label: 'Atualização pronta!',
          action: 'restart',
          version: updateStatus.updateInfo?.version,
        });
        break;

      case 'error':
        markError(UPDATE_KEY, updateStatus.error ?? 'Erro desconhecido');
        break;
    }
  }, [updateStatus, upsertTask, setProgress, markDone, markError, removeTask]);
}
