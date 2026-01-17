"""
Tool execution framework for the terminal agent.

Provides base classes and registry for defining and executing tools.
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Any


@dataclass
class ToolDefinition:
    """OpenAI-compatible tool definition."""

    name: str
    description: str
    parameters: dict[str, Any]

    def to_openai_format(self) -> dict[str, Any]:
        """Convert to OpenAI tool format."""
        return {
            "type": "function",
            "function": {
                "name": self.name,
                "description": self.description,
                "parameters": self.parameters,
            },
        }


@dataclass
class ToolExecutionResult:
    """Result from executing a tool."""

    success: bool
    data: dict[str, Any]
    error_message: str | None = None

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        if self.success:
            return self.data
        return {"error": self.error_message or "Unknown error", **self.data}


class Tool(ABC):
    """
    Base class for all tools.

    Tools encapsulate a single operation the agent can perform.
    Subclasses must implement the execute method and define their schema.
    """

    @property
    @abstractmethod
    def name(self) -> str:
        """Unique name for this tool."""
        ...

    @property
    @abstractmethod
    def description(self) -> str:
        """Description of what this tool does."""
        ...

    @property
    @abstractmethod
    def parameters(self) -> dict[str, Any]:
        """JSON Schema for the tool's parameters."""
        ...

    @abstractmethod
    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        """
        Execute the tool with given parameters.

        Args:
            **kwargs: Tool-specific parameters

        Returns:
            ToolExecutionResult with success status and data
        """
        ...

    def get_definition(self) -> ToolDefinition:
        """Get the tool definition for LLM."""
        return ToolDefinition(
            name=self.name,
            description=self.description,
            parameters=self.parameters,
        )


class ToolRegistry:
    """
    Registry for tools available to the agent.

    Manages tool registration, lookup, and execution.
    """

    def __init__(self) -> None:
        self._tools: dict[str, Tool] = {}

    def register(self, tool: Tool) -> None:
        """
        Register a tool.

        Args:
            tool: Tool instance to register

        Raises:
            ValueError: If tool with same name already registered
        """
        if tool.name in self._tools:
            raise ValueError(f"Tool '{tool.name}' already registered")
        self._tools[tool.name] = tool

    def unregister(self, name: str) -> None:
        """
        Unregister a tool by name.

        Args:
            name: Name of tool to unregister

        Raises:
            KeyError: If tool not found
        """
        if name not in self._tools:
            raise KeyError(f"Tool '{name}' not found")
        del self._tools[name]

    def get(self, name: str) -> Tool | None:
        """
        Get a tool by name.

        Args:
            name: Tool name

        Returns:
            Tool instance or None if not found
        """
        return self._tools.get(name)

    async def execute(self, name: str, args: dict[str, Any]) -> dict[str, Any]:
        """
        Execute a tool by name.

        Args:
            name: Tool name
            args: Arguments to pass to tool

        Returns:
            Dictionary with tool result or error
        """
        tool = self._tools.get(name)
        if tool is None:
            return {"error": f"Tool '{name}' not found"}

        try:
            result = await tool.execute(**args)
            return result.to_dict()
        except TypeError as e:
            return {"error": f"Invalid arguments for '{name}': {e}"}
        except Exception as e:
            return {"error": f"Tool '{name}' failed: {e}"}

    def get_definitions(self) -> list[dict[str, Any]]:
        """
        Get OpenAI-format definitions for all registered tools.

        Returns:
            List of tool definitions in OpenAI format
        """
        return [tool.get_definition().to_openai_format() for tool in self._tools.values()]

    def list_tools(self) -> list[str]:
        """
        List names of all registered tools.

        Returns:
            List of tool names
        """
        return list(self._tools.keys())

    def __len__(self) -> int:
        return len(self._tools)

    def __contains__(self, name: str) -> bool:
        return name in self._tools
