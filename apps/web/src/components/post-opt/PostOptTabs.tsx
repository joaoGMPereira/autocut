'use client';

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useEffectsStore } from '@/store/effectsStore';
import { AntiDupTab } from './AntiDupTab';
import { SpeedTab } from './SpeedTab';
import { TransitionsTab } from './TransitionsTab';
import { TextOverlayTab } from './TextOverlayTab';
import { LogoTab } from './LogoTab';
import { SseLogPanel } from './SseLogPanel';

const TABS = [
  { value: 'anti-dup', label: 'Anti-Duplicate' },
  { value: 'speed', label: 'Speed Zones' },
  { value: 'transitions', label: 'Transitions' },
  { value: 'text', label: 'Text Overlay' },
  { value: 'logo', label: 'Logo/Watermark' },
] as const;

export function PostOptTabs() {
  const { antiDupJob, speedJob, transitionJob, textJob, overlayJob } = useEffectsStore();

  return (
    <Tabs defaultValue="anti-dup" className="w-full">
      <TabsList className="mb-4 flex-wrap h-auto gap-1">
        {TABS.map((tab) => (
          <TabsTrigger key={tab.value} value={tab.value}>
            {tab.label}
          </TabsTrigger>
        ))}
      </TabsList>

      <TabsContent value="anti-dup">
        <AntiDupTab />
        <SseLogPanel logs={antiDupJob.logs} status={antiDupJob.status} />
      </TabsContent>

      <TabsContent value="speed">
        <SpeedTab />
        <SseLogPanel logs={speedJob.logs} status={speedJob.status} />
      </TabsContent>

      <TabsContent value="transitions">
        <TransitionsTab />
        <SseLogPanel logs={transitionJob.logs} status={transitionJob.status} />
      </TabsContent>

      <TabsContent value="text">
        <TextOverlayTab />
        <SseLogPanel logs={textJob.logs} status={textJob.status} />
      </TabsContent>

      <TabsContent value="logo">
        <LogoTab />
        <SseLogPanel logs={overlayJob.logs} status={overlayJob.status} />
      </TabsContent>
    </Tabs>
  );
}
