'use client';

import Link from 'next/link';
import Image from 'next/image';
import { usePathname } from 'next/navigation';
import {
  Home,
  Play,
  Upload,
  Users,
  Settings,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Separator } from '@/components/ui/separator';

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}

const navItems: NavItem[] = [
  { label: 'Dashboard', href: '/',         icon: Home },
  { label: 'Pipeline',  href: '/pipeline', icon: Play },
  { label: 'Queue',     href: '/queue',    icon: Upload },
  { label: 'Channels',  href: '/channels', icon: Users },
  { label: 'Settings',  href: '/settings', icon: Settings },
];

export function AppSidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex flex-col w-14 bg-sidebar border-r border-sidebar-border shrink-0 pb-3 gap-1">
      {/* macOS hiddenInset safe area — 28px drag region above content */}
      <div className="drag-region h-7 w-full shrink-0" />

      {/* Logo */}
      <div className="flex items-center justify-center h-9 mb-1">
        <Image src="/logo-mark.svg" alt="AutoCut" width={28} height={28} priority />
      </div>

      <Separator className="bg-sidebar-border mx-2 w-auto" />

      {/* Nav items */}
      <nav className="flex flex-col gap-1 px-2 pt-1 flex-1">
        {navItems.map(({ label, href, icon: Icon }) => {
          const isActive =
            href === '/' ? pathname === '/' : pathname.startsWith(href);

          return (
            <Tooltip key={href} delayDuration={200}>
              <TooltipTrigger asChild>
                <Link
                  href={href}
                  className={cn(
                    'flex items-center justify-center h-9 w-9 rounded-md transition-colors',
                    'hover:bg-surface hover:text-brand',
                    isActive
                      ? 'bg-surface text-brand'
                      : 'text-subtle',
                  )}
                >
                  <Icon className="h-4 w-4" />
                  <span className="sr-only">{label}</span>
                </Link>
              </TooltipTrigger>
              <TooltipContent side="right" sideOffset={8}>
                {label}
              </TooltipContent>
            </Tooltip>
          );
        })}
      </nav>
    </aside>
  );
}
