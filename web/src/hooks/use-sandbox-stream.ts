import { useEffect, useRef, useState, useCallback } from "react";

// Stream event types matching backend
interface StreamEvent {
  type:
    | "connected"
    | "command_history"
    | "command_new"
    | "heartbeat"
    | "file_change";
  timestamp: string;
  data?: unknown;
  sandbox_id?: string;
}

interface CommandData {
  command_id: string;
  command: string;
  stdout?: string;
  stderr?: string;
  exit_code?: number;
  started_at: string;
  ended_at: string;
}

interface ConnectionData {
  sandbox_id: string;
  sandbox_name: string;
  state: string;
  ip_address?: string;
}

interface UseSandboxStreamOptions {
  onCommand?: (command: CommandData) => void;
  onConnected?: (data: ConnectionData) => void;
  onError?: (error: Event) => void;
  autoReconnect?: boolean;
  reconnectInterval?: number;
}

interface UseSandboxStreamReturn {
  isConnected: boolean;
  commands: CommandData[];
  connectionData: ConnectionData | null;
  error: string | null;
  connect: () => void;
  disconnect: () => void;
}

export function useSandboxStream(
  sandboxId: string | undefined,
  options: UseSandboxStreamOptions = {}
): UseSandboxStreamReturn {
  const {
    onCommand,
    onConnected,
    onError,
    autoReconnect = true,
    reconnectInterval = 5000,
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [commands, setCommands] = useState<CommandData[]>([]);
  const [connectionData, setConnectionData] = useState<ConnectionData | null>(
    null
  );
  const [error, setError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);

  const connect = useCallback(() => {
    if (!sandboxId) return;
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    // Determine WebSocket URL based on current location
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/v1/sandboxes/${sandboxId}/stream`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    ws.onmessage = (event) => {
      try {
        const streamEvent: StreamEvent = JSON.parse(event.data);

        switch (streamEvent.type) {
          case "connected": {
            const data = streamEvent.data as ConnectionData;
            setConnectionData(data);
            onConnected?.(data);
            break;
          }
          case "command_history":
          case "command_new": {
            const cmdData = streamEvent.data as CommandData;
            setCommands((prev) => {
              // Avoid duplicates
              if (prev.some((c) => c.command_id === cmdData.command_id)) {
                return prev;
              }
              return [...prev, cmdData];
            });
            onCommand?.(cmdData);
            break;
          }
          case "heartbeat":
            // Keep-alive, no action needed
            break;
        }
      } catch (e) {
        console.error("Failed to parse WebSocket message:", e);
      }
    };

    ws.onerror = (event) => {
      setError("WebSocket connection error");
      onError?.(event);
    };

    ws.onclose = () => {
      setIsConnected(false);
      wsRef.current = null;

      // Auto-reconnect if enabled
      if (autoReconnect && sandboxId) {
        reconnectTimeoutRef.current = window.setTimeout(() => {
          connect();
        }, reconnectInterval);
      }
    };
  }, [
    sandboxId,
    onCommand,
    onConnected,
    onError,
    autoReconnect,
    reconnectInterval,
  ]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  // Connect when sandboxId changes
  useEffect(() => {
    if (sandboxId) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [sandboxId, connect, disconnect]);

  return {
    isConnected,
    commands,
    connectionData,
    error,
    connect,
    disconnect,
  };
}
