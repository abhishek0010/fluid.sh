import { Link, useRouterState } from "@tanstack/react-router";
import { Server, Box, Play } from "lucide-react";
import { cn } from "~/lib/utils";

const navItems = [
  { to: "/", label: "VMs", icon: Server },
  { to: "/sandboxes", label: "Sandboxes", icon: Box },
  { to: "/ansible", label: "Ansible Runs", icon: Play },
] as const;

export function Navbar() {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  return (
    <nav className="border-b bg-card">
      <div className="container mx-auto px-4">
        <div className="flex h-14 items-center gap-6">
          <Link to="/" className="font-bold text-lg">
            virsh-sandbox
          </Link>
          <div className="flex items-center gap-1">
            {navItems.map((item) => {
              const isActive =
                item.to === "/"
                  ? currentPath === "/"
                  : currentPath.startsWith(item.to);
              const Icon = item.icon;
              return (
                <Link
                  key={item.to}
                  to={item.to}
                  className={cn(
                    "flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                    isActive
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                  )}
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </Link>
              );
            })}
          </div>
        </div>
      </div>
    </nav>
  );
}
