"use client";

import { createColumnHelper, tableFeatures, useTable, type ColumnDef, type RowData } from "@tanstack/react-table";
import type { ReactNode } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

/**
 * Per-column presentation hints the list tables share. `className` lands on
 * both the header and every cell, so responsive hiding stays in step;
 * `skeleton` is what a loading row shows in that column.
 */
export interface ListColumnMeta {
  className?: string;
  skeleton?: ReactNode;
}

/**
 * Every list in the app is paged and sorted by the API, so the table only
 * needs the core model: no client sorting, filtering or pagination features.
 */
export const listFeatures = tableFeatures({ columnMeta: {} as ListColumnMeta });

export type ListFeatures = typeof listFeatures;

// The helper's own `columns()` uses `any` for TValue so accessor columns of
// different value types can share one array; mirror that here.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type ListColumns<TData extends RowData> = ReadonlyArray<ColumnDef<ListFeatures, TData, any>>;

/** A column helper bound to the list feature set, so `meta` is typed as ListColumnMeta. */
export function createListColumnHelper<TData extends RowData>() {
  return createColumnHelper<ListFeatures, TData>();
}

/** A shared, stable "no rows" fallback: a fresh `[]` each render would rebuild the row model. */
export const NO_ROWS: never[] = [];

const DEFAULT_SKELETON = <Skeleton className="h-4 w-16" />;

interface DataTableProps<TData extends { id: string }> {
  columns: ListColumns<TData>;
  data: readonly TData[];
  /** Render skeleton rows instead of data while the first page loads. */
  isPending?: boolean;
  skeletonRows?: number;
  /** When set, an empty `data` renders this message in one row spanning every column. */
  emptyMessage?: ReactNode;
}

/**
 * Renders a TanStack column model through the shadcn table primitives. The
 * list component around it still owns filters, pagination, the empty state
 * and the error panel; this only turns rows and columns into markup.
 */
export function DataTable<TData extends { id: string }>({
  columns,
  data,
  isPending = false,
  skeletonRows = 5,
  emptyMessage,
}: DataTableProps<TData>) {
  const table = useTable({
    features: listFeatures,
    columns: columns as ColumnDef<ListFeatures, TData>[],
    data: data as TData[],
    getRowId: (row) => row.id,
  });
  const leafColumns = table.getAllLeafColumns();
  const rows = table.getRowModel().rows;

  return (
    <Table>
      <TableHeader>
        {table.getHeaderGroups().map((group) => (
          <TableRow key={group.id}>
            {group.headers.map((header) => (
              <TableHead key={header.id} className={header.column.columnDef.meta?.className}>
                {header.isPlaceholder ? null : <table.FlexRender header={header} />}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {isPending ? (
          Array.from({ length: skeletonRows }, (_, i) => (
            <TableRow key={i} aria-hidden="true" className="hover:bg-transparent">
              {leafColumns.map((column) => (
                <TableCell key={column.id} className={column.columnDef.meta?.className}>
                  {column.columnDef.meta?.skeleton === undefined ? DEFAULT_SKELETON : column.columnDef.meta.skeleton}
                </TableCell>
              ))}
            </TableRow>
          ))
        ) : rows.length === 0 && emptyMessage !== undefined ? (
          <TableRow>
            <TableCell colSpan={leafColumns.length} className="py-8 text-center text-muted-foreground">
              {emptyMessage}
            </TableCell>
          </TableRow>
        ) : (
          rows.map((row) => (
            <TableRow key={row.id}>
              {row.getAllCells().map((cell) => (
                <TableCell key={cell.id} className={cell.column.columnDef.meta?.className}>
                  <table.FlexRender cell={cell} />
                </TableCell>
              ))}
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  );
}
