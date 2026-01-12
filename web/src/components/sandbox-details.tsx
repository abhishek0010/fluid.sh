import * as React from "react";
import { useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Play, Terminal, FileText, Clock } from "lucide-react";
import { Button } from "~/components/ui/button";
import { Badge } from "~/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "~/components/ui/card";
import { useGetSandbox } from "~/virsh-sandbox/sandbox/sandbox";
import { useCreateAnsibleJob } from "~/virsh-sandbox/ansible/ansible";
import { useSandboxStream } from "~/hooks/use-sandbox-stream";

function getStateBadgeVariant(
  state: string | undefined,
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

interface SandboxDetailsProps {
  sandboxId: string;
}

export function SandboxDetails({ sandboxId }: SandboxDetailsProps) {
  const navigate = useNavigate();
  const [showAnsibleDialog, setShowAnsibleDialog] = React.useState(false);
  const [playbookPath, setPlaybookPath] = React.useState("");

  const {
    data: response,
    isLoading,
    isError,
    error,
  } = useGetSandbox(sandboxId);

  const sandboxData = response?.data;

  const {
    isConnected,
    commands: streamCommands,
    connectionData,
  } = useSandboxStream(sandboxId);

  const createAnsibleJobMutation = useCreateAnsibleJob();

  // Combine commands from initial fetch and stream
  const allCommands = React.useMemo(() => {
    const initialCommands = sandboxData?.commands || [];
    const commandMap = new Map<string, (typeof initialCommands)[0]>();

    // Add initial commands
    for (const cmd of initialCommands) {
      if (cmd.id) {
        commandMap.set(cmd.id, cmd);
      }
    }

    // Add/update with stream commands
    for (const cmd of streamCommands) {
      commandMap.set(cmd.command_id, {
        id: cmd.command_id,
        command: cmd.command,
        stdout: cmd.stdout,
        stderr: cmd.stderr,
        exit_code: cmd.exit_code,
        started_at: cmd.started_at,
        ended_at: cmd.ended_at,
      });
    }

    return Array.from(commandMap.values()).sort((a, b) => {
      const dateA = a.started_at ? new Date(a.started_at).getTime() : 0;
      const dateB = b.started_at ? new Date(b.started_at).getTime() : 0;
      return dateA - dateB;
    });
  }, [sandboxData?.commands, streamCommands]);

  const handleRunAnsible = () => {
    if (!sandboxData?.sandbox?.sandbox_name || !playbookPath) return;

    createAnsibleJobMutation.mutate(
      {
        data: {
          vm_name: sandboxData.sandbox.sandbox_name,
          playbook: playbookPath,
          check: false,
        },
      },
      {
        onSuccess: (data) => {
          setShowAnsibleDialog(false);
          console.log("Ansible job created:", data);
        },
      },
    );
  };

  if (isLoading) {
    return (
      <main className="container mx-auto py-8 px-4">
        <div className="flex items-center justify-center p-8">
          <p className="text-muted-foreground">Loading sandbox details...</p>
        </div>
      </main>
    );
  }

  if (isError || !sandboxData) {
    return (
      <main className="container mx-auto py-8 px-4">
        <Button
          variant="ghost"
          className="mb-6"
          onClick={() => navigate({ to: "/sandboxes" })}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Sandboxes
        </Button>
        <div className="flex flex-col items-center justify-center p-8 gap-4">
          <p className="text-destructive">Failed to load sandbox details</p>
          {error && (
            <p className="text-sm text-muted-foreground">{String(error)}</p>
          )}
        </div>
      </main>
    );
  }

  const sandbox = sandboxData.sandbox;

  return (
    <main className="container mx-auto py-8 px-4">
      <Button
        variant="ghost"
        className="mb-6"
        onClick={() => navigate({ to: "/sandboxes" })}
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Sandboxes
      </Button>

      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-3xl font-bold">Sandbox Details</h1>
          <Badge variant={getStateBadgeVariant(sandbox?.state)}>
            {sandbox?.state}
          </Badge>
          {isConnected && (
            <Badge
              variant="outline"
              className="text-green-600 border-green-600"
            >
              <span className="mr-1 h-2 w-2 rounded-full bg-green-600 inline-block animate-pulse" />
              Live
            </Badge>
          )}
        </div>
        <p className="text-muted-foreground font-mono text-sm">{sandbox?.id}</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Sandbox Information */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              Sandbox Information
            </CardTitle>
            <CardDescription>Basic details about this sandbox</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Sandbox Name
              </p>
              <p className="font-mono text-sm">{sandbox?.sandbox_name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                IP Address
              </p>
              <p className="font-mono text-sm">
                {sandbox?.ip_address || connectionData?.ip_address || "-"}
              </p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Base Image
              </p>
              <p className="text-sm">{sandbox?.base_image}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Network
              </p>
              <p className="text-sm">{sandbox?.network}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Agent ID
              </p>
              <p className="font-mono text-sm">{sandbox?.agent_id}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Job ID
              </p>
              <p className="font-mono text-sm">{sandbox?.job_id}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Created
              </p>
              <p className="text-sm">
                {sandbox?.created_at
                  ? new Date(sandbox.created_at).toLocaleString()
                  : "-"}
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Command Stream */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Terminal className="h-5 w-5" />
              Command Stream
              {allCommands.length > 0 && (
                <Badge variant="secondary">{allCommands.length} commands</Badge>
              )}
            </CardTitle>
            <CardDescription>
              Realtime view of commands executed in this sandbox
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4 max-h-[600px] overflow-y-auto">
              {allCommands.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  No commands executed yet
                </div>
              ) : (
                allCommands.map((cmd, index) => (
                  <div
                    key={cmd.id || index}
                    className="rounded-lg border bg-muted/50 p-4 space-y-2"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4 text-muted-foreground" />
                        <span className="text-xs text-muted-foreground">
                          {cmd.started_at
                            ? new Date(cmd.started_at).toLocaleTimeString()
                            : ""}
                        </span>
                      </div>
                      {cmd.exit_code !== undefined && (
                        <Badge
                          variant={
                            cmd.exit_code === 0 ? "default" : "destructive"
                          }
                        >
                          Exit: {cmd.exit_code}
                        </Badge>
                      )}
                    </div>
                    <div className="rounded-md bg-background p-3">
                      <code className="text-sm font-mono text-foreground">
                        $ {cmd.command}
                      </code>
                    </div>
                    {cmd.stdout && (
                      <div>
                        <p className="text-xs font-medium text-muted-foreground mb-1">
                          stdout
                        </p>
                        <pre className="rounded-md bg-background p-3 text-xs font-mono whitespace-pre-wrap text-muted-foreground overflow-x-auto">
                          {cmd.stdout}
                        </pre>
                      </div>
                    )}
                    {cmd.stderr && (
                      <div>
                        <p className="text-xs font-medium text-destructive mb-1">
                          stderr
                        </p>
                        <pre className="rounded-md bg-destructive/10 p-3 text-xs font-mono whitespace-pre-wrap text-destructive overflow-x-auto">
                          {cmd.stderr}
                        </pre>
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Run Ansible Section */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Play className="h-5 w-5" />
            Deploy to Production
          </CardTitle>
          <CardDescription>
            Run Ansible playbook on the production machine based on sandbox
            changes
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!showAnsibleDialog ? (
            <Button
              onClick={() => setShowAnsibleDialog(true)}
              disabled={sandbox?.state !== "RUNNING"}
            >
              <Play className="mr-2 h-4 w-4" />
              Run Ansible Playbook
            </Button>
          ) : (
            <div className="space-y-4">
              <div>
                <label
                  htmlFor="playbook-path"
                  className="text-sm font-medium text-muted-foreground"
                >
                  Playbook Path
                </label>
                <input
                  id="playbook-path"
                  type="text"
                  value={playbookPath}
                  onChange={(e) => setPlaybookPath(e.target.value)}
                  placeholder="/path/to/playbook.yml"
                  className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  onClick={handleRunAnsible}
                  disabled={!playbookPath || createAnsibleJobMutation.isPending}
                >
                  {createAnsibleJobMutation.isPending
                    ? "Creating Job..."
                    : "Run Playbook"}
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setShowAnsibleDialog(false)}
                >
                  Cancel
                </Button>
              </div>
              {createAnsibleJobMutation.isError && (
                <p className="text-sm text-destructive">
                  Failed to create Ansible job
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
