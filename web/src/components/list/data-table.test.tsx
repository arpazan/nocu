import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createListColumnHelper, DataTable } from "@/components/list/data-table";

interface Fruit {
  id: string;
  name: string;
  colour: string;
}

const helper = createListColumnHelper<Fruit>();

const columns = helper.columns([
  helper.accessor("name", { header: "Name" }),
  helper.accessor("colour", {
    header: "Colour",
    meta: { className: "hidden lg:table-cell", skeleton: <span data-testid="colour-skeleton" /> },
  }),
  helper.display({
    id: "actions",
    header: () => <span className="sr-only">Actions</span>,
    cell: ({ row }) => <button type="button">Eat {row.original.name}</button>,
  }),
]);

const fruit: Fruit[] = [
  { id: "f1", name: "Apple", colour: "red" },
  { id: "f2", name: "Banana", colour: "yellow" },
];

function tbody() {
  return screen.getByRole("table").querySelector("tbody")!;
}

describe("DataTable", () => {
  it("renders a header per column and a row per item", () => {
    render(<DataTable columns={columns} data={fruit} />);

    expect(screen.getAllByRole("columnheader").map((th) => th.textContent)).toEqual(["Name", "Colour", "Actions"]);
    const rows = within(tbody()).getAllByRole("row");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByText("Apple")).toBeInTheDocument();
    expect(within(rows[0]).getByRole("button", { name: "Eat Apple" })).toBeInTheDocument();
    expect(within(rows[1]).getByText("yellow")).toBeInTheDocument();
  });

  it("applies a column's className to its header and every cell", () => {
    render(<DataTable columns={columns} data={fruit} />);

    expect(screen.getByRole("columnheader", { name: "Colour" })).toHaveClass("hidden", "lg:table-cell");
    for (const cell of [screen.getByText("red"), screen.getByText("yellow")]) {
      expect(cell.closest("td")).toHaveClass("hidden", "lg:table-cell");
    }
    expect(screen.getByText("Apple").closest("td")).not.toHaveClass("hidden");
  });

  it("shows hidden skeleton rows while pending, using each column's skeleton", () => {
    render(<DataTable columns={columns} data={[]} isPending skeletonRows={3} />);

    const rows = tbody().querySelectorAll("tr");
    expect(rows).toHaveLength(3);
    for (const row of rows) expect(row).toHaveAttribute("aria-hidden", "true");
    expect(screen.getAllByTestId("colour-skeleton")).toHaveLength(3);
    expect(tbody().querySelectorAll("td")).toHaveLength(9);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("renders the empty message across every column when there are no rows", () => {
    render(<DataTable columns={columns} data={[]} emptyMessage="Nothing here." />);

    expect(screen.getByText("Nothing here.")).toHaveAttribute("colspan", "3");
  });

  it("renders an empty body for no rows without an empty message", () => {
    render(<DataTable columns={columns} data={[]} />);

    expect(tbody().children).toHaveLength(0);
  });
});
