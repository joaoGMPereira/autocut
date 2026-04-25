// apps/web/src/store/logStore.ts
import { create } from 'zustand';
import type { LogEntry } from '@autocut/shared';

const MAX_ENTRIES = 1000;

interface LogState {
  entries: LogEntry[];
  search: string;
  open: boolean;
  addEntry: (entry: LogEntry) => void;
  clear: () => void;
  setSearch: (search: string) => void;
  setOpen: (open: boolean) => void;
  toggleOpen: () => void;
}

export const useLogStore = create<LogState>((set) => ({
  entries: [],
  search: '',
  open: false,

  addEntry: (entry) =>
    set((state) => {
      const next = [...state.entries, entry];
      return {
        entries: next.length > MAX_ENTRIES ? next.slice(next.length - MAX_ENTRIES) : next,
      };
    }),

  clear: () => set({ entries: [] }),
  setSearch: (search) => set({ search }),
  setOpen: (open) => set({ open }),
  toggleOpen: () => set((state) => ({ open: !state.open })),
}));
