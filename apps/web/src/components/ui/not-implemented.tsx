interface NotImplementedProps {
  feature: string;
}

export function NotImplemented({ feature }: NotImplementedProps) {
  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      <h1 className="font-display text-[32px] font-bold tracking-[-0.02em] text-heading">
        {feature}
      </h1>
      <div className="rounded-xl bg-card border border-border p-8 flex flex-col items-center justify-center gap-3 min-h-[200px]">
        <p className="text-sm text-muted-foreground">Not yet implemented</p>
        <p className="text-xs text-subtle text-center max-w-sm">
          This feature is tracked in the Feature Parity Tracker and will be built
          feature-by-feature from the Kotlin source.
        </p>
      </div>
    </div>
  );
}
