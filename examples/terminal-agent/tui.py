"""
Terminal UI for the terminal agent using textual.

Provides a rich TUI with input handling, output formatting, and spinners.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any, cast

from textual import work
from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Container, ScrollableContainer, VerticalScroll, Horizontal
from textual.screen import ModalScreen
from textual.widgets import Footer, Header, Input, Label, Markdown, Static, Button
from textual.worker import Worker

if TYPE_CHECKING:
    from agent import AgentLoop, AgentResponse, ToolResult
    from tools import ToolRegistry


class Message(Static):
    """A single message in the conversation."""

    def __init__(self, content: str, role: str = "assistant") -> None:
        super().__init__()
        self.content = content
        self.role = role

    def compose(self) -> ComposeResult:
        if self.role == "user":
             yield Label(f"> {self.content}", classes=f"message-{self.role}")
        else:
             yield Markdown(self.content, classes=f"message-{self.role}")


class ToolCallDisplay(Static):
    """Display for a tool call result."""

    def __init__(self, result: ToolResult) -> None:
        super().__init__()
        self.result = result

    def compose(self) -> ComposeResult:
        icon = "✓" if not self.result.error else "✗"
        status_class = "ok" if not self.result.error else "err"

        yield Label(f"  {icon} {self.result.name}", classes=f"tool-{status_class}")

        # Display result details differently based on error status
        if self.result.error:
             yield Label(f"    Error: {self.result.result.get('error', 'Unknown error')}", classes="tool-details-err")
        else:
             # Just show a summary or the first few keys/lines to avoid clutter
             content = str(self.result.result)
             if len(content) > 100:
                 content = content[:100] + "..."
             yield Label(f"    -> {content}", classes="tool-details")


class ThinkingIndicator(Static):
    """Animated thinking indicator."""

    DEFAULT_CSS = """
    ThinkingIndicator {
        color: $text-muted;
    }
    """

    def __init__(self) -> None:
        super().__init__()
        self._dots = 0

    def on_mount(self) -> None:
        self.update_timer = self.set_interval(0.3, self._animate)

    def _animate(self) -> None:
        self._dots = (self._dots + 1) % 4
        dots = "." * self._dots
        self.update(f"Thinking{dots}")

    def stop(self) -> None:
        if hasattr(self, "update_timer"):
            self.update_timer.stop()


class ConversationView(VerticalScroll):
    """Scrollable conversation view."""

    DEFAULT_CSS = """
    ConversationView {
        height: 1fr;
        padding: 1;
        background: $surface;
    }

    .message-user {
        color: $success;
        padding-bottom: 1;
    }

    .message-assistant {
        color: $text;
        padding-bottom: 1;
    }

    .tool-ok {
        color: $accent;
    }

    .tool-err {
        color: $error;
    }

    .tool-details {
        color: $text-muted;
        padding-left: 2;
    }

    .tool-details-err {
        color: $error;
        padding-left: 2;
    }
    """

    def add_message(self, content: str, role: str = "assistant") -> None:
        """Add a message to the conversation."""
        msg = Message(content, role)
        self.mount(msg)
        self.scroll_end(animate=False)

    def add_tool_result(self, result: ToolResult) -> None:
        """Add a tool result indicator."""
        tool_display = ToolCallDisplay(result)
        self.mount(tool_display)
        self.scroll_end(animate=False)

    def show_thinking(self) -> ThinkingIndicator:
        """Show thinking indicator."""
        indicator = ThinkingIndicator()
        self.mount(indicator)
        self.scroll_end(animate=False)
        return indicator

    def clear_conversation(self) -> None:
        """Clear all messages."""
        for child in self.children:
            child.remove()


class StatusBar(Static):
    """Status bar showing provider and model info."""

    DEFAULT_CSS = """
    StatusBar {
        dock: top;
        height: 1;
        padding: 0 1;
        background: $primary;
        color: $text;
    }
    """

    def __init__(self, provider: str, model: str) -> None:
        super().__init__()
        self.provider = provider
        self.model = model

    def compose(self) -> ComposeResult:
        yield Label(f"Terminal Agent - {self.provider}: {self.model}")


class ReviewScreen(ModalScreen[bool]):
    """Screen for human review of agent actions."""

    DEFAULT_CSS = """
    ReviewScreen {
        align: center middle;
    }

    #review-container {
        width: 80%;
        height: 80%;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
        background: $surface;
    }

    #review-title {
        text-align: center;
        width: 100%;
        background: $primary;
        color: $text;
        text-style: bold;
        margin-bottom: 1;
    }

    .review-section-title {
        text-style: bold;
        color: $accent;
        margin-top: 1;
    }

    #review-buttons {
        dock: bottom;
        height: 3;
        align: center middle;
    }

    Button {
        margin: 0 2;
    }
    """

    def __init__(self, reason: str, session_summary: dict[str, Any], playbook_yaml: str | None = None) -> None:
        super().__init__()
        self.reason = reason
        self.summary = session_summary
        self.playbook_yaml = playbook_yaml

    def compose(self) -> ComposeResult:
        with VerticalScroll(id="review-container"):
            yield Label("HUMAN REVIEW REQUESTED", id="review-title")
            yield Label(f"Reason: {self.reason}", classes="review-section-title")

            yield Label("Session Summary:", classes="review-section-title")
            yield Markdown(self._format_summary())

            if self.playbook_yaml:
                yield Label("Current Playbook:", classes="review-section-title")
                yield Markdown(f"```yaml\n{self.playbook_yaml}\n```")

            with Horizontal(id="review-buttons"):
                yield Button("Approve", variant="success", id="approve")
                yield Button("Reject / Provide Feedback", variant="error", id="reject")

    def _format_summary(self) -> str:
        s = self.summary
        lines = [
            f"- Sandboxes created: {s.get('sandboxes_created', 0)}",
            f"- Commands executed: {s.get('commands_executed', 0)}",
            f"- Playbook tasks: {s.get('playbook_tasks', 0)}",
        ]
        if s.get("current_sandboxes"):
            lines.append("\nActive Sandboxes:")
            for sb in s["current_sandboxes"]:
                lines.append(f"  - {sb['name']} ({sb['id']}) at {sb.get('ip', 'N/A')}")
        return "\n".join(lines)

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "approve":
            self.dismiss(True)
        else:
            self.dismiss(False)


class CompletionScreen(ModalScreen[None]):
    """Screen shown when a task is completed."""

    DEFAULT_CSS = """
    CompletionScreen {
        align: center middle;
    }

    #completion-container {
        width: 60%;
        height: auto;
        border: thick $success;
        background: $surface;
        padding: 1 2;
    }

    #completion-title {
        text-align: center;
        width: 100%;
        background: $success;
        color: $text;
        text-style: bold;
        margin-bottom: 1;
    }

    .completion-section-title {
        text-style: bold;
        color: $accent;
        margin-top: 1;
    }

    #completion-footer {
        margin-top: 2;
        align: center middle;
    }
    """

    def __init__(self, summary: str, session_stats: dict[str, Any]) -> None:
        super().__init__()
        self.summary = summary
        self.stats = session_stats

    def compose(self) -> ComposeResult:
        with VerticalScroll(id="completion-container"):
            yield Label("TASK COMPLETE", id="completion-title")

            yield Label("Summary:", classes="completion-section-title")
            yield Markdown(self.summary)

            yield Label("Session Stats:", classes="completion-section-title")
            yield Markdown(self._format_stats())

            with Horizontal(id="completion-footer"):
                yield Button("Close", variant="primary", id="close")

    def _format_stats(self) -> str:
        s = self.stats
        return f"- Duration: {s.get('session_duration_seconds', 0):.1f}s\n- Sandboxes: {s.get('sandboxes_created', 0)}\n- Commands: {s.get('commands_executed', 0)}\n- Playbook Tasks: {s.get('playbook_tasks', 0)}"

    def on_button_pressed(self, event: Button.Pressed) -> None:
        self.dismiss()


class TerminalAgentApp(App):
    """Main TUI application for the terminal agent."""

    CSS = """
    Screen {
        layout: vertical;
    }

    #input-container {
        dock: bottom;
        height: auto;
        padding: 1;
        background: $surface;
    }

    #user-input {
        width: 100%;
    }

    #user-input:focus {
        border: tall $accent;
    }
    """

    BINDINGS = [
        Binding("ctrl+c", "quit", "Quit"),
        Binding("ctrl+r", "reset", "Reset"),
        Binding("escape", "blur_input", "Unfocus", show=False),
    ]

    def __init__(
        self,
        agent: AgentLoop,
        provider_type: str = "unknown",
        model: str = "unknown",
    ) -> None:
        super().__init__()
        self.agent = agent
        self.provider_type = provider_type
        self.model = model
        self._thinking_indicator: ThinkingIndicator | None = None

    def compose(self) -> ComposeResult:
        yield StatusBar(self.provider_type, self.model)
        yield ConversationView(id="conversation")
        with Container(id="input-container"):
            yield Input(placeholder="Type your message...", id="user-input")
        yield Footer()

    def on_mount(self) -> None:
        """Focus input on startup."""
        self.query_one("#user-input", Input).focus()
        conv = self.query_one("#conversation", ConversationView)
        conv.add_message("Type a message to begin. Use Ctrl+R to reset, Ctrl+C to quit.")

    async def on_input_submitted(self, event: Input.Submitted) -> None:
        """Handle user input submission."""
        user_input = event.value.strip()
        if not user_input:
            return

        # Clear input
        input_widget = self.query_one("#user-input", Input)
        input_widget.value = ""

        conv = self.query_one("#conversation", ConversationView)

        # Add user message
        conv.add_message(user_input, role="user")

        # Show thinking indicator
        self._thinking_indicator = conv.show_thinking()

        # Run agent in background worker
        self.run_agent(user_input)

    @work(exclusive=True)
    async def run_agent(self, user_input: str) -> list[AgentResponse]:
        """Run the agent as an async worker."""
        return await self.agent.run(user_input)

    def on_worker_state_changed(self, event: Worker.StateChanged) -> None:
        """Handle worker state changes."""
        if event.state.name == "SUCCESS":
            # Remove thinking indicator
            if self._thinking_indicator:
                self._thinking_indicator.stop()
                self._thinking_indicator.remove()
                self._thinking_indicator = None

            # Process responses
            responses: list[AgentResponse] = event.worker.result
            conv = self.query_one("#conversation", ConversationView)

            last_response = None
            task_complete_result = None

            for response in responses:
                last_response = response
                # Show tool results
                for tool_result in response.tool_results:
                    conv.add_tool_result(tool_result)
                    if tool_result.name == "task_complete" and not tool_result.error:
                        task_complete_result = tool_result

                # Show content
                if response.content:
                    conv.add_message(response.content)

            # Check for task completion
            if task_complete_result:
                self.handle_task_completion(task_complete_result)
            # Check if we need human review
            elif last_response and last_response.awaiting_input:
                self.handle_review_request(last_response)
            else:
                # Refocus input
                self.query_one("#user-input", Input).focus()

    def handle_task_completion(self, result: ToolResult) -> None:
        """Handle task completion."""
        summary = result.result.get("summary", "No summary provided")
        stats = result.result.get("session_stats", {})

        self.push_screen(CompletionScreen(summary, stats), lambda _: self.query_one("#user-input", Input).focus())

    def handle_review_request(self, response: AgentResponse) -> None:
        """Handle a request for human review."""
        # Find the request_review tool result
        review_result = None
        for tr in response.tool_results:
            if tr.name == "request_review":
                review_result = tr
                break

        if not review_result:
            self.query_one("#user-input", Input).focus()
            return

        reason = review_result.result.get("reason", "No reason provided")
        summary = review_result.result.get("summary", {})

        # Try to get playbook from agent's tool handler (PlaybookManager)
        playbook_yaml = None
        if hasattr(self.agent, "tool_handler") and hasattr(self.agent.tool_handler, "__self__"):
            # We cast to ToolRegistry because we know AgentLoop.from_registry sets it this way
            registry = cast("ToolRegistry", self.agent.tool_handler.__self__)  # type: ignore
            # This is a bit hacky, but we need the playbook manager
            # In main.py, it's passed to tools
            for tool in registry._tools.values():
                if hasattr(tool, "manager") and hasattr(tool.manager, "to_yaml"):
                    playbook_yaml = tool.manager.to_yaml()
                    break

        def check_review_result(approved: bool) -> None:
            if approved:
                conv = self.query_one("#conversation", ConversationView)
                conv.add_message("Human approved the review.", role="assistant")
                self._thinking_indicator = conv.show_thinking()
                self.run_agent_continuation("Review approved by human. You may proceed.")
            else:
                # For rejection, we'll focus the input and let the user type feedback
                conv = self.query_one("#conversation", ConversationView)
                conv.add_message("Review rejected. Please provide feedback to the agent.", role="assistant")
                self.query_one("#user-input", Input).focus()

        self.push_screen(ReviewScreen(reason, summary, playbook_yaml), check_review_result)

    @work(exclusive=True)
    async def run_agent_continuation(self, user_input: str) -> list[AgentResponse]:
        """Continue the agent loop after review."""
        return await self.agent.run(user_input)

    def action_reset(self) -> None:
        """Reset the conversation."""
        self.agent.reset()
        conv = self.query_one("#conversation", ConversationView)
        conv.clear_conversation()
        conv.add_message("Conversation reset.")
        self.query_one("#user-input", Input).focus()

    def action_blur_input(self) -> None:
        """Unfocus the input."""
        self.query_one("#user-input", Input).blur()


async def run_tui(
    agent: AgentLoop,
    provider_type: str = "unknown",
    model: str = "unknown",
) -> None:
    """
    Run the terminal agent TUI.

    Args:
        agent: Configured AgentLoop instance
        provider_type: Provider type for display
        model: Model name for display
    """
    app = TerminalAgentApp(agent, provider_type, model)
    await app.run_async()
