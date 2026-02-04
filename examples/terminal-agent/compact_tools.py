"""
Compaction tools for the terminal agent.

Provides tools for manually triggering context compaction.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from tools import Tool, ToolExecutionResult

if TYPE_CHECKING:
    from agent import AgentLoop


class CompactTool(Tool):
    """
    Tool to compact the conversation history.

    Summarizes the current conversation and starts fresh with the summary
    as context. Useful when the context window is getting full.
    """

    def __init__(self, agent: AgentLoop) -> None:
        """
        Initialize the compact tool.

        Args:
            agent: The AgentLoop instance to compact
        """
        self._agent = agent

    @property
    def name(self) -> str:
        return "compact"

    @property
    def description(self) -> str:
        return (
            "Compact the conversation history by summarizing it. "
            "Use this when the conversation is getting long or you're running low on context. "
            "The summary preserves important information about what was accomplished, "
            "current work, files involved, and next steps."
        )

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {},
            "required": [],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        """Execute the compaction."""
        try:
            # Get current token usage before compaction
            old_tokens, max_tokens, old_usage = self._agent.get_token_usage()

            # Perform compaction
            summary = await self._agent.compact()

            # Get new token usage
            new_tokens, _, new_usage = self._agent.get_token_usage()

            return ToolExecutionResult(
                success=True,
                data={
                    "message": "Conversation compacted successfully",
                    "tokens_before": old_tokens,
                    "tokens_after": new_tokens,
                    "tokens_saved": old_tokens - new_tokens,
                    "usage_before": f"{old_usage:.1%}",
                    "usage_after": f"{new_usage:.1%}",
                    "context_limit": max_tokens,
                    "summary_preview": summary[:200] + "..." if len(summary) > 200 else summary,
                },
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=f"Failed to compact conversation: {e}",
            )


class ContextStatusTool(Tool):
    """
    Tool to check the current context window status.

    Reports token usage and whether compaction is recommended.
    """

    def __init__(self, agent: AgentLoop) -> None:
        """
        Initialize the context status tool.

        Args:
            agent: The AgentLoop instance to check
        """
        self._agent = agent

    @property
    def name(self) -> str:
        return "context_status"

    @property
    def description(self) -> str:
        return (
            "Check the current context window usage. "
            "Reports how many tokens are used, the limit, and whether compaction is recommended."
        )

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {},
            "required": [],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        """Get context status."""
        current, limit, usage = self._agent.get_token_usage()
        should_compact = self._agent.should_compact()

        return ToolExecutionResult(
            success=True,
            data={
                "current_tokens": current,
                "context_limit": limit,
                "usage_percent": f"{usage:.1%}",
                "messages_count": len(self._agent.messages),
                "should_compact": should_compact,
                "auto_compact_enabled": self._agent.auto_compact,
                "compact_threshold": f"{self._agent.compact_threshold:.0%}",
                "compaction_count": self._agent._compaction_count,
            },
        )
