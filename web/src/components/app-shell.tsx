import type { ReactNode } from "react";

import { Brand } from "@/components/brand";
import { MobileNav, SidebarNav } from "@/components/sidebar-nav";
import { ThemeToggle } from "@/components/theme-toggle";
import { UserMenu } from "@/components/user-menu";

/**
 * Page chrome shared by every route.
 * "app" renders the sidebar layout; "bare" centers its children (login, register).
 */
export function AppShell({
  children,
  variant = "app",
}: {
  children: ReactNode;
  variant?: "app" | "bare";
}) {
  if (variant === "bare") {
    return (
      <div className="flex min-h-full flex-1 flex-col items-center justify-center gap-6 p-6">
        <Brand />
        {children}
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-1 flex-col md:flex-row">
      {/* Narrow screens: top bar with the nav and account in a sheet. */}
      <header className="flex h-14 items-center justify-between border-b bg-sidebar px-4 md:hidden">
        <div className="flex items-center gap-1">
          <MobileNav footer={<UserMenu />} />
          <Brand />
        </div>
        <ThemeToggle className="size-11" />
      </header>

      {/* Wide screens: fixed-width sidebar. */}
      <aside aria-label="Sidebar" className="sticky top-0 hidden h-dvh w-64 shrink-0 flex-col border-r bg-sidebar text-sidebar-foreground md:flex">
        <div className="flex h-14 items-center px-6">
          <Brand className="text-lg" />
        </div>
        <SidebarNav className="px-3" />
        <div className="mt-auto flex items-center gap-1 border-t px-3 py-3">
          <div className="min-w-0 flex-1">
            <UserMenu />
          </div>
          <ThemeToggle />
        </div>
      </aside>

      <main className="min-w-0 flex-1 px-6 py-8 lg:px-10">
        <div className="mx-auto w-full max-w-6xl">{children}</div>
      </main>
    </div>
  );
}
