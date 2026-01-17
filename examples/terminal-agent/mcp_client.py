
import asyncio
from typing import Any, Optional, Dict, List
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

from tools import Tool, ToolDefinition, ToolExecutionResult

class MCPTool(Tool):
    """A tool that proxies calls to an MCP server."""

    def __init__(self, session: ClientSession, name: str, description: str, input_schema: Dict[str, Any]) -> None:
        self.session = session
        self._name = name
        self._description = description
        self._parameters = input_schema

    @property
    def name(self) -> str:
        return f"mcp_{self._name}"

    @property
    def description(self) -> str:
        return self._description

    @property
    def parameters(self) -> Dict[str, Any]:
        return self._parameters

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        """Execute the tool on the MCP server."""
        try:
            result = await self.session.call_tool(self._name, kwargs)
            return ToolExecutionResult(
                success=not result.is_error,
                data={"content": [c.model_dump() for c in result.content]},
                error_message=None if not result.is_error else "MCP tool execution failed"
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e)
            )

class BaseMCPClient:
    """Base client for connecting to MCP servers."""

    def __init__(self, command: str, args: List[str], env: Optional[Dict[str, str]] = None) -> None:
        self.server_params = StdioServerParameters(
            command=command,
            args=args,
            env=env
        )
        self.session: Optional[ClientSession] = None
        self._client_context = None

    async def connect(self) -> List[Tool]:
        """Connect to the MCP server and discover tools."""
        self._client_context = stdio_client(self.server_params)
        read, write = await self._client_context.__aenter__()
        self.session = ClientSession(read, write)
        await self.session.__aenter__()
        await self.session.initialize()
        
        # List tools
        mcp_tools = await self.session.list_tools()
        
        tools = []
        for tool in mcp_tools.tools:
            tools.append(MCPTool(
                session=self.session,
                name=tool.name,
                description=tool.description or "",
                input_schema=tool.inputSchema
            ))
        return tools

    async def disconnect(self) -> None:
        """Disconnect from the MCP server."""
        if self.session:
            await self.session.__aexit__(None, None, None)
        if self._client_context:
            await self._client_context.__aexit__(None, None, None)
