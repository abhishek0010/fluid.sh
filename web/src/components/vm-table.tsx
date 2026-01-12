import * as React from "react";
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
import { useListVirtualMachines } from "~/virsh-sandbox/vms/vms";
import { useCreateSandbox } from "~/virsh-sandbox/sandbox/sandbox";
import { type InternalRestVmInfo } from "~/virsh-sandbox/model";

export function VMTable() {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [cloningId, setCloningId] = React.useState<string | null>(null);

  // Fetch VMs using React Query
  const { data: response, isLoading, isError } = useListVirtualMachines();
  const vms = response?.data?.vms ?? [];

  // Clone VM mutation
  const cloneMutation = useCreateSandbox();

  // Define columns
  const columns: ColumnDef<InternalRestVmInfo>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="font-medium">{row.getValue("name")}</div>
      ),
    },
    {
      accessorKey: "ipAddress",
      header: "IP Address",
      cell: ({ row }) => (
        <div className="font-mono text-muted-foreground">
          {row.getValue("ipAddress")}
        </div>
      ),
    },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => {
        const isCloning = cloningId === row.original.uuid;
        return (
          <Button
            size="sm"
            onClick={() => {
              setCloningId(row?.original?.uuid ?? null);
              cloneMutation.mutate({
                data: { source_vm_name: row?.original?.name },
              });
            }}
            disabled={isCloning}
          >
            {isCloning ? "Cloning..." : "Clone"}
          </Button>
        );
      },
    },
  ];

  // Create table instance
  const table = useReactTable({
    data: vms as InternalRestVmInfo[],
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
        <p className="text-muted-foreground">Loading VMs...</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-destructive">Failed to load VMs</p>
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
                        header.getContext(),
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
                No VMs found.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
