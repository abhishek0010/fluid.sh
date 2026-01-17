"""
Sandbox management tools for the terminal agent.
"""

from typing import Any
from uuid import uuid4

try:
    from virsh_sandbox import VirshSandbox
except ImportError:
    VirshSandbox = Any

from tools import Tool, ToolExecutionResult
from session_manager import SessionState
from telemetry import get_telemetry


class CreateSandboxTool(Tool):
    """Tool to create a new sandbox from an existing VM."""

    def __init__(self, client: VirshSandbox, session_state: SessionState) -> None:
        self.client = client
        self.session_state = session_state

    @property
    def name(self) -> str:
        return "create_sandbox"

    @property
    def description(self) -> str:
        return "Create a new sandbox by cloning from an existing VM"

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "source_vm_name": {
                    "type": "string",
                    "description": "Name of existing VM to clone from",
                },
                "agent_id": {
                    "type": "string",
                    "description": "Identifier for the requesting agent (optional)",
                },
                "vm_name": {
                    "type": "string",
                    "description": "Optional name for the new sandbox VM",
                },
            },
            "required": ["source_vm_name"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            source_vm_name = kwargs["source_vm_name"]
            agent_id = kwargs.get("agent_id", str(uuid4()))
            vm_name = kwargs.get("vm_name")

            # Call SDK
            response = self.client.sandbox.create_sandbox(
                source_vm_name=source_vm_name,
                agent_id=agent_id,
                vm_name=vm_name,
                auto_start=True,
                wait_for_ip=True,
                request_timeout=180.0,
            )

            # Extract sandbox data
            sandbox = response.sandbox

            # Convert to dict
            if hasattr(sandbox, "to_dict"):
                data = sandbox.to_dict()
            else:
                try:
                    data = sandbox.model_dump()
                except AttributeError:
                    try:
                        data = sandbox.__dict__
                    except AttributeError:
                        data = {"id": str(sandbox)}

            # Track in session state
            self.session_state.add_sandbox(data)

            # Track sandbox creation
            get_telemetry().track_sandbox_created(source_vm_name=source_vm_name)

            return ToolExecutionResult(success=True, data=data)

        except Exception as e:
            return ToolExecutionResult(success=False, data={}, error_message=str(e))


class RunCommandTool(Tool):
    """Tool to run a command in a sandbox."""

    def __init__(self, client: VirshSandbox, session_state: SessionState) -> None:
        self.client = client
        self.session_state = session_state

    @property
    def name(self) -> str:
        return "run_command"

    @property
    def description(self) -> str:
        return "Run a single command in a specified sandbox"

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "sandbox_id": {
                    "type": "string",
                    "description": "ID of the sandbox to run the command in",
                },
                "command": {
                    "type": "string",
                    "description": "The command to run",
                },
            },
            "required": ["sandbox_id", "command"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        # Disallow chained commands
        command = kwargs.get("command", "")
        if any(op in command for op in ["&&", "||", ";", "|", "`"]):
            get_telemetry().track_command_executed(success=False)
            return ToolExecutionResult(
                success=False,
                data={},
                error_message="Chained commands are not allowed",
            )

        try:
            sandbox_id = kwargs["sandbox_id"]

            # Call SDK
            response = self.client.sandbox.run_command(
                sandbox_id=sandbox_id,
                cmd=command,
                request_timeout=30.0,
            )

            result = response.result

            if hasattr(result, "to_dict"):
                data = result.to_dict()
            else:
                try:
                    data = result.model_dump()
                except AttributeError:
                    data = {"output": str(result)}

            # Track in session state if successful
            self.session_state.add_command(sandbox_id, command, data)

            # Track command execution
            get_telemetry().track_command_executed(success=True)

            return ToolExecutionResult(success=True, data=data)

        except Exception as e:
            get_telemetry().track_command_executed(success=False)
            return ToolExecutionResult(success=False, data={}, error_message=str(e))
