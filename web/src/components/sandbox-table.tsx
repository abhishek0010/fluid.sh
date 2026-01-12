import * as React from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "~/components/ui/table";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import { useListSandboxes } from "~/virsh-sandbox/sandbox/sandbox";
import type { InternalRestSandboxInfo } from "~/virsh-sandbox/model";

function getStateBadgeVariant(
  state: string | undefined
): "default" | "secondary" | "destructive" | "outline" {
  switch (state) {
    case "RUNNING":
      return "default";
    case "CREATED":
    case "STARTING":
      return "secondary";
    case "ERROR":
    case "DESTROYED":
      return "destructive";
    default:
      return "outline";
  }
}

export function SandboxTable() {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const navigate = useNavigate();

  const { data: response, isLoading, isError } = useListSandboxes();
  const sandboxes = response?.data?.sandboxes ?? [];

  const columns: ColumnDef<InternalRestSandboxInfo>[] = [
    {
      accessorKey: "id",
      header: "ID",
      cell: ({ row }) => (
        <div className="font-mono text-sm">{row.getValue("id")}</div>
      ),
    },
    {
      accessorKey: "sandbox_name",
      header: "Name",
      cell: ({ row }) => (
        <div className="font-medium">{row.getValue("sandbox_name")}</div>
      ),
    },
    {
      accessorKey: "state",
      header: "State",
      cell: ({ row }) => {
        const state = row.getValue("state") as string;
        return <Badge variant={getStateBadgeVariant(state)}>{state}</Badge>;
      },
    },
    {
      accessorKey: "ip_address",
      header: "IP Address",
      cell: ({ row }) => (
        <div className="font-mono text-sm text-muted-foreground">
          {row.getValue("ip_address") || "-"}
        </div>
      ),
    },
    {
      accessorKey: "base_image",
      header: "Base Image",
      cell: ({ row }) => (
        <div className="text-sm text-muted-foreground">
          {row.getValue("base_image")}
        </div>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }) => {
        const date = row.getValue("created_at") as string;
        return (
          <div className="text-sm text-muted-foreground">
            {date ? new Date(date).toLocaleString() : "-"}
          </div>
        );
      },
    },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => {
        return (
          <Button
            size="sm"
            onClick={() => {
              navigate({ to: `/sandboxes/${row.original.id}` });
            }}
          >
            View Details
          </Button>
        );
      },
    },
  ];

  const table = useReactTable({
    data: sandboxes as InternalRestSandboxInfo[],
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: {
      sorting,
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-muted-foreground">Loading sandboxes...</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-destructive">Failed to load sandboxes</p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-card">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow
                key={row.id}
                data-state={row.getIsSelected() && "selected"}
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-24 text-center">
                No sandboxes found.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
