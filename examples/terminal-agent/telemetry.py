"""
Telemetry module for anonymous usage tracking via PostHog.

Tracks agent actions, tool calls, prompts, and playbook events.
Respects ANONYMIZED_TELEMETRY env var to enable/disable tracking.
"""

from __future__ import annotations

import hashlib
import os
import platform
import uuid
from typing import Any, Optional
from posthog import Posthog

POSTHOG_API_KEY = "phc_GYlAA4sZbgoDEjkhaziuNwP7qiKaEOmVM7khlwMW5xP"
POSTHOG_HOST = "https://us.i.posthog.com"
posthog = Posthog(
    project_api_key=POSTHOG_API_KEY,
    host=POSTHOG_HOST,
    disable_geoip=True
)



def _generate_anonymous_id() -> str:
    """
    Generate a stable anonymous ID based on machine characteristics.

    Uses a hash of machine-specific info to create a consistent ID
    across sessions without storing any PII.
    """
    # Combine machine-specific but non-identifying info
    machine_info = f"{platform.node()}-{platform.machine()}-{platform.system()}"

    # Hash to create anonymous ID
    return hashlib.sha256(machine_info.encode()).hexdigest()[:32]


class Telemetry:
    """
    Anonymous telemetry tracker using PostHog.

    All events are anonymous - no PII is collected.
    Controlled by ANONYMIZED_TELEMETRY env var (default: disabled).
    """

    _instance: Optional[Telemetry] = None

    def __init__(self) -> None:
        self._enabled = self._check_enabled()
        self._distinct_id = _generate_anonymous_id()
        self._session_id = str(uuid.uuid4())

    @classmethod
    def get_instance(cls) -> Telemetry:
        """Get or create the singleton telemetry instance."""
        if cls._instance is None:
            cls._instance = cls()
        return cls._instance

    def _check_enabled(self) -> bool:
        """Check if telemetry is enabled via environment variable."""
        env_value = os.getenv("ANONYMIZED_TELEMETRY", "false").lower()
        return env_value in ("true", "1", "yes", "on")

    @property
    def enabled(self) -> bool:
        """Whether telemetry is enabled."""
        return self._enabled

    @property
    def session_id(self) -> str:
        """Current session ID."""
        return self._session_id

    def _capture(self, event: str, properties: dict[str, Any] | None = None) -> None:
        """
        Internal method to capture an event.

        Args:
            event: Event name
            properties: Event properties
        """
        if not self.enabled:
            return

        props = properties or {}
        props["session_id"] = self._session_id
        props["$process_person_profile"] = False  # Anonymous events only

        try:
            posthog.capture(
                distinct_id=self._distinct_id,
                event=event,
                properties=props,
            )
        except Exception:
            # Silently fail - telemetry should never break the app
            pass

    # Session events

    def track_session_start(
        self,
        provider_type: str,
        model: str,
    ) -> None:
        """Track session start."""
        self._capture("session_start", {
            "provider_type": provider_type,
            "model": model,
            "platform": platform.system(),
            "python_version": platform.python_version(),
        })

    def track_session_end(
        self,
        duration_seconds: float,
        commands_executed: int,
        sandboxes_created: int,
    ) -> None:
        """Track session end."""
        self._capture("session_end", {
            "duration_seconds": duration_seconds,
            "commands_executed": commands_executed,
            "sandboxes_created": sandboxes_created,
        })

    # Prompt events

    def track_user_prompt(
        self,
        prompt_length: int,
        message_count: int,
    ) -> None:
        """
        Track when a user sends a prompt.

        Args:
            prompt_length: Length of the prompt in characters
            message_count: Current message count in conversation
        """
        self._capture("user_prompt", {
            "prompt_length": prompt_length,
            "message_count": message_count,
        })

    def track_agent_response(
        self,
        response_length: int,
        has_tool_calls: bool,
        tool_call_count: int,
        done: bool,
    ) -> None:
        """
        Track agent response.

        Args:
            response_length: Length of response content
            has_tool_calls: Whether response includes tool calls
            tool_call_count: Number of tool calls
            done: Whether agent indicates task completion
        """
        self._capture("agent_response", {
            "response_length": response_length,
            "has_tool_calls": has_tool_calls,
            "tool_call_count": tool_call_count,
            "done": done,
        })

    # Tool events

    def track_tool_call(
        self,
        tool_name: str,
        args_keys: list[str],
    ) -> None:
        """
        Track when a tool is called.

        Args:
            tool_name: Name of the tool
            args_keys: List of argument keys (not values for privacy)
        """
        self._capture("tool_call", {
            "tool_name": tool_name,
            "args_keys": args_keys,
        })

    def track_tool_result(
        self,
        tool_name: str,
        success: bool,
        has_error: bool,
    ) -> None:
        """
        Track tool execution result.

        Args:
            tool_name: Name of the tool
            success: Whether execution succeeded
            has_error: Whether result contains error
        """
        self._capture("tool_result", {
            "tool_name": tool_name,
            "success": success,
            "has_error": has_error,
        })

    # Playbook events

    def track_playbook_init(
        self,
        playbook_name: str,
        hosts: str,
    ) -> None:
        """Track playbook initialization."""
        self._capture("playbook_init", {
            "playbook_name": playbook_name,
            "hosts": hosts,
        })

    def track_playbook_task_added(
        self,
        task_name: str,
        module: str,
        task_count: int,
    ) -> None:
        """Track task added to playbook."""
        self._capture("playbook_task_added", {
            "task_name": task_name,
            "module": module,
            "task_count": task_count,
        })

    def track_playbook_validated(
        self,
        is_valid: bool,
        error_count: int,
    ) -> None:
        """Track playbook validation."""
        self._capture("playbook_validated", {
            "is_valid": is_valid,
            "error_count": error_count,
        })

    def track_playbook_run(
        self,
        check_mode: bool,
        success: bool,
    ) -> None:
        """Track playbook execution."""
        self._capture("playbook_run", {
            "check_mode": check_mode,
            "success": success,
        })

    # Sandbox events

    def track_sandbox_created(
        self,
        source_vm_name: str,
    ) -> None:
        """Track sandbox creation."""
        self._capture("sandbox_created", {
            "source_vm_name": source_vm_name,
        })

    def track_command_executed(
        self,
        success: bool,
    ) -> None:
        """Track command execution in sandbox."""
        self._capture("command_executed", {
            "success": success,
        })

    # Plan events

    def track_plan_created(
        self,
        step_count: int,
    ) -> None:
        """Track plan creation."""
        self._capture("plan_created", {
            "step_count": step_count,
        })

    # Review events

    def track_review_requested(
        self,
        reason_length: int,
    ) -> None:
        """Track review request."""
        self._capture("review_requested", {
            "reason_length": reason_length,
        })

    def track_task_completed(self) -> None:
        """Track task completion."""
        self._capture("task_completed", {})

    def flush(self) -> None:
        """Flush any pending events."""
        if self.enabled:
            try:
                posthog.flush()
            except Exception:
                pass

    def shutdown(self) -> None:
        """Shutdown telemetry and flush remaining events."""
        if self.enabled:
            try:
                posthog.shutdown()
            except Exception:
                pass


# Convenience functions for global access

def get_telemetry() -> Telemetry:
    """Get the global telemetry instance."""
    return Telemetry.get_instance()


def is_telemetry_enabled() -> bool:
    """Check if telemetry is enabled."""
    return get_telemetry().enabled
