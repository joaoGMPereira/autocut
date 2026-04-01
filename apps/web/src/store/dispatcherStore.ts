import { create } from 'zustand';

export interface DispatcherTask {
  key: string;
  status: 'pending' | 'running' | 'done' | 'error';
  progress: number;
  data: Record<string, unknown>;
  logs: string[];
  error?: string;
}

interface DispatcherState {
  tasks: Record<string, DispatcherTask>;
  upsertTask: (key: string, update: Partial<Omit<DispatcherTask, 'key'>>) => void;
  appendLog: (key: string, message: string) => void;
  setProgress: (key: string, progress: number) => void;
  markDone: (key: string, data: Record<string, unknown>) => void;
  markError: (key: string, error: string) => void;
  getTask: (key: string) => DispatcherTask | undefined;
  removeTask: (key: string) => void;
  reset: () => void;
}

export const useDispatcherStore = create<DispatcherState>((set, get) => ({
  tasks: {},

  upsertTask: (key, update) =>
    set((state) => {
      const existing: DispatcherTask = state.tasks[key] ?? {
        key,
        status: 'pending',
        progress: 0,
        data: {},
        logs: [],
      };
      return { tasks: { ...state.tasks, [key]: { ...existing, ...update, key } } };
    }),

  appendLog: (key, message) =>
    set((state) => {
      const task = state.tasks[key];
      if (!task) return state;
      return { tasks: { ...state.tasks, [key]: { ...task, logs: [...task.logs, message] } } };
    }),

  setProgress: (key, progress) =>
    set((state) => {
      const task = state.tasks[key];
      if (!task) return state;
      return { tasks: { ...state.tasks, [key]: { ...task, progress } } };
    }),

  markDone: (key, data) =>
    set((state) => {
      const task = state.tasks[key] ?? {
        key,
        status: 'pending' as const,
        progress: 0,
        data: {},
        logs: [],
      };
      return {
        tasks: { ...state.tasks, [key]: { ...task, status: 'done' as const, progress: 100, data } },
      };
    }),

  markError: (key, error) =>
    set((state) => {
      const task = state.tasks[key] ?? {
        key,
        status: 'pending' as const,
        progress: 0,
        data: {},
        logs: [],
      };
      return { tasks: { ...state.tasks, [key]: { ...task, status: 'error' as const, error } } };
    }),

  getTask: (key) => get().tasks[key],

  removeTask: (key) =>
    set((state) => {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { [key]: _removed, ...rest } = state.tasks;
      return { tasks: rest };
    }),

  reset: () => set({ tasks: {} }),
}));
