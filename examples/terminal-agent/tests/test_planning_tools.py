
from planning_tools import PlanModeTool

def test_plan_mode_tool():
    tool = PlanModeTool()
    assert tool.name == "set_plan"
    assert "steps" in tool.parameters["properties"]
    
    # Test valid execution
    result = tool.execute(steps=["Step 1", "Step 2"])
    assert result.success
    assert result.data["plan"] == ["Step 1", "Step 2"]
    
    # Test invalid execution
    result = tool.execute(steps="Invalid")
    assert not result.success
    assert "Steps must be a list" in result.error_message
