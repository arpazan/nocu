"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ChevronsUpDownIcon, LogOutIcon } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { signOut } from "@/lib/auth/refresh";
import { useSession } from "@/lib/auth/session-store";
import { useMe } from "@/lib/auth/use-me";

function initialOf(name: string, email: string): string {
  const source = name.trim().length > 0 ? name.trim() : email;
  return source.charAt(0).toUpperCase();
}

/** Compact sidebar footer: who is signed in, their roles and a sign-out action. */
export function UserMenu() {
  const session = useSession();
  const me = useMe();
  const router = useRouter();
  const queryClient = useQueryClient();

  async function handleSignOut() {
    await signOut();
    queryClient.clear();
    router.push("/login");
  }

  if (!session) {
    return (
      <Button asChild variant="outline" size="sm" className="w-full">
        <Link href="/login">Sign in</Link>
      </Button>
    );
  }

  if (!me.data) {
    return (
      <div className="flex items-center gap-2 px-1" aria-busy="true" aria-label="Loading account">
        <Skeleton className="size-8 shrink-0 rounded-full" />
        <div className="flex-1 space-y-1.5">
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-3 w-32" />
        </div>
      </div>
    );
  }

  const { name, email, roles } = me.data;
  const displayName = name.trim().length > 0 ? name : email;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" className="h-auto w-full min-w-0 justify-start gap-2 px-1 py-1">
          <span
            aria-hidden="true"
            className="flex size-8 shrink-0 items-center justify-center rounded-full bg-secondary text-sm font-medium text-secondary-foreground"
          >
            {initialOf(name, email)}
          </span>
          <span className="min-w-0 flex-1 text-left">
            <span className="block truncate text-sm font-medium">{displayName}</span>
            <span className="block truncate text-xs text-muted-foreground">{email}</span>
            <span className="sr-only">, account menu</span>
          </span>
          <ChevronsUpDownIcon className="size-4 shrink-0 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" side="top" className="w-56">
        <DropdownMenuLabel className="space-y-1.5">
          <span className="block truncate text-foreground">{email}</span>
          <span className="flex flex-wrap gap-1">
            {roles.map((role) => (
              <Badge key={role} variant="secondary">
                {role}
              </Badge>
            ))}
          </span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => void handleSignOut()}>
          <LogOutIcon />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
