import { createFileRoute } from "@tanstack/react-router";
import { SandboxTable } from "~/components/sandbox-table";

export const Route = createFileRoute("/sandboxes/")({
  component: SandboxesPage,
});

function SandboxesPage() {
  return (
    <main className="container mx-auto py-8 px-4">
      <div className="mb-8">
        <h1 className="text-3xl font-bold">Sandboxes</h1>
        <p className="text-muted-foreground">
          View and manage your virtual machine sandboxes
        </p>
      </div>
      <SandboxTable />
    </main>
  );
}
