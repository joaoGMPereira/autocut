'use client';

import { Button } from '@/components/ui/button';
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
  variant: 'cancel-operation' | 'leave-page' | 'start-over';
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
  'start-over': {
    title: 'Start over?',
    body: 'This will cancel the current pipeline run and return you to the beginning.',
    confirmLabel: 'Start Over',
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
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Keep going
          </Button>
          <Button variant="destructive" onClick={handleConfirm}>
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
