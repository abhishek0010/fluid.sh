
import asyncio
import logging
from typing import Dict, List, Any, Optional

from mcp_client import BaseMCPClient
from tools import Tool, ToolRegistry

logger = logging.getLogger(__name__)

class MCPManager:
    """Manages multiple MCP server connections."""

    def __init__(self) -> None:
        self.clients: Dict[str, BaseMCPClient] = {}
        self.all_tools: List[Tool] = []

    async def add_server(self, name: str, command: str, args: List[str], env: Optional[Dict[str, str]] = None) -> List[Tool]:
        """Add and connect to an MCP server."""
        try:
            client = BaseMCPClient(command, args, env)
            tools = await client.connect()
            self.clients[name] = client
            self.all_tools.extend(tools)
            return tools
        except Exception as e:
            logger.error(f"Failed to connect to MCP server {name}: {e}")
            return []

    async def disconnect_all(self) -> None:
        """Disconnect all MCP servers."""
        for name, client in self.clients.items():
            try:
                await client.disconnect()
            except Exception as e:
                logger.error(f"Error disconnecting from {name}: {e}")
        self.clients.clear()
        self.all_tools.clear()

    def register_tools(self, registry: ToolRegistry) -> None:
        """Register all discovered tools in the given registry."""
        for tool in self.all_tools:
            registry.register(tool)
