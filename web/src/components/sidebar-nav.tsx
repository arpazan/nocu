"use client";

import { ActivityIcon, LayoutDashboardIcon, MenuIcon, SirenIcon, XIcon } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, type ReactNode } from "react";

import { Brand } from "@/components/brand";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { href: "/", label: "Overview", icon: LayoutDashboardIcon },
  { href: "/monitors", label: "Monitors", icon: ActivityIcon },
  { href: "/incidents", label: "Incidents", icon: SirenIcon },
] as const;

function isActive(pathname: string, href: string) {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

/** Primary navigation; highlights the current section with an orange left marker. */
export function SidebarNav({
  onNavigate,
  className,
}: {
  onNavigate?: () => void;
  className?: string;
}) {
  const pathname = usePathname();

  return (
    <nav aria-label="Primary" className={cn("flex flex-col gap-0.5", className)}>
      {NAV_ITEMS.map(({ href, label, icon: Icon }) => {
        const active = isActive(pathname, href);
        return (
          <Link
            key={href}
            href={href}
            onClick={onNavigate}
            aria-current={active ? "page" : undefined}
            className={cn(
              "relative flex h-11 items-center gap-3 rounded-md px-3 text-base font-medium transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
              "before:absolute before:top-2 before:bottom-2 before:-left-3 before:w-[3px] before:rounded-r-full before:bg-primary before:opacity-0 before:transition-opacity",
              active
                ? "text-foreground before:opacity-100"
                : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
            )}
          >
            <Icon className="size-5 shrink-0" aria-hidden="true" />
            {label}
          </Link>
        );
      })}
    </nav>
  );
}

/**
 * Hamburger trigger plus a sheet holding the brand, the same nav, and a footer
 * slot for the account menu, so narrow screens lose nothing the sidebar has.
 */
export function MobileNav({ footer }: { footer?: ReactNode }) {
  const [open, setOpen] = useState(false);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="size-11" aria-label="Open navigation">
          <MenuIcon className="size-4" aria-hidden="true" />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" showCloseButton={false} className="gap-0 bg-sidebar text-sidebar-foreground">
        {/* Touch surface: 44px targets for the close, matching the hamburger that opened it. */}
        <SheetHeader className="relative h-14 justify-center px-6 py-0">
          <SheetTitle className="sr-only">Navigation</SheetTitle>
          <SheetDescription className="sr-only">Jump to a section of Nocu.</SheetDescription>
          <Brand className="w-fit text-lg" />
          <SheetClose asChild>
            <Button variant="ghost" size="icon" className="absolute top-1.5 right-1.5 size-11" aria-label="Close navigation">
              <XIcon className="size-4" aria-hidden="true" />
            </Button>
          </SheetClose>
        </SheetHeader>
        <SidebarNav className="px-3" onNavigate={() => setOpen(false)} />
        {footer ? <div className="mt-auto border-t px-3 py-3">{footer}</div> : null}
      </SheetContent>
    </Sheet>
  );
}
