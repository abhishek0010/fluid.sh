"""Tests for the agent loop."""

import json
from typing import Any
from unittest.mock import MagicMock, patch

import pytest

from agent import AgentLoop, AgentResponse, ToolResult


class MockMessage:
    """Mock OpenAI message."""

    def __init__(
        self,
        content: str | None = None,
        tool_calls: list[Any] | None = None,
    ) -> None:
        self.content = content
        self.tool_calls = tool_calls

    def model_dump(self) -> dict[str, Any]:
        return {
            "role": "assistant",
            "content": self.content,
            "tool_calls": (
                [
                    {
                        "id": tc.id,
                        "type": "function",
                        "function": {
                            "name": tc.function.name,
                            "arguments": tc.function.arguments,
                        },
                    }
                    for tc in self.tool_calls
                ]
                if self.tool_calls
                else None
            ),
        }


class MockToolCall:
    """Mock tool call."""

    def __init__(self, id: str, name: str, arguments: str) -> None:
        self.id = id
        self.function = MagicMock()
        self.function.name = name
        self.function.arguments = arguments


class MockChoice:
    """Mock choice."""

    def __init__(self, message: MockMessage) -> None:
        self.message = message


class MockResponse:
    """Mock OpenAI response."""

    def __init__(self, message: MockMessage) -> None:
        self.choices = [MockChoice(message)]


class MockProvider:
    """Mock LLM provider for testing."""

    def __init__(self) -> None:
        self._mock = MagicMock()
        self._name = "mock"
        self._model = "test-model"

    @property
    def name(self) -> str:
        return self._name

    @property
    def model(self) -> str:
        return self._model

    def chat_completion(
        self,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        tool_choice: str | None = None,
    ) -> Any:
        return self._mock.chat_completion(messages, tools, tool_choice)

    def set_response(self, response: Any) -> None:
        """Set the mock response for testing."""
        self._mock.chat_completion.return_value = response

    def set_responses(self, responses: list[Any]) -> None:
        """Set multiple mock responses for testing."""
        self._mock.chat_completion.side_effect = responses


@pytest.fixture
def mock_provider() -> MockProvider:
    """Create a mock LLM provider."""
    return MockProvider()


@pytest.fixture
def simple_tool_handler() -> Any:
    """Create a simple tool handler for testing."""

    def handler(name: str, args: dict[str, Any]) -> dict[str, Any]:
        if name == "test_tool":
            return {"result": "success", "input": args}
        return {"error": f"Unknown tool: {name}"}

    return handler


@pytest.fixture
def agent(mock_provider: MockProvider, simple_tool_handler: Any) -> AgentLoop:
    """Create an agent for testing."""
    tools = [
        {
            "type": "function",
            "function": {
                "name": "test_tool",
                "description": "A test tool",
                "parameters": {
                    "type": "object",
                    "properties": {"value": {"type": "string"}},
                    "required": ["value"],
                },
            },
        }
    ]
    return AgentLoop(
        provider=mock_provider,
        system_prompt="You are a test agent.",
        tools=tools,
        tool_handler=simple_tool_handler,
    )


def test_agent_initialization(agent: AgentLoop) -> None:
    """Test agent initializes with system message."""
    assert len(agent.messages) == 1
    assert agent.messages[0]["role"] == "system"
    assert agent.messages[0]["content"] == "You are a test agent."


def test_add_user_message(agent: AgentLoop) -> None:
    """Test adding user message."""
    agent.add_user_message("Hello")
    assert len(agent.messages) == 2
    assert agent.messages[1]["role"] == "user"
    assert agent.messages[1]["content"] == "Hello"


def test_step_with_text_response(agent: AgentLoop, mock_provider: MockProvider) -> None:
    """Test step with a text response (no tools)."""
    mock_provider.set_response(MockResponse(
        MockMessage(content="Hello! How can I help?")
    ))

    agent.add_user_message("Hi")
    response = agent.step()

    assert response.content == "Hello! How can I help?"
    assert len(response.tool_calls) == 0
    assert len(response.tool_results) == 0
    assert not response.done


