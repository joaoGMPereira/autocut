'use client';

import { useSetupStore } from '@/store/setupStore';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export function HardwareCard() {
  const hardware = useSetupStore((s) => s.hardware);
  const recommendation = useSetupStore((s) => s.recommendation);
  const hardwareLoading = useSetupStore((s) => s.hardwareLoading);

  if (hardwareLoading) {
    return (
      <Card>
        <CardContent className="py-4 text-sm text-muted-foreground">
          Detecting hardware...
        </CardContent>
      </Card>
    );
  }

  if (!hardware || !recommendation) return null;

  const ramGB = Math.round(hardware.total_ram_mb / 1024);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">Hardware Profile</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {/* Hardware summary */}
        <div className="grid grid-cols-2 gap-1.5 text-muted-foreground">
          <span>RAM</span>
          <span className="text-foreground font-medium">{ramGB} GB</span>
          <span>CPU</span>
          <span className="text-foreground font-medium">{hardware.cpu_cores} cores ({hardware.arch})</span>
          <span>GPU</span>
          <span className="text-foreground font-medium">{hardware.gpu_vendor}</span>
        </div>

        <div className="border-t border-border pt-3 space-y-2">
          {/* Whisper recommendation */}
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Whisper model</span>
            <div className="flex items-center gap-2">
              <span className="font-medium">{recommendation.whisper_tier}</span>
              <span className="text-xs text-muted-foreground">({recommendation.whisper_size_mb} MB)</span>
              {recommendation.upgrade_available && (
                <Badge variant="outline" className="text-xs border-amber-500 text-amber-500">
                  Upgrade available
                </Badge>
              )}
            </div>
          </div>
          <p className="text-xs text-muted-foreground">{recommendation.whisper_reason}</p>

          {/* Ollama recommendation */}
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Ollama model</span>
            <span className="font-medium">{recommendation.ollama_model}</span>
          </div>
          <p className="text-xs text-muted-foreground">{recommendation.ollama_reason}</p>
        </div>
      </CardContent>
    </Card>
  );
}
