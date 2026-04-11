import { create } from 'zustand';
import type { ModeConfig } from '@/types/pipeline';

export interface WizardStepDef {
  id: string;
  label: string;
  hidden?: boolean;
}

const DEFAULT_STEPS: WizardStepDef[] = [
  { id: 'channel', label: 'Canal' },
  { id: 'url', label: 'URL' },
  { id: 'mode', label: 'Modo' },
  { id: 'execute', label: 'Executar' },
  { id: 'highlights', label: 'Highlights', hidden: true },
  { id: 'metadata', label: 'Metadados' },
  { id: 'clips', label: 'Cortes' },
  { id: 'upload', label: 'Upload' },
];

interface WizardState {
  currentStep: number;
  steps: WizardStepDef[];
  canAdvance: boolean;
  maxReached: number;

  // Pending data collected across steps before run creation
  pendingUrl: string;
  pendingModeConfig: ModeConfig | null;
  pendingChannelId: number | null;
  sourceRunId: number | null;

  visibleSteps: () => WizardStepDef[];
  currentStepDef: () => WizardStepDef | undefined;

  goNext: () => void;
  goBack: () => void;
  goToStep: (index: number) => void;
  jumpToStep: (index: number) => void;
  setCanAdvance: (can: boolean) => void;
  setStepHidden: (stepId: string, hidden: boolean) => void;
  setPendingUrl: (url: string) => void;
  setPendingModeConfig: (config: ModeConfig) => void;
  setPendingChannelId: (id: number | null) => void;
  setSourceRunId: (id: number | null) => void;
  resetWizard: () => void;
}

export const useWizardStore = create<WizardState>((set, get) => ({
  currentStep: 0,
  steps: DEFAULT_STEPS.map((s) => ({ ...s })),
  canAdvance: false,
  maxReached: 0,
  pendingUrl: '',
  pendingModeConfig: null,
  pendingChannelId: null,
  sourceRunId: null,

  visibleSteps: () => get().steps.filter((s) => !s.hidden),

  currentStepDef: () => {
    const { steps, currentStep } = get();
    return steps[currentStep];
  },

  goNext: () => {
    const { steps, currentStep, canAdvance } = get();
    if (!canAdvance) return;
    for (let i = currentStep + 1; i < steps.length; i++) {
      if (!steps[i].hidden) {
        set((state) => ({
          currentStep: i,
          canAdvance: false,
          maxReached: Math.max(state.maxReached, i),
        }));
        return;
      }
    }
  },

  goBack: () => {
    const { steps, currentStep } = get();
    for (let i = currentStep - 1; i >= 0; i--) {
      if (!steps[i].hidden) {
        set({ currentStep: i, canAdvance: true });
        return;
      }
    }
  },

  goToStep: (index: number) => {
    const { steps, maxReached } = get();
    if (index < 0 || index >= steps.length) return;
    if (steps[index].hidden) return;
    if (index > maxReached) return;
    set({ currentStep: index, canAdvance: index < maxReached });
  },

  jumpToStep: (index: number) => {
    const { steps } = get();
    if (index < 0 || index >= steps.length) return;
    if (steps[index].hidden) return;
    set({ currentStep: index, canAdvance: true, maxReached: index });
  },

  setCanAdvance: (can) => set({ canAdvance: can }),

  setPendingUrl: (url) => set({ pendingUrl: url }),

  setPendingModeConfig: (config) => set({ pendingModeConfig: config }),

  setPendingChannelId: (id) => set({ pendingChannelId: id }),

  setSourceRunId: (id) => set({ sourceRunId: id }),

  setStepHidden: (stepId, hidden) => {
    set((state) => ({
      steps: state.steps.map((s) => (s.id === stepId ? { ...s, hidden } : s)),
    }));
  },

  resetWizard: () =>
    set({
      currentStep: 0,
      steps: DEFAULT_STEPS.map((s) => ({ ...s })),
      canAdvance: false,
      maxReached: 0,
      pendingUrl: '',
      pendingModeConfig: null,
      pendingChannelId: null,
      sourceRunId: null,
    }),
}));
