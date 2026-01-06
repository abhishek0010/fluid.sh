import { createFileRoute } from "@tanstack/react-router";
import { SandboxDetails } from "~/components/sandbox-details";

export const Route = createFileRoute("/sandboxes/$id")({
  component: SandboxDetailsPage,
});

function SandboxDetailsPage() {
  const { id } = Route.useParams();
  return <SandboxDetails sandboxId={id} />;
}
