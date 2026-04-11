import { Separator } from '@/components/ui/separator';
import { ChannelsSection } from '@/components/settings/ChannelsSection';
import { ToolsSection } from '@/components/settings/ToolsSection';
import { AppSettingsSection } from '@/components/settings/AppSettingsSection';
import { OAuthProfilesSection } from '@/components/settings/OAuthProfilesSection';
import { ChannelAuthSection } from '@/components/settings/ChannelAuthSection';
import { createLogger } from '@/lib/logger';

const log = createLogger('SettingsPage');

export default function SettingsPage() {
  log.info('rendering settings page');

  return (
    <div className="p-6 max-w-3xl space-y-8">
      <h1 className="text-2xl font-semibold text-foreground">Settings</h1>

      <ChannelsSection />

      <Separator />

      <OAuthProfilesSection />

      <Separator />

      <ChannelAuthSection />

      <Separator />

      <ToolsSection />

      <Separator />

      <AppSettingsSection />
    </div>
  );
}
