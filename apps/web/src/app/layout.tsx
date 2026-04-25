import type { Metadata } from 'next';
import { GeistSans } from 'geist/font/sans';
import { GeistMono } from 'geist/font/mono';
import { Space_Grotesk, JetBrains_Mono } from 'next/font/google';
import { TooltipProvider } from '@/components/ui/tooltip';
import { AppSidebar } from '@/components/app-sidebar';
import { SetupProvider } from '@/components/setup/SetupProvider';
import { UpdateStatusProvider } from '@/components/UpdateStatusProvider';
import { LogsProvider } from '@/components/LogsProvider';
import './globals.css';

const spaceGrotesk = Space_Grotesk({
  weight: ['500', '600', '700'],
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-display',
});

const jetbrainsMono = JetBrains_Mono({
  weight: ['400', '500', '600', '700'],
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-jetbrains-mono',
});

export const metadata: Metadata = {
  title: 'AutoCut',
  description: 'YouTube video processing pipeline',
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable} ${spaceGrotesk.variable} ${jetbrainsMono.variable} dark h-full`}
    >
      <body className="h-full antialiased bg-background text-foreground">
        <TooltipProvider>
          <UpdateStatusProvider>
            <div className="flex h-full">
              <AppSidebar />
              <main className="flex-1 overflow-auto">
                <SetupProvider>{children}</SetupProvider>
              </main>
            </div>
            <LogsProvider />
          </UpdateStatusProvider>
        </TooltipProvider>
      </body>
    </html>
  );
}
