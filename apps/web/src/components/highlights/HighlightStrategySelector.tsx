'use client';

import { Brain, Volume2, Eye, MessageSquare } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
      <span className="text-xs text-subtle font-medium">Strategies de Detecção</span>
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
                'hover:border-info/60 hover:bg-info/5',
                isSelected
                  ? 'border-info bg-info/10'
                  : 'border-border bg-transparent',
              )}
            >
              <span
                className={cn(
                  'mt-0.5 shrink-0',
                  isSelected ? 'text-info' : 'text-subtle',
                )}
              >
                {s.icon}
              </span>
              <div className="flex flex-col gap-0.5">
                <span
                  className={cn(
                    'text-xs font-medium',
                    isSelected ? 'text-info' : 'text-prose',
                  )}
                >
                  {s.label}
                </span>
                <span className="text-[11px] text-subtle leading-snug">{s.description}</span>
              </div>
            </button>
          );
        })}
      </div>

      {showChatInput && (
        <div className="flex flex-col gap-1.5">
          <Label className="text-xs font-medium text-subtle">Chat JSON Path</Label>
          <Input
            type="text"
            value={chatJsonPath}
            onChange={(e) => onChatJsonPathChange(e.target.value)}
            placeholder="/path/to/chat_export.json"
          />
        </div>
      )}
    </div>
  );
}
