"""Tests for the middleware system."""

import pytest

from middleware import (
    Middleware,
    MiddlewareChain,
    MiddlewareResult,
    PlaybookNudgeMiddleware,
    ToolExecutionContext,
    create_default_middleware_chain,
)


class TestToolExecutionContext:
    """Tests for ToolExecutionContext dataclass."""

    def test_context_creation(self) -> None:
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "ls -la"},
            result={"output": "file.txt"},
            error=False,
            tool_call_id="call_123",
        )
        assert ctx.tool_name == "run_command"
        assert ctx.tool_args == {"command": "ls -la"}
        assert ctx.result == {"output": "file.txt"}
        assert ctx.error is False
        assert ctx.tool_call_id == "call_123"


class TestMiddlewareResult:
    """Tests for MiddlewareResult dataclass."""

    def test_default_empty_messages(self) -> None:
        result = MiddlewareResult()
        assert result.messages_to_add == []

    def test_with_messages(self) -> None:
        result = MiddlewareResult(
            messages_to_add=[{"role": "system", "content": "test"}]
        )
        assert len(result.messages_to_add) == 1


class TestPlaybookNudgeMiddleware:
    """Tests for PlaybookNudgeMiddleware."""

    def test_name(self) -> None:
        middleware = PlaybookNudgeMiddleware()
        assert middleware.name == "playbook_nudge"

    def test_enabled_by_default(self) -> None:
        middleware = PlaybookNudgeMiddleware()
        assert middleware.enabled is True

    def test_can_disable(self) -> None:
        middleware = PlaybookNudgeMiddleware(enabled=False)
        assert middleware.enabled is False

    def test_nudge_on_successful_run_command(self) -> None:
        middleware = PlaybookNudgeMiddleware()
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "apt install nginx"},
            result={"output": "success"},
            error=False,
            tool_call_id="call_1",
        )
        result = middleware.after_tool_execution(ctx)
        assert len(result.messages_to_add) == 1
        assert result.messages_to_add[0]["role"] == "system"
        assert "add_task" in result.messages_to_add[0]["content"]

    def test_no_nudge_on_error(self) -> None:
        middleware = PlaybookNudgeMiddleware()
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "invalid_command"},
            result={"error": "command not found"},
            error=True,
            tool_call_id="call_1",
        )
        result = middleware.after_tool_execution(ctx)
        assert len(result.messages_to_add) == 0

    def test_no_nudge_for_other_tools(self) -> None:
        middleware = PlaybookNudgeMiddleware()
        ctx = ToolExecutionContext(
            tool_name="view_playbook",
            tool_args={},
            result={"playbook": "content"},
            error=False,
            tool_call_id="call_1",
        )
        result = middleware.after_tool_execution(ctx)
        assert len(result.messages_to_add) == 0

    def test_no_nudge_when_disabled(self) -> None:
        middleware = PlaybookNudgeMiddleware(enabled=False)
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "apt install nginx"},
            result={"output": "success"},
            error=False,
            tool_call_id="call_1",
        )
        result = middleware.after_tool_execution(ctx)
        assert len(result.messages_to_add) == 0

    def test_max_nudges_limit(self) -> None:
        middleware = PlaybookNudgeMiddleware(max_nudges=2)
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "cmd"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_1",
        )

        # First two should produce nudges
        result1 = middleware.after_tool_execution(ctx)
        assert len(result1.messages_to_add) == 1

        result2 = middleware.after_tool_execution(ctx)
        assert len(result2.messages_to_add) == 1

        # Third should be blocked by limit
        result3 = middleware.after_tool_execution(ctx)
        assert len(result3.messages_to_add) == 0

    def test_unlimited_nudges_with_zero(self) -> None:
        middleware = PlaybookNudgeMiddleware(max_nudges=0)
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "cmd"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_1",
        )

        # Should keep producing nudges
        for _ in range(10):
            result = middleware.after_tool_execution(ctx)
            assert len(result.messages_to_add) == 1

    def test_reset_clears_nudge_count(self) -> None:
        middleware = PlaybookNudgeMiddleware(max_nudges=1)
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "cmd"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_1",
        )

        # Use up the nudge
        result1 = middleware.after_tool_execution(ctx)
        assert len(result1.messages_to_add) == 1

        result2 = middleware.after_tool_execution(ctx)
        assert len(result2.messages_to_add) == 0

        # Reset
        middleware.reset()

        # Should nudge again
        result3 = middleware.after_tool_execution(ctx)
        assert len(result3.messages_to_add) == 1

    def test_custom_target_tools(self) -> None:
        middleware = PlaybookNudgeMiddleware(target_tools=["custom_tool"])
        ctx1 = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "cmd"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_1",
        )
        ctx2 = ToolExecutionContext(
            tool_name="custom_tool",
            tool_args={"arg": "value"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_2",
        )

        # run_command should not trigger nudge
        result1 = middleware.after_tool_execution(ctx1)
        assert len(result1.messages_to_add) == 0

        # custom_tool should trigger nudge
        result2 = middleware.after_tool_execution(ctx2)
        assert len(result2.messages_to_add) == 1


class TestMiddlewareChain:
    """Tests for MiddlewareChain."""

    def test_empty_chain(self) -> None:
        chain = MiddlewareChain()
        assert len(chain) == 0

    def test_add_middleware(self) -> None:
        chain = MiddlewareChain()
        middleware = PlaybookNudgeMiddleware()
        chain.add(middleware)
        assert len(chain) == 1

    def test_remove_middleware(self) -> None:
        chain = MiddlewareChain()
        middleware = PlaybookNudgeMiddleware()
        chain.add(middleware)
        assert chain.remove("playbook_nudge") is True
        assert len(chain) == 0

    def test_remove_nonexistent_middleware(self) -> None:
        chain = MiddlewareChain()
        assert chain.remove("nonexistent") is False

    def test_get_middleware(self) -> None:
        chain = MiddlewareChain()
        middleware = PlaybookNudgeMiddleware()
        chain.add(middleware)
        assert chain.get("playbook_nudge") is middleware
        assert chain.get("nonexistent") is None

    def test_process_tool_execution(self) -> None:
        chain = MiddlewareChain()
        chain.add(PlaybookNudgeMiddleware())
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "cmd"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_1",
        )
        messages = chain.process_tool_execution(ctx)
        assert len(messages) == 1

    def test_process_skips_disabled_middleware(self) -> None:
        chain = MiddlewareChain()
        chain.add(PlaybookNudgeMiddleware(enabled=False))
        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "cmd"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_1",
        )
        messages = chain.process_tool_execution(ctx)
        assert len(messages) == 0

    def test_reset_all_middleware(self) -> None:
        chain = MiddlewareChain()
        middleware = PlaybookNudgeMiddleware(max_nudges=1)
        chain.add(middleware)

        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={"command": "cmd"},
            result={"output": "ok"},
            error=False,
            tool_call_id="call_1",
        )

        # Use up nudge
        chain.process_tool_execution(ctx)
        msgs1 = chain.process_tool_execution(ctx)
        assert len(msgs1) == 0

        # Reset chain
        chain.reset()

        # Should nudge again
        msgs2 = chain.process_tool_execution(ctx)
        assert len(msgs2) == 1

    def test_iteration(self) -> None:
        chain = MiddlewareChain()
        m1 = PlaybookNudgeMiddleware()
        chain.add(m1)

        middlewares = list(chain)
        assert len(middlewares) == 1
        assert middlewares[0] is m1


