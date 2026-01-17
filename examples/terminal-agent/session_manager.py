"""
Session management for the terminal agent.

Tracks created sandboxes, commands run, and playbook state.
"""

from dataclasses import dataclass, field
from typing import Any, Optional
from datetime import datetime

from ansible_tools import PlaybookManager
from telemetry import get_telemetry


@dataclass
class CommandEntry:
    """Record of a command execution."""
    sandbox_id: str
    command: str
    exit_code: int
    stdout: str
    stderr: str
    timestamp: datetime = field(default_factory=datetime.now)


@dataclass
class SandboxEntry:
    """Record of a created sandbox."""
    sandbox_id: str
    vm_name: str
    source_vm: str
    ip_address: Optional[str] = None
    timestamp: datetime = field(default_factory=datetime.now)


class SessionState:
    """
    Maintains the state of the current agent session.
    """

    def __init__(self, playbook_manager: Optional[PlaybookManager] = None) -> None:
        self.sandboxes: list[SandboxEntry] = []
        self.commands: list[CommandEntry] = []
        self.playbook_manager = playbook_manager or PlaybookManager()
        self.start_time = datetime.now()

    def add_sandbox(self, sandbox_data: dict[str, Any]) -> None:
        """Add a sandbox to the session state."""
        # sandbox_data is usually from the SDK response
        entry = SandboxEntry(
            sandbox_id=sandbox_data.get("id") or sandbox_data.get("sandbox_id", "unknown"),
            vm_name=sandbox_data.get("vm_name", "unknown"),
            source_vm=sandbox_data.get("source_vm_name", "unknown"),
            ip_address=sandbox_data.get("ip_address"),
        )
        self.sandboxes.append(entry)

    def add_command(self, sandbox_id: str, command: str, result_data: dict[str, Any]) -> None:
        """Add a command execution to the session state."""
        entry = CommandEntry(
            sandbox_id=sandbox_id,
            command=command,
            exit_code=result_data.get("exit_code", 0),
            stdout=result_data.get("stdout", ""),
            stderr=result_data.get("stderr", ""),
        )
        self.commands.append(entry)

    def get_summary(self) -> dict[str, Any]:
        """Get a summary of the current session."""
        return {
            "session_duration_seconds": (datetime.now() - self.start_time).total_seconds(),
            "sandboxes_created": len(self.sandboxes),
            "commands_executed": len(self.commands),
            "playbook_tasks": len(self.playbook_manager.playbook[0]["tasks"]) if self.playbook_manager.playbook else 0,
            "current_sandboxes": [
                {"id": s.sandbox_id, "name": s.vm_name, "ip": s.ip_address}
                for s in self.sandboxes
            ]
        }


from tools import Tool, ToolExecutionResult

class ViewSessionTool(Tool):
    """Tool to view the current session state."""

    def __init__(self, session_state: SessionState) -> None:
        self.session_state = session_state

    @property
    def name(self) -> str:
        return "view_session"

    @property
    def description(self) -> str:
        return "View a summary of the current session, including created sandboxes and executed commands."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {},
            "required": [],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            summary = self.session_state.get_summary()
            return ToolExecutionResult(
                success=True,
                data=summary,
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )


class RequestReviewTool(Tool):
    """Tool to request a human review of the session."""

    def __init__(self, session_state: SessionState) -> None:
        self.session_state = session_state

    @property
    def name(self) -> str:
        return "request_review"

    @property
    def description(self) -> str:
        return "Request a human review of the current session state and Ansible playbook before proceeding to critical actions."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "reason": {
                    "type": "string",
                    "description": "Reason for requesting review",
                },
            },
            "required": ["reason"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            reason = kwargs.get("reason", "No reason provided")
            summary = self.session_state.get_summary()

            # Track review request
            get_telemetry().track_review_requested(reason_length=len(reason))

            return ToolExecutionResult(
                success=True,
                data={
                    "status": "awaiting_review",
                    "reason": reason,
                    "summary": summary,
                },
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )


class TaskCompletionTool(Tool):
    """Tool to signal that a task has been completed."""

    def __init__(self, session_state: SessionState) -> None:
        self.session_state = session_state

    @property
    def name(self) -> str:
        return "task_complete"

    @property
    def description(self) -> str:
        return "Signal that the current task has been successfully completed. Provide a final summary of actions taken."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "summary": {
                    "type": "string",
                    "description": "A final summary of the actions taken and the result of the task.",
                },
            },
            "required": ["summary"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            summary = kwargs["summary"]
            session_summary = self.session_state.get_summary()

            # Track task completion
            get_telemetry().track_task_completed()

            return ToolExecutionResult(
                success=True,
                data={
                    "status": "task_complete",
                    "summary": summary,
                    "session_stats": session_summary,
                },
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )
