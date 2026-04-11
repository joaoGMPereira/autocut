'use client';

import { Brain, Volume2, Eye, MessageSquare } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Strategy {
  id: string;
  label: string;
  description: string;
  icon: React.ReactNode;
}

const STRATEGIES: Strategy[] = [
  {
    id: 'ai',
    label: 'IA (Ollama)',
    description: 'Análise semântica do conteúdo',
    icon: <Brain className="h-4 w-4" />,
  },
  {
    id: 'audio',
    label: 'Áudio',
    description: 'Picos de volume e silences',
    icon: <Volume2 className="h-4 w-4" />,
  },
  {
    id: 'visual',
    label: 'Visual',
    description: 'Mudanças de cena',
    icon: <Eye className="h-4 w-4" />,
  },
  {
    id: 'chat',
    label: 'Chat Twitch',
    description: 'Atividade de emotes e caps',
    icon: <MessageSquare className="h-4 w-4" />,
  },
];

interface HighlightStrategySelectorProps {
  selected: string[];
  chatJsonPath: string;
  onToggle: (id: string) => void;
  onChatJsonPathChange: (p: string) => void;
}

export function HighlightStrategySelector({
  selected,
  chatJsonPath,
  onToggle,
  onChatJsonPathChange,
}: HighlightStrategySelectorProps) {
  const showChatInput = selected.includes('chat');

  return (
    <div className="flex flex-col gap-3">
      <span className="text-xs text-zinc-400 font-medium">Strategies de Detecção</span>
      <div className="grid grid-cols-2 gap-2">
        {STRATEGIES.map((s) => {
          const isSelected = selected.includes(s.id);
          return (
            <button
              key={s.id}
              type="button"
              onClick={() => onToggle(s.id)}
              className={cn(
                'flex items-start gap-2 rounded-lg border p-3 text-left transition-colors',
                'hover:border-blue-500/60 hover:bg-blue-500/5',
                isSelected
                  ? 'border-blue-500 bg-blue-500/10'
                  : 'border-zinc-700 bg-transparent',
              )}
            >
              <span
                className={cn(
                  'mt-0.5 shrink-0',
                  isSelected ? 'text-blue-400' : 'text-zinc-500',
                )}
              >
                {s.icon}
              </span>
              <div className="flex flex-col gap-0.5">
                <span
                  className={cn(
                    'text-xs font-medium',
                    isSelected ? 'text-blue-300' : 'text-zinc-300',
                  )}
                >
                  {s.label}
                </span>
                <span className="text-[11px] text-zinc-500 leading-snug">{s.description}</span>
              </div>
            </button>
          );
        })}
      </div>

      {showChatInput && (
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-zinc-400">Chat JSON Path</span>
          <input
            type="text"
            value={chatJsonPath}
            onChange={(e) => onChatJsonPathChange(e.target.value)}
            placeholder="/path/to/chat_export.json"
            className="w-full rounded-lg bg-zinc-900 border border-zinc-700 px-3 py-2 text-sm focus:outline-none focus:border-blue-500/60 placeholder:text-zinc-600 text-zinc-200"
          />
        </label>
      )}
    </div>
  );
}
