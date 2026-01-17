"""
Planning tools for the terminal agent.
"""

from typing import Any
from tools import Tool, ToolExecutionResult
from telemetry import get_telemetry

class PlanModeTool(Tool):
    """Tool to create and display execution plans."""

    @property
    def name(self) -> str:
        return "set_plan"

    @property
    def description(self) -> str:
        return "Create or update the execution plan with a list of steps."

    @property
    def parameters(self) -> dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "steps": {
                    "type": "array",
                    "items": {"type": "string"},
                    "description": "List of steps in the plan",
                },
            },
            "required": ["steps"],
        }

    async def execute(self, **kwargs: Any) -> ToolExecutionResult:
        steps = kwargs.get("steps", [])
        if not isinstance(steps, list) or not all(isinstance(s, str) for s in steps):
            return ToolExecutionResult(
                success=False,
                data={},
                error_message="Steps must be a list of strings",
            )

        # Track plan creation
        get_telemetry().track_plan_created(step_count=len(steps))

        return ToolExecutionResult(
            success=True,
            data={"plan": steps},
        )
