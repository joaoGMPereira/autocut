'use client';

import { useProcessorStore } from '@/store/processorStore';
import { ShortCard } from './ShortCard';

interface ShortsResultListProps {
  jobId: string;
  language: string;
}

export function ShortsResultList({ jobId, language }: ShortsResultListProps) {
  const shortsResults = useProcessorStore((s) => s.shortsResults);
  const shortsJobs = useProcessorStore((s) => s.shortsJobs);

  const raw = shortsResults[jobId] ?? [];
  const isDone = shortsJobs[jobId]?.status === 'done';

  const results = isDone
    ? [...raw].sort((a, b) => b.engagement_score - a.engagement_score)
    : raw;

  if (results.length === 0) {
    return (
      <p className="text-xs text-subtle text-center py-4">
        No shorts yet…
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-subtle">
        {results.length} short{results.length !== 1 ? 's' : ''}{isDone ? ' (sorted by score)' : ' (in progress…)'}
      </p>
      {results.map((r, i) => (
        <ShortCard key={`${jobId}-${i}`} result={r} language={language} />
      ))}
    </div>
  );
}