def test_step_with_tool_call(agent: AgentLoop, mock_provider: MockProvider) -> None:
    """Test step with a tool call."""
    tool_call = MockToolCall(
        id="call_123",
        name="test_tool",
        arguments='{"value": "test"}',
    )
    mock_provider.set_response(MockResponse(
        MockMessage(tool_calls=[tool_call])
    ))

    agent.add_user_message("Use the test tool")
    response = agent.step()

    assert response.content is None
    assert len(response.tool_calls) == 1
    assert len(response.tool_results) == 1
    assert response.tool_results[0].name == "test_tool"
    assert response.tool_results[0].result == {
        "result": "success",
        "input": {"value": "test"},
    }
    assert not response.tool_results[0].error


def test_step_with_invalid_tool_args(
    agent: AgentLoop, mock_provider: MockProvider
) -> None:
    """Test step with invalid JSON in tool arguments."""
    tool_call = MockToolCall(
        id="call_123",
        name="test_tool",
        arguments="invalid json",
    )
    mock_provider.set_response(MockResponse(
        MockMessage(tool_calls=[tool_call])
    ))

    agent.add_user_message("Use the test tool")
    response = agent.step()

    assert len(response.tool_results) == 1
    assert response.tool_results[0].error
    assert "Invalid JSON" in response.tool_results[0].result["error"]


def test_done_detection(agent: AgentLoop, mock_provider: MockProvider) -> None:
    """Test detection of completion phrases."""
    mock_provider.set_response(MockResponse(
        MockMessage(content="Task complete! I've finished the work.")
    ))

    agent.add_user_message("Do something")
    response = agent.step()

    assert response.done


def test_reset(agent: AgentLoop) -> None:
    """Test conversation reset."""
    agent.add_user_message("Hello")
    agent.add_user_message("World")
    assert len(agent.messages) == 3

    agent.reset()
    assert len(agent.messages) == 1
    assert agent.messages[0]["role"] == "system"


def test_reset_with_new_prompt(agent: AgentLoop) -> None:
    """Test reset with new system prompt."""
    agent.add_user_message("Hello")
    agent.reset(system_prompt="New prompt")

    assert len(agent.messages) == 1
    assert agent.messages[0]["content"] == "New prompt"


def test_run_until_done(agent: AgentLoop, mock_provider: MockProvider) -> None:
    """Test run method stops when done."""
    mock_provider.set_response(MockResponse(
        MockMessage(content="Task complete!")
    ))

    responses = agent.run("Do something")

    assert len(responses) == 1
    assert responses[0].done


def test_run_with_tool_then_done(agent: AgentLoop, mock_provider: MockProvider) -> None:
    """Test run method with tool call then completion."""
    tool_call = MockToolCall(
        id="call_123",
        name="test_tool",
        arguments='{"value": "test"}',
    )

    # First call returns tool, second call returns done
    mock_provider.set_responses([
        MockResponse(MockMessage(tool_calls=[tool_call])),
        MockResponse(MockMessage(content="Task complete!")),
    ])

    responses = agent.run("Use the tool then finish")

    assert len(responses) == 2
    assert len(responses[0].tool_calls) == 1
    assert responses[1].done


def test_run_max_turns(agent: AgentLoop, mock_provider: MockProvider) -> None:
    """Test run respects max_turns limit."""
    # Always return text without done phrase
    mock_provider.set_response(MockResponse(
        MockMessage(content="Still working...")
    ))

    responses = agent.run("Keep going", max_turns=3)

    # Should stop after first response since it has content but no tool calls
    assert len(responses) == 1


def test_tool_result_dataclass() -> None:
    """Test ToolResult dataclass."""
    result = ToolResult(
        tool_call_id="123",
        name="test",
        result={"key": "value"},
        error=False,
    )
    assert result.tool_call_id == "123"
    assert result.name == "test"
    assert result.result == {"key": "value"}
    assert not result.error


def test_agent_response_dataclass() -> None:
    """Test AgentResponse dataclass."""
    response = AgentResponse()
    assert response.content is None
    assert response.tool_calls == []
    assert response.tool_results == []
    assert not response.done
