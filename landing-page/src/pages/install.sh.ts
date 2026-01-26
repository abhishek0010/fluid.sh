import type { APIRoute } from "astro";
import { PostHog } from "posthog-node";

export const GET: APIRoute = async ({ request }) => {
  const client = new PostHog(import.meta.env.PUBLIC_POSTHOG_API_KEY, {
    host: import.meta.env.PUBLIC_POSTHOG_HOST,
  });

  // Log the install event to PostHog
  try {
    const userAgent = request.headers.get("user-agent") || "unknown";
    const ip =
      request.headers.get("x-forwarded-for") ||
      request.headers.get("x-real-ip") ||
      "unknown";

    client.capture({
      distinctId: ip,
      event: "install_script_fetched",
      properties: {
        $os: userAgent.includes("Darwin")
          ? "macOS"
          : userAgent.includes("Linux")
            ? "Linux"
            : "unknown",
        $browser: "curl/wget",
        user_agent: userAgent,
        ip: ip,
      },
    });

    // Send queued events immediately
    await client.shutdown();
  } catch (e) {
    console.error("Failed to log to PostHog:", e);
  }

  const script = `#!/bin/sh
set -e

# Install Fluid
echo "Installing Fluid..."

if ! command -v go &> /dev/null; then
    echo "Error: 'go' is not installed. Please install Go first: https://go.dev/doc/install"
    exit 1
fi

echo "Running: go install github.com/aspectrr/fluid@latest"
go install github.com/aspectrr/fluid@latest

echo ""
echo "Fluid installed successfully!"
echo "Ensure that $(go env GOPATH)/bin is in your PATH."
echo "Run 'fluid --help' to get started."
`;

  return new Response(script, {
    headers: {
      "Content-Type": "text/plain",
    },
  });
};
