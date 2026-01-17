import subprocess
import tempfile
import os
from dataclasses import dataclass, field
from typing import Any, Optional
import yaml

from tools import Tool, ToolExecutionResult
from telemetry import get_telemetry


class AnsibleDumper(yaml.SafeDumper):
    def increase_indent(self, flow=False, indentless=False):
        return super(AnsibleDumper, self).increase_indent(flow, False)


class PlaybookSerializer:
    """Handles YAML serialization of playbooks."""

    def to_yaml(self, playbook: list[dict[str, Any]]) -> str:
        """Convert playbook to YAML string."""
        return yaml.dump(playbook, Dumper=AnsibleDumper, sort_keys=False, indent=2)

    def write_temp_file(self, playbook: list[dict[str, Any]]) -> str:
        """Write playbook to a temporary file and return the path."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yml', delete=False) as tmp:
            tmp.write(self.to_yaml(playbook))
            return tmp.name


class PlaybookValidator:
    """Handles playbook validation: structural checks, ansible-lint, syntax-check."""

    def __init__(self, serializer: PlaybookSerializer) -> None:
        self._serializer = serializer

    def validate(self, playbook: list[dict[str, Any]]) -> tuple[bool, list[str]]:
        """
        Perform validation of the playbook.

        Returns:
            Tuple of (is_valid, list of error messages)
        """
        errors = []
        if not playbook:
            errors.append("Playbook is empty or not initialized.")
            return False, errors

        # Structural validation
        structural_errors = self._validate_structure(playbook)
        if structural_errors:
            return False, structural_errors

        # External tool validation
        tool_errors = self._validate_with_tools(playbook)
        errors.extend(tool_errors)

        return len(errors) == 0, errors

    def _validate_structure(self, playbook: list[dict[str, Any]]) -> list[str]:
        """Check basic playbook structure."""
        errors = []
        for i, play in enumerate(playbook):
            if not isinstance(play, dict):
                errors.append(f"Play at index {i} is not a dictionary.")
                continue
            if "name" not in play:
                errors.append(f"Play at index {i} is missing 'name'.")
            if "hosts" not in play:
                errors.append(f"Play at index {i} is missing 'hosts'.")
            if "tasks" not in play:
                errors.append(f"Play at index {i} is missing 'tasks'.")
        return errors

    def _validate_with_tools(self, playbook: list[dict[str, Any]]) -> list[str]:
        """Run ansible-playbook --syntax-check and ansible-lint."""
        errors = []
        tmp_path = self._serializer.write_temp_file(playbook)

        try:
            # Syntax check
            syntax_check = subprocess.run(
                ["ansible-playbook", "--syntax-check", tmp_path],
                capture_output=True,
                text=True
            )
            if syntax_check.returncode != 0:
                errors.append("Ansible syntax check failed:")
                errors.append(syntax_check.stderr or syntax_check.stdout)

            # Lint check
            lint_check = subprocess.run(
                ["ansible-lint", "--offline", "--nocolor", tmp_path],
                capture_output=True,
                text=True
            )
            if lint_check.returncode != 0:
                errors.append("Ansible-lint found issues:")
                errors.append(lint_check.stdout or lint_check.stderr)

        except FileNotFoundError as e:
            errors.append(f"Required tool not found: {str(e)}")
        except Exception as e:
            errors.append(f"Validation error: {str(e)}")
        finally:
            if os.path.exists(tmp_path):
                os.remove(tmp_path)

        return errors


class PlaybookExecutor:
    """Handles playbook execution via ansible-playbook."""

    def __init__(self, serializer: PlaybookSerializer) -> None:
        self._serializer = serializer

    def run(
        self,
        playbook: list[dict[str, Any]],
        check_mode: bool = False,
        target_host: Optional[str] = None
    ) -> tuple[bool, str, str]:
        """
        Run the playbook using ansible-playbook.

        Args:
            playbook: The playbook data to run.
            check_mode: If True, run with --check (dry run).
            target_host: Optional host to target.

        Returns:
            Tuple of (success, stdout, stderr)
        """
        if not playbook:
            return False, "", "Playbook is empty or not initialized."

        tmp_path = self._serializer.write_temp_file(playbook)

        try:
            cmd = ["ansible-playbook", tmp_path]
            if check_mode:
                cmd.append("--check")

            if target_host:
                cmd.extend(["-i", f"{target_host},"])

            env = os.environ.copy()
            env["ANSIBLE_HOST_KEY_CHECKING"] = "False"

            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                env=env
            )

            return result.returncode == 0, result.stdout, result.stderr

        except Exception as e:
            return False, "", str(e)
        finally:
            if os.path.exists(tmp_path):
                os.remove(tmp_path)


@dataclass
class PlaybookManager:
    """Manages playbook state and coordinates serialization, validation, execution."""

    playbook: list[dict[str, Any]] = field(default_factory=list)
    _serializer: PlaybookSerializer = field(default_factory=PlaybookSerializer)
    _validator: PlaybookValidator = field(init=False)
    _executor: PlaybookExecutor = field(init=False)

    def __post_init__(self) -> None:
        self._validator = PlaybookValidator(self._serializer)
        self._executor = PlaybookExecutor(self._serializer)

    def init_playbook(self, name: str, hosts: str = "all") -> None:
        """Initialize a new playbook."""
        self.playbook = [{
            "name": name,
            "hosts": hosts,
            "tasks": []
        }]

    def get_playbook(self) -> list[dict[str, Any]]:
        """Get the current playbook."""
        return self.playbook

    def to_yaml(self) -> str:
        """Convert playbook to YAML string."""
        return self._serializer.to_yaml(self.playbook)

    def add_task(self, name: str, module: str, args: dict[str, Any]) -> None:
        """Add a task to the current playbook."""
        if not self.playbook:
            raise ValueError("No playbook initialized. Call init_playbook first.")

        task = {
            "name": name,
            module: args
        }
        self.playbook[0]["tasks"].append(task)

    def validate_playbook(self) -> tuple[bool, list[str]]:
        """Validate the playbook."""
        return self._validator.validate(self.playbook)

    def run_playbook(
        self,
        check_mode: bool = False,
        target_host: Optional[str] = None
    ) -> tuple[bool, str, str]:
        """Run the playbook."""
        return self._executor.run(self.playbook, check_mode, target_host)


class InitPlaybookTool(Tool):
    """Tool to initialize a new Ansible playbook."""
    
    def __init__(self, manager: PlaybookManager) -> None:
        self.manager = manager

    @property
    def name(self) -> str:
        return "init_playbook"

    @property
    def description(self) -> str:
        return "Initialize a new Ansible playbook for the current session."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "name": {
                    "type": "string",
                    "description": "Name of the playbook/play",
                },
                "hosts": {
                    "type": "string",
                    "description": "Hosts to target (default: all)",
                    "default": "all"
                },
            },
            "required": ["name"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            name = kwargs["name"]
            hosts = kwargs.get("hosts", "all")

            self.manager.init_playbook(name, hosts)

            # Track playbook init
            get_telemetry().track_playbook_init(playbook_name=name, hosts=hosts)

            return ToolExecutionResult(
                success=True,
                data={"playbook": self.manager.get_playbook()},
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )


class AddTaskTool(Tool):
    """Tool to add a task to the Ansible playbook."""

    def __init__(self, manager: PlaybookManager) -> None:
        self.manager = manager

    @property
    def name(self) -> str:
        return "add_task"

    @property
    def description(self) -> str:
        return "Add a task to the current Ansible playbook."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "name": {
                    "type": "string",
                    "description": "Description of the task",
                },
                "module": {
                    "type": "string",
                    "description": "Ansible module to use (e.g., shell, apt, copy)",
                },
                "args": {
                    "type": "object",
                    "description": "Arguments for the module",
                },
            },
            "required": ["name", "module", "args"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            name = kwargs["name"]
            module = kwargs["module"]
            args = kwargs["args"]

            self.manager.add_task(name, module, args)

            # Track task added
            task_count = len(self.manager.playbook[0]["tasks"]) if self.manager.playbook else 0
            get_telemetry().track_playbook_task_added(
                task_name=name,
                module=module,
                task_count=task_count,
            )

            return ToolExecutionResult(
                success=True,
                data={"playbook": self.manager.get_playbook()},
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )


class DryRunPlaybookTool(Tool):
    """Tool to perform a dry-run of the Ansible playbook."""

    def __init__(self, manager: PlaybookManager) -> None:
        self.manager = manager

    @property
    def name(self) -> str:
        return "dry_run_playbook"

    @property
    def description(self) -> str:
        return "Perform a dry-run of the current Ansible playbook on a target host (ansible-playbook --check)."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "target_host": {
                    "type": "string",
                    "description": "Host to target for the dry-run (e.g., IP address or hostname).",
                },
            },
            "required": ["target_host"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            target_host = kwargs["target_host"]
            success, stdout, stderr = self.manager.run_playbook(check_mode=True, target_host=target_host)

            # Track dry run
            get_telemetry().track_playbook_run(check_mode=True, success=success)

            return ToolExecutionResult(
                success=success,
                data={
                    "stdout": stdout,
                    "stderr": stderr,
                    "check_mode": True
                },
                error_message=stderr if not success else None
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )


class RunPlaybookTool(Tool):
    """Tool to execute the Ansible playbook."""

    def __init__(self, manager: PlaybookManager) -> None:
        self.manager = manager

    @property
    def name(self) -> str:
        return "run_playbook"

    @property
    def description(self) -> str:
        return "Execute the current Ansible playbook on a target host. This is a critical action that modifies the target system."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "target_host": {
                    "type": "string",
                    "description": "Host to target for the execution (e.g., IP address or hostname).",
                },
                "confirm": {
                    "type": "boolean",
                    "description": "Must be set to true to confirm execution of this critical action.",
                },
            },
            "required": ["target_host", "confirm"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            target_host = kwargs["target_host"]
            confirm = kwargs.get("confirm", False)

            if not confirm:
                return ToolExecutionResult(
                    success=False,
                    data={},
                    error_message="Execution not confirmed. Set 'confirm' to true to run the playbook.",
                )

            success, stdout, stderr = self.manager.run_playbook(check_mode=False, target_host=target_host)

            # Track playbook run
            get_telemetry().track_playbook_run(check_mode=False, success=success)

            return ToolExecutionResult(
                success=success,
                data={
                    "stdout": stdout,
                    "stderr": stderr,
                    "check_mode": False
                },
                error_message=stderr if not success else None
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )


class ViewPlaybookTool(Tool):
    """Tool to view the current Ansible playbook."""

    def __init__(self, manager: PlaybookManager) -> None:
        self.manager = manager

    @property
    def name(self) -> str:
        return "view_playbook"

    @property
    def description(self) -> str:
        return "View the current Ansible playbook in YAML format."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {},
            "required": [],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            return ToolExecutionResult(
                success=True,
                data={"playbook_yaml": self.manager.to_yaml()},
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )


class ValidatePlaybookTool(Tool):
    """Tool to validate the current Ansible playbook."""

    def __init__(self, manager: PlaybookManager) -> None:
        self.manager = manager

    @property
    def name(self) -> str:
        return "validate_playbook"

    @property
    def description(self) -> str:
        return "Validate the current Ansible playbook for structural correctness."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {},
            "required": [],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        try:
            is_valid, errors = self.manager.validate_playbook()

            # Track validation
            get_telemetry().track_playbook_validated(
                is_valid=is_valid,
                error_count=len(errors),
            )

            return ToolExecutionResult(
                success=is_valid,
                data={
                    "is_valid": is_valid,
                    "errors": errors,
                    "playbook_yaml": self.manager.to_yaml() if is_valid else None
                },
                error_message="\n".join(errors) if not is_valid else None
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                data={},
                error_message=str(e),
            )
