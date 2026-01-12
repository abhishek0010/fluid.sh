import { createFileRoute } from "@tanstack/react-router";
import { VMTable } from "~/components/vm-table";

export const Route = createFileRoute("/")({
  component: HomePage,
});

function HomePage() {
  return (
    <main className="container mx-auto py-8 px-4">
      <h1 className="text-3xl font-bold mb-6">Virtual Machines</h1>
      <VMTable />
    </main>
  );
}
