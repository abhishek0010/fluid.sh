"""Tests for the tool execution framework."""

from typing import Any

import pytest

from tools import Tool, ToolDefinition, ToolExecutionResult, ToolRegistry


class EchoTool(Tool):
    """Simple test tool that echoes input."""

    @property
    def name(self) -> str:
        return "echo"

    @property
    def description(self) -> str:
        return "Echo the input message"

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "message": {"type": "string", "description": "Message to echo"},
            },
            "required": ["message"],
        }

    def execute(self, message: str) -> ToolExecutionResult:
        return ToolExecutionResult(success=True, data={"echo": message})


class FailingTool(Tool):
    """Tool that always fails."""

    @property
    def name(self) -> str:
        return "fail"

    @property
    def description(self) -> str:
        return "A tool that always fails"

    @property
    def parameters(self) -> dict[str, Any]:
        return {"type": "object", "properties": {}}

    def execute(self) -> ToolExecutionResult:
        return ToolExecutionResult(
            success=False, data={}, error_message="Intentional failure"
        )


class RaisingTool(Tool):
    """Tool that raises an exception."""

    @property
    def name(self) -> str:
        return "raise"

    @property
    def description(self) -> str:
        return "A tool that raises"

    @property
    def parameters(self) -> dict[str, Any]:
        return {"type": "object", "properties": {}}

    def execute(self) -> ToolExecutionResult:
        raise RuntimeError("Unexpected error")


class TestToolDefinition:
    """Tests for ToolDefinition."""

    def test_to_openai_format(self) -> None:
        """Test conversion to OpenAI format."""
        definition = ToolDefinition(
            name="test_tool",
            description="A test tool",
            parameters={
                "type": "object",
                "properties": {"value": {"type": "string"}},
            },
        )

        result = definition.to_openai_format()

        assert result["type"] == "function"
        assert result["function"]["name"] == "test_tool"
        assert result["function"]["description"] == "A test tool"
        assert result["function"]["parameters"]["type"] == "object"


class TestToolExecutionResult:
    """Tests for ToolExecutionResult."""

    def test_success_to_dict(self) -> None:
        """Test successful result conversion."""
        result = ToolExecutionResult(success=True, data={"key": "value"})
        assert result.to_dict() == {"key": "value"}

    def test_error_to_dict(self) -> None:
        """Test error result conversion."""
        result = ToolExecutionResult(
            success=False, data={"context": "info"}, error_message="Something went wrong"
        )
        expected = {"error": "Something went wrong", "context": "info"}
        assert result.to_dict() == expected

    def test_error_without_message(self) -> None:
        """Test error with no message uses default."""
        result = ToolExecutionResult(success=False, data={})
        assert result.to_dict() == {"error": "Unknown error"}


class TestTool:
    """Tests for Tool base class."""

    def test_get_definition(self) -> None:
        """Test getting tool definition."""
        tool = EchoTool()
        definition = tool.get_definition()

        assert definition.name == "echo"
        assert definition.description == "Echo the input message"
        assert "message" in definition.parameters["properties"]

    def test_execute_success(self) -> None:
        """Test successful execution."""
        tool = EchoTool()
        result = tool.execute(message="hello")

        assert result.success
        assert result.data == {"echo": "hello"}

    def test_execute_failure(self) -> None:
        """Test failed execution."""
        tool = FailingTool()
        result = tool.execute()

        assert not result.success
        assert result.error_message == "Intentional failure"


class TestToolRegistry:
    """Tests for ToolRegistry."""

    @pytest.fixture
    def registry(self) -> ToolRegistry:
        """Create a fresh registry."""
        return ToolRegistry()

    def test_register_tool(self, registry: ToolRegistry) -> None:
        """Test registering a tool."""
        tool = EchoTool()
        registry.register(tool)

        assert "echo" in registry
        assert len(registry) == 1

    def test_register_duplicate_raises(self, registry: ToolRegistry) -> None:
        """Test registering duplicate tool raises."""
        tool = EchoTool()
        registry.register(tool)

        with pytest.raises(ValueError, match="already registered"):
            registry.register(tool)

    def test_unregister_tool(self, registry: ToolRegistry) -> None:
        """Test unregistering a tool."""
        tool = EchoTool()
        registry.register(tool)
        registry.unregister("echo")

        assert "echo" not in registry
        assert len(registry) == 0

    def test_unregister_missing_raises(self, registry: ToolRegistry) -> None:
        """Test unregistering missing tool raises."""
        with pytest.raises(KeyError, match="not found"):
            registry.unregister("nonexistent")

    def test_get_tool(self, registry: ToolRegistry) -> None:
        """Test getting a tool."""
        tool = EchoTool()
        registry.register(tool)

        retrieved = registry.get("echo")
        assert retrieved is tool

    def test_get_missing_returns_none(self, registry: ToolRegistry) -> None:
        """Test getting missing tool returns None."""
        assert registry.get("nonexistent") is None

    def test_execute_tool(self, registry: ToolRegistry) -> None:
        """Test executing a tool."""
        registry.register(EchoTool())

        result = registry.execute("echo", {"message": "test"})
        assert result == {"echo": "test"}

    def test_execute_missing_tool(self, registry: ToolRegistry) -> None:
        """Test executing missing tool returns error."""
        result = registry.execute("nonexistent", {})
        assert "error" in result
        assert "not found" in result["error"]

    def test_execute_with_invalid_args(self, registry: ToolRegistry) -> None:
        """Test executing with invalid args."""
        registry.register(EchoTool())

        result = registry.execute("echo", {"wrong_arg": "value"})
        assert "error" in result
        assert "Invalid arguments" in result["error"]

    def test_execute_raising_tool(self, registry: ToolRegistry) -> None:
        """Test executing tool that raises exception."""
        registry.register(RaisingTool())

        result = registry.execute("raise", {})
        assert "error" in result
        assert "failed" in result["error"]

    def test_execute_failing_tool(self, registry: ToolRegistry) -> None:
        """Test executing tool that returns failure."""
        registry.register(FailingTool())

        result = registry.execute("fail", {})
        assert "error" in result
        assert result["error"] == "Intentional failure"

    def test_get_definitions(self, registry: ToolRegistry) -> None:
        """Test getting all definitions."""
        registry.register(EchoTool())
        registry.register(FailingTool())

        definitions = registry.get_definitions()

        assert len(definitions) == 2
        names = {d["function"]["name"] for d in definitions}
        assert names == {"echo", "fail"}

    def test_list_tools(self, registry: ToolRegistry) -> None:
        """Test listing tool names."""
        registry.register(EchoTool())
        registry.register(FailingTool())

        names = registry.list_tools()
        assert set(names) == {"echo", "fail"}

    def test_empty_registry(self, registry: ToolRegistry) -> None:
        """Test empty registry."""
        assert len(registry) == 0
        assert registry.get_definitions() == []
        assert registry.list_tools() == []
