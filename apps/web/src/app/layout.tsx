import type { Metadata } from 'next';
import { GeistSans } from 'geist/font/sans';
import { GeistMono } from 'geist/font/mono';
import { TooltipProvider } from '@/components/ui/tooltip';
import { AppSidebar } from '@/components/app-sidebar';
import { SetupProvider } from '@/components/setup/SetupProvider';
import './globals.css';

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
      className={`${GeistSans.variable} ${GeistMono.variable} dark h-full`}
    >
      <body className="h-full antialiased bg-background text-foreground">
        <TooltipProvider>
          <div className="flex h-full">
            <AppSidebar />
            <main className="flex-1 overflow-auto">
              <SetupProvider>{children}</SetupProvider>
            </main>
          </div>
        </TooltipProvider>
      </body>
    </html>
  );
}
