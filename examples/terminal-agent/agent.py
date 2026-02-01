"""
Agent loop for the terminal agent.

Processes user input, calls LLM, executes tool calls, and returns responses.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Any, Callable

from openai.types.chat import ChatCompletionMessageToolCall, ChatCompletionMessageParam

from compact import (
    estimate_conversation_tokens,
    get_context_limit,
    compact_conversation,
)
from middleware import MiddlewareChain, ToolExecutionContext
from telemetry import get_telemetry

if TYPE_CHECKING:
    from llm import LLMProvider
    from tools import ToolRegistry


@dataclass
class ToolResult:
    """Result of a tool execution."""

    tool_call_id: str
    name: str
    args: dict[str, Any]
    result: dict[str, Any]
    error: bool = False


@dataclass
class AgentResponse:
    """Response from a single agent turn."""

    content: str | None = None
    tool_calls: list[ChatCompletionMessageToolCall] = field(default_factory=list)
    tool_results: list[ToolResult] = field(default_factory=list)
    done: bool = False
    awaiting_input: bool = False


ToolHandler = Callable[[str, dict[str, Any]], dict[str, Any]]


class AgentLoop:
    """
    Main agent loop that processes user input and executes tools.

    The loop:
    1. Takes user input
    2. Sends to LLM with conversation history
    3. Processes any tool calls
    4. Returns response or continues if tools were called
    """

    def __init__(
        self,
        provider: LLMProvider,
        system_prompt: str,
        tools: list[dict[str, Any]],
        tool_handler: ToolHandler,
        max_history_messages: int = 20,
        middleware: MiddlewareChain | None = None,
    ) -> None:
        """
        Initialize the agent loop.

        Args:
            provider: LLM provider instance (OpenAI, OpenRouter, local)
            system_prompt: System prompt for the agent
            tools: List of tool definitions in OpenAI format
            tool_handler: Function to execute tools, takes (name, args) returns dict
            max_history_messages: Maximum number of messages to keep in history (excluding system prompt)
            middleware: Optional middleware chain for hooking into tool execution
        """
        self.provider = provider
        self.tools = tools
        self.tool_handler = tool_handler
        self.max_history_messages = max_history_messages
        self.middleware = middleware or MiddlewareChain()
        self.messages: list[ChatCompletionMessageParam] = [
            {"role": "system", "content": system_prompt}
        ]

    def _prune_history(self) -> None:
        """Ensure message history doesn't exceed max_history_messages."""
        # Keep system prompt (index 0) and remove oldest messages if over limit
        while len(self.messages) > self.max_history_messages + 1:
            # Remove the message at index 1 (oldest non-system message)
            self.messages.pop(1)

    def add_user_message(self, content: str) -> None:
        """Add a user message to the conversation."""
        self.messages.append({"role": "user", "content": content})
        self._prune_history()

        # Track user prompt
        get_telemetry().track_user_prompt(
            prompt_length=len(content),
            message_count=len(self.messages),
        )

    async def _execute_tool(self, tool_call: ChatCompletionMessageToolCall) -> ToolResult:
        """Execute a single tool call."""
        name = tool_call.function.name
        telemetry = get_telemetry()

        try:
            args = json.loads(tool_call.function.arguments)
        except json.JSONDecodeError:
            telemetry.track_tool_result(tool_name=name, success=False, has_error=True)
            return ToolResult(
                tool_call_id=tool_call.id,
                name=name,
                args={},
                result={"error": "Invalid JSON in tool arguments"},
                error=True,
            )

        # Track tool call with arg keys only (not values for privacy)
        telemetry.track_tool_call(tool_name=name, args_keys=list(args.keys()))

        try:
            result = await self.tool_handler(name, args)
            is_error = isinstance(result, dict) and result.get("error", False)

            # Track tool result
            telemetry.track_tool_result(tool_name=name, success=not is_error, has_error=is_error)

            return ToolResult(
                tool_call_id=tool_call.id,
                name=name,
                args=args,
                result=result,
                error=is_error,
            )
        except Exception as e:
            telemetry.track_tool_result(tool_name=name, success=False, has_error=True)
            return ToolResult(
                tool_call_id=tool_call.id,
                name=name,
                args=args,
                result={"error": str(e)},
                error=True,
            )

    async def step(self) -> AgentResponse:
        """
        Execute a single step of the agent loop.

        Returns:
            AgentResponse with content, tool calls/results, and done status
        """
        response = self.provider.chat_completion(
            messages=self.messages,
            tools=self.tools if self.tools else None,
            tool_choice="auto" if self.tools else None,
        )

        msg = response.choices[0].message
        agent_response = AgentResponse()

        if msg.tool_calls:
            # Add assistant message with tool calls
            # Use model_dump to convert Pydantic model to dict for messages list
            self.messages.append(msg.model_dump(exclude_none=True))  # type: ignore
            agent_response.tool_calls = list(msg.tool_calls)

            # Execute each tool
            for tool_call in msg.tool_calls:
                tool_result = await self._execute_tool(tool_call)
                agent_response.tool_results.append(tool_result)

                # Add tool result to messages
                self.messages.append(
                    {
                        "role": "tool",
                        "tool_call_id": tool_call.id,
                        "content": json.dumps(tool_result.result),
                    }
                )

                # If request_review was called, we need to wait for human input
                if tool_call.function.name == "request_review":
                    agent_response.awaiting_input = True

                # Process middleware hooks
                context = ToolExecutionContext(
                    tool_name=tool_result.name,
                    tool_args=tool_result.args,
                    result=tool_result.result,
                    error=tool_result.error,
                    tool_call_id=tool_result.tool_call_id,
                )
                middleware_messages = self.middleware.process_tool_execution(context)
                for msg_to_add in middleware_messages:
                    self.messages.append(msg_to_add)  # type: ignore
        else:
            # Regular assistant message
            content = msg.content or ""
            self.messages.append({"role": "assistant", "content": content})
            agent_response.content = content

            # Check for completion phrases
            if self._is_done(content):
                agent_response.done = True

        self._prune_history()

        # Track agent response
        get_telemetry().track_agent_response(
            response_length=len(agent_response.content or ""),
            has_tool_calls=bool(agent_response.tool_calls),
            tool_call_count=len(agent_response.tool_calls),
            done=agent_response.done,
        )

        return agent_response

    def _is_done(self, content: str) -> bool:
        """Check if the agent indicates task completion."""
        lower = content.lower()
        done_phrases = [
            "task complete",
            "task completed",
            "i'm done",
            "i am done",
            "finished",
            "all done",
        ]
        return any(phrase in lower for phrase in done_phrases)

    async def run(self, user_input: str, max_turns: int = 50) -> list[AgentResponse]:
        """
        Run the agent loop until completion or max turns.

        Args:
            user_input: Initial user message
            max_turns: Maximum number of turns before stopping

        Returns:
            List of all agent responses
        """
        self.add_user_message(user_input)
        responses: list[AgentResponse] = []

        for _ in range(max_turns):
            response = await self.step()
            responses.append(response)

            if response.done or response.awaiting_input:
                break

            # If no tool calls and we have content, agent is waiting for input
            if not response.tool_calls and response.content:
                break

        return responses

    def reset(self, system_prompt: str | None = None) -> None:
        """Reset conversation history, optionally with new system prompt."""
        if system_prompt:
            self.messages = [{"role": "system", "content": system_prompt}]
        else:
            # Keep only system message
            self.messages = [m for m in self.messages if m.get("role") == "system"][:1]

        # Reset middleware state
        self.middleware.reset()

    @classmethod
    def from_registry(
        cls,
        provider: LLMProvider,
        system_prompt: str,
        registry: ToolRegistry,
        max_history_messages: int = 20,
        middleware: MiddlewareChain | None = None,
    ) -> AgentLoop:
        """
        Create an AgentLoop from a ToolRegistry.

        This is the preferred way to create an agent with tools.

        Args:
            provider: LLM provider instance (OpenAI, OpenRouter, local)
            system_prompt: System prompt for the agent
            registry: ToolRegistry containing available tools
            max_history_messages: Maximum number of messages to keep in history
            middleware: Optional middleware chain for hooking into tool execution

        Returns:
            Configured AgentLoop instance
        """
        return cls(
            provider=provider,
            system_prompt=system_prompt,
            tools=registry.get_definitions(),
            tool_handler=registry.execute,
            max_history_messages=max_history_messages,
            middleware=middleware,
        )
