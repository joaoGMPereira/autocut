'use client';

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';

interface BackConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  variant: 'cancel-operation' | 'leave-page';
  onConfirm: () => void;
}

const VARIANTS = {
  'cancel-operation': {
    title: 'Cancel active operation?',
    body: 'Going back will cancel the active operation. The current progress will be lost.',
    confirmLabel: 'Cancel Operation',
  },
  'leave-page': {
    title: 'Leave pipeline?',
    body: 'Leaving this page will cancel your active pipeline run.',
    confirmLabel: 'Leave',
  },
} as const;

export function BackConfirmDialog({ open, onOpenChange, variant, onConfirm }: BackConfirmDialogProps) {
  const { title, body, confirmLabel } = VARIANTS[variant];

  const handleConfirm = () => {
    onOpenChange(false);
    onConfirm();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{body}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <button
            onClick={() => onOpenChange(false)}
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-zinc-200"
          >
            Keep going
          </button>
          <button
            onClick={handleConfirm}
            className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700"
          >
            {confirmLabel}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
