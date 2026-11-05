"use client";

import type { MouseEvent } from "react";

import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { cn } from "@/lib/utils";

/** 1 … 4 5 6 … 12: the current page with one neighbour on each side, plus the ends. */
export function pageItems(page: number, pageCount: number): (number | "gap")[] {
  if (pageCount <= 7) return Array.from({ length: pageCount }, (_, i) => i + 1);
  const pages = new Set([1, pageCount, page - 1, page, page + 1].filter((p) => p >= 1 && p <= pageCount));
  const sorted = [...pages].sort((a, b) => a - b);
  const items: (number | "gap")[] = [];
  for (const [i, p] of sorted.entries()) {
    if (i > 0 && p - sorted[i - 1] > 1) items.push("gap");
    items.push(p);
  }
  return items;
}

function ListPagination({
  page,
  pageCount,
  onPage,
}: {
  page: number;
  pageCount: number;
  onPage: (page: number) => void;
}) {
  const go = (target: number) => (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault();
    if (target >= 1 && target <= pageCount && target !== page) onPage(target);
  };
  const disabled = "pointer-events-none opacity-50";
  return (
    <Pagination className="mx-0 w-auto">
      <PaginationContent>
        <PaginationItem>
          <PaginationPrevious
            href={`?page=${page - 1}`}
            onClick={go(page - 1)}
            aria-disabled={page <= 1}
            className={cn(page <= 1 && disabled)}
          />
        </PaginationItem>
        {pageItems(page, pageCount).map((item, i) =>
          item === "gap" ? (
            <PaginationItem key={`gap-${i}`}>
              <PaginationEllipsis />
            </PaginationItem>
          ) : (
            <PaginationItem key={item}>
              <PaginationLink href={`?page=${item}`} onClick={go(item)} isActive={item === page}>
                {item}
              </PaginationLink>
            </PaginationItem>
          ),
        )}
        <PaginationItem>
          <PaginationNext
            href={`?page=${page + 1}`}
            onClick={go(page + 1)}
            aria-disabled={page >= pageCount}
            className={cn(page >= pageCount && disabled)}
          />
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  );
}

/**
 * "Showing 1–20 of 64" plus pagination when there is more than one page.
 * Renders nothing for an empty list; the empty state speaks instead.
 */
export function ListFooter({
  offset,
  shown,
  total,
  pageSize,
  page,
  onPage,
}: {
  offset: number;
  shown: number;
  total: number;
  pageSize: number;
  page: number;
  onPage: (page: number) => void;
}) {
  if (total === 0) return null;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-muted-foreground">
      <p className="tabular-nums">
        Showing {offset + 1}–{Math.min(offset + shown, total)} of {total}
      </p>
      {pageCount > 1 ? <ListPagination page={page} pageCount={pageCount} onPage={onPage} /> : null}
    </div>
  );
}