class TestCreateDefaultMiddlewareChain:
    """Tests for create_default_middleware_chain factory."""

    def test_default_creates_playbook_nudge(self) -> None:
        chain = create_default_middleware_chain()
        assert chain.get("playbook_nudge") is not None

    def test_can_disable_playbook_nudge(self) -> None:
        chain = create_default_middleware_chain(playbook_nudge_enabled=False)
        assert chain.get("playbook_nudge") is None

    def test_custom_max_nudges(self) -> None:
        chain = create_default_middleware_chain(playbook_nudge_max=5)
        middleware = chain.get("playbook_nudge")
        assert middleware is not None
        assert isinstance(middleware, PlaybookNudgeMiddleware)
        assert middleware.max_nudges == 5


class TestCustomMiddleware:
    """Tests for creating custom middleware."""

    def test_custom_middleware(self) -> None:
        class LoggingMiddleware(Middleware):
            """Custom middleware that logs tool calls."""

            def __init__(self) -> None:
                self.calls: list[str] = []

            @property
            def name(self) -> str:
                return "logging"

            def after_tool_execution(
                self, context: ToolExecutionContext
            ) -> MiddlewareResult:
                self.calls.append(context.tool_name)
                return MiddlewareResult()

        chain = MiddlewareChain()
        logging_mw = LoggingMiddleware()
        chain.add(logging_mw)

        ctx = ToolExecutionContext(
            tool_name="run_command",
            tool_args={},
            result={},
            error=False,
            tool_call_id="call_1",
        )
        chain.process_tool_execution(ctx)
        chain.process_tool_execution(ctx)

        assert logging_mw.calls == ["run_command", "run_command"]
