"""Tests for the TUI module."""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock

import pytest
from textual.widgets import Input, Label

from agent import AgentLoop, AgentResponse, ToolResult
from tui import (
    ConversationView,
    Message,
    StatusBar,
    TerminalAgentApp,
    ThinkingIndicator,
    ToolCallDisplay,
)


class MockProvider:
    """Mock LLM provider for testing."""

    def __init__(self, responses: list[Any] | None = None) -> None:
        self.responses = responses or []
        self.call_count = 0

    def chat_completion(
        self,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        tool_choice: str | None = None,
    ) -> Any:
        if self.call_count < len(self.responses):
            response = self.responses[self.call_count]
            self.call_count += 1
            return response
        raise RuntimeError("No more mock responses")


def create_mock_response(content: str, tool_calls: list[Any] | None = None) -> Any:
    """Create a mock chat completion response."""
    mock = MagicMock()
    mock.choices = [MagicMock()]
    mock.choices[0].message.content = content
    mock.choices[0].message.tool_calls = tool_calls
    mock.choices[0].message.model_dump.return_value = {
        "role": "assistant",
        "content": content,
        "tool_calls": tool_calls,
    }
    return mock


def create_test_agent() -> AgentLoop:
    """Create a test agent with mock provider."""
    provider = MockProvider([create_mock_response("Hello, I can help you.")])
    return AgentLoop(
        provider=provider,
        system_prompt="Test system prompt",
        tools=[],
        tool_handler=lambda name, args: {},
    )


class TestMessage:
    """Tests for Message widget."""

    def test_message_user_role(self) -> None:
        msg = Message("Hello", role="user")
        assert msg.content == "Hello"
        assert msg.role == "user"

    def test_message_assistant_role(self) -> None:
        msg = Message("Hi there", role="assistant")
        assert msg.content == "Hi there"
        assert msg.role == "assistant"


class TestToolCallDisplay:
    """Tests for ToolCallDisplay widget."""

    def test_tool_ok_status(self) -> None:
        result = ToolResult(
            tool_call_id="1",
            name="test_tool",
            result={"output": "success"},
            error=False
        )
        display = ToolCallDisplay(result)
        assert display.result.name == "test_tool"
        assert not display.result.error

    def test_tool_err_status(self) -> None:
        result = ToolResult(
            tool_call_id="2",
            name="failing_tool",
            result={"error": "Something went wrong"},
            error=True
        )
        display = ToolCallDisplay(result)
        assert display.result.name == "failing_tool"
        assert display.result.error


class TestThinkingIndicator:
    """Tests for ThinkingIndicator widget."""

    def test_indicator_init(self) -> None:
        indicator = ThinkingIndicator()
        assert indicator._dots == 0

    def test_stop_method_exists(self) -> None:
        indicator = ThinkingIndicator()
        assert hasattr(indicator, "stop")
        assert callable(indicator.stop)


class TestStatusBar:
    """Tests for StatusBar widget."""

    def test_status_bar_init(self) -> None:
        bar = StatusBar(provider="openai", model="gpt-4o")
        assert bar.provider == "openai"
        assert bar.model == "gpt-4o"


class TestTerminalAgentApp:
    """Tests for TerminalAgentApp."""

    def test_app_init(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(
            agent=agent,
            provider_type="openai",
            model="gpt-4o",
        )
        assert app.agent is agent
        assert app.provider_type == "openai"
        assert app.model == "gpt-4o"
        assert app._thinking_indicator is None

    def test_app_bindings(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)
        binding_keys = [b.key for b in app.BINDINGS]
        assert "ctrl+c" in binding_keys
        assert "ctrl+r" in binding_keys
        assert "escape" in binding_keys


@pytest.mark.asyncio
class TestTerminalAgentAppAsync:
    """Async tests for TerminalAgentApp using textual's test driver."""

    async def test_app_startup(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(
            agent=agent,
            provider_type="test",
            model="test-model",
        )

        async with app.run_test() as pilot:
            # Check app mounted correctly
            assert app.query_one("#conversation", ConversationView) is not None
            assert app.query_one("#user-input", Input) is not None

    async def test_app_shows_initial_message(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            conv = app.query_one("#conversation", ConversationView)
            # Should have at least one child (the welcome message)
            assert len(conv.children) >= 1

    async def test_app_input_focused_on_mount(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            input_widget = app.query_one("#user-input", Input)
            assert input_widget.has_focus

    async def test_app_reset_action(self) -> None:
        agent = create_test_agent()
        # Add a message to history
        agent.add_user_message("test")
        initial_msg_count = len(agent.messages)

        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            # Trigger reset action
            await pilot.press("ctrl+r")

            # Agent messages should be reset (only system prompt remains)
            assert len(agent.messages) == 1
            assert agent.messages[0]["role"] == "system"

    async def test_empty_input_ignored(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            conv = app.query_one("#conversation", ConversationView)
            initial_children = len(conv.children)

            # Submit empty input
            input_widget = app.query_one("#user-input", Input)
            input_widget.value = ""
            await pilot.press("enter")

            # No new messages should be added
            assert len(conv.children) == initial_children


class TestConversationView:
    """Tests for ConversationView."""

    @pytest.mark.asyncio
    async def test_add_message(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            conv = app.query_one("#conversation", ConversationView)
            initial_count = len(conv.children)

            conv.add_message("Test message", role="assistant")
            await pilot.pause()

            assert len(conv.children) == initial_count + 1

    @pytest.mark.asyncio
    async def test_add_tool_result(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            conv = app.query_one("#conversation", ConversationView)
            initial_count = len(conv.children)

            result = ToolResult(
                tool_call_id="1",
                name="test_tool",
                result={"output": "success"},
                error=False
            )
            conv.add_tool_result(result)
            await pilot.pause()

            assert len(conv.children) == initial_count + 1

    @pytest.mark.asyncio
    async def test_add_tool_result_error(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            conv = app.query_one("#conversation", ConversationView)
            initial_count = len(conv.children)

            result = ToolResult(
                tool_call_id="2",
                name="failing_tool",
                result={"error": "failed"},
                error=True
            )
            conv.add_tool_result(result)
            await pilot.pause()

            assert len(conv.children) == initial_count + 1

    @pytest.mark.asyncio
    async def test_show_thinking(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            conv = app.query_one("#conversation", ConversationView)
            indicator = conv.show_thinking()
            await pilot.pause()

            assert isinstance(indicator, ThinkingIndicator)
            # Indicator should be in conversation view
            assert indicator in conv.children

    @pytest.mark.asyncio
    async def test_clear_conversation(self) -> None:
        agent = create_test_agent()
        app = TerminalAgentApp(agent=agent)

        async with app.run_test() as pilot:
            conv = app.query_one("#conversation", ConversationView)
            # Add some messages
            conv.add_message("Message 1")
            conv.add_message("Message 2")
            await pilot.pause()

            conv.clear_conversation()
            await pilot.pause()

            assert len(conv.children) == 0
