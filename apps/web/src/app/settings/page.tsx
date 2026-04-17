'use client';

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ToolsSection } from '@/components/settings/ToolsSection';
import { AppSettingsSection } from '@/components/settings/AppSettingsSection';
import { ModelsSection } from '@/components/settings/ModelsSection';
import { ChannelsSection } from '@/components/settings/ChannelsSection';
import { ChannelAuthSection } from '@/components/settings/ChannelAuthSection';
import { OAuthProfilesSection } from '@/components/settings/OAuthProfilesSection';
import { createLogger } from '@/lib/logger';

const log = createLogger('[SettingsPage]');

export default function SettingsPage() {
  log.info('settings page rendered');

  return (
    <div className="flex flex-col gap-6 p-6 max-w-3xl mx-auto w-full" data-testid="settings-page">
      <div>
        <h1 className="text-2xl font-semibold text-foreground">Settings</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Configure tools, preferences, models, and connected channels.
        </p>
      </div>

      <Tabs defaultValue="tools" className="w-full">
        <TabsList className="mb-6">
          <TabsTrigger value="tools">Tools</TabsTrigger>
          <TabsTrigger value="app">App</TabsTrigger>
          <TabsTrigger value="models">Models</TabsTrigger>
          <TabsTrigger value="channels">Channels</TabsTrigger>
          <TabsTrigger value="oauth">OAuth</TabsTrigger>
        </TabsList>

        <TabsContent value="tools">
          <ToolsSection />
        </TabsContent>

        <TabsContent value="app">
          <AppSettingsSection />
        </TabsContent>

        <TabsContent value="models">
          <ModelsSection />
        </TabsContent>

        <TabsContent value="channels" className="space-y-6">
          <ChannelsSection />
          <ChannelAuthSection />
        </TabsContent>

        <TabsContent value="oauth">
          <OAuthProfilesSection />
        </TabsContent>
      </Tabs>
    </div>
  );
}
