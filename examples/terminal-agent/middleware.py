"""
Middleware system for the agent loop.

Provides a plugin architecture for hooking into tool execution events
without coupling specific behaviors to the agent loop itself.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from openai.types.chat import ChatCompletionMessageParam


@dataclass
class ToolExecutionContext:
    """Context passed to middleware after tool execution."""

    tool_name: str
    tool_args: dict[str, Any]
    result: dict[str, Any]
    error: bool
    tool_call_id: str


@dataclass
class MiddlewareResult:
    """Result from middleware processing."""

    messages_to_add: list[ChatCompletionMessageParam] = field(default_factory=list)


class Middleware(ABC):
    """
    Base class for agent middleware.

    Middleware can hook into tool execution events and optionally
    add messages to the conversation. Each middleware should be
    focused on a single concern.
    """

    @property
    @abstractmethod
    def name(self) -> str:
        """Unique name for this middleware."""
        ...

    @property
    def enabled(self) -> bool:
        """Whether this middleware is enabled. Can be overridden."""
        return True

    @abstractmethod
    def after_tool_execution(self, context: ToolExecutionContext) -> MiddlewareResult:
        """
        Called after a tool is executed.

        Args:
            context: Information about the tool execution

        Returns:
            MiddlewareResult with any messages to add to conversation
        """
        ...


class PlaybookNudgeMiddleware(Middleware):
    """
    Middleware that nudges the agent to track commands in playbooks.

    Adds a system hint after successful run_command executions to
    remind the agent to add state-modifying commands to the Ansible playbook.

    Configuration:
        enabled: Whether to add nudges (default: True)
        max_nudges: Maximum nudges per conversation to avoid context bloat (default: 3)
        target_tools: Which tools trigger the nudge (default: ["run_command"])
    """

    def __init__(
        self,
        enabled: bool = True,
        max_nudges: int = 3,
        target_tools: list[str] | None = None,
    ) -> None:
        """
        Initialize the playbook nudge middleware.

        Args:
            enabled: Whether nudges are enabled
            max_nudges: Max nudges before stopping (0 = unlimited)
            target_tools: Tool names that trigger nudges
        """
        self._enabled = enabled
        self.max_nudges = max_nudges
        self.target_tools = target_tools or ["run_command"]
        self._nudge_count = 0

    @property
    def name(self) -> str:
        return "playbook_nudge"

    @property
    def enabled(self) -> bool:
        return self._enabled

    def reset(self) -> None:
        """Reset nudge count. Call when conversation resets."""
        self._nudge_count = 0

    def after_tool_execution(self, context: ToolExecutionContext) -> MiddlewareResult:
        result = MiddlewareResult()

        if not self.enabled:
            return result

        # Check if we've hit the nudge limit
        if self.max_nudges > 0 and self._nudge_count >= self.max_nudges:
            return result

        # Only nudge for target tools that succeeded
        if context.tool_name not in self.target_tools:
            return result

        if context.error:
            return result

        # Add the nudge
        self._nudge_count += 1
        result.messages_to_add.append(
            {
                "role": "system",
                "content": (
                    "Hint: The command was successful. If this command modifies "
                    "system state, remember to add it to the Ansible playbook "
                    "using 'add_task'."
                ),
            }
        )

        return result


class MiddlewareChain:
    """
    Manages a chain of middleware instances.

    Middleware is executed in order of registration. Results from
    all middleware are aggregated.
    """

    def __init__(self, middlewares: list[Middleware] | None = None) -> None:
        """
        Initialize the middleware chain.

        Args:
            middlewares: Initial list of middleware instances
        """
        self._middlewares: list[Middleware] = middlewares or []

    def add(self, middleware: Middleware) -> None:
        """Add middleware to the chain."""
        self._middlewares.append(middleware)

    def remove(self, name: str) -> bool:
        """
        Remove middleware by name.

        Args:
            name: Name of middleware to remove

        Returns:
            True if middleware was found and removed
        """
        for i, m in enumerate(self._middlewares):
            if m.name == name:
                self._middlewares.pop(i)
                return True
        return False

    def get(self, name: str) -> Middleware | None:
        """Get middleware by name."""
        for m in self._middlewares:
            if m.name == name:
                return m
        return None

    def process_tool_execution(
        self, context: ToolExecutionContext
    ) -> list[dict[str, Any]]:
        """
        Process a tool execution through all middleware.

        Args:
            context: Tool execution context

        Returns:
            List of messages to add to conversation
        """
        messages: list[dict[str, Any]] = []

        for middleware in self._middlewares:
            if not middleware.enabled:
                continue

            result = middleware.after_tool_execution(context)
            messages.extend(result.messages_to_add)  # type: ignore

        return messages

    def reset(self) -> None:
        """Reset all middleware state."""
        for middleware in self._middlewares:
            reset_fn = getattr(middleware, "reset", None)
            if callable(reset_fn):
                reset_fn()

    def __len__(self) -> int:
        return len(self._middlewares)

    def __iter__(self):
        return iter(self._middlewares)


def create_default_middleware_chain(
    playbook_nudge_enabled: bool = True,
    playbook_nudge_max: int = 3,
) -> MiddlewareChain:
    """
    Create a middleware chain with default middleware.

    Args:
        playbook_nudge_enabled: Enable playbook nudge middleware
        playbook_nudge_max: Max playbook nudges (0 = unlimited)

    Returns:
        Configured MiddlewareChain
    """
    chain = MiddlewareChain()

    if playbook_nudge_enabled:
        chain.add(
            PlaybookNudgeMiddleware(
                enabled=True,
                max_nudges=playbook_nudge_max,
            )
        )

    return chain
