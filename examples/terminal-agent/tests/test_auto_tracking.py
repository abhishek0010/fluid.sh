
import pytest
import json
from unittest.mock import MagicMock
from agent import AgentLoop, ToolResult
from llm import LLMProvider

class MockProvider(LLMProvider):
    def __init__(self):
        self.mock = MagicMock()
    
    @property
    def model(self) -> str:
        return "mock-model"
    
    @property
    def name(self) -> str:
        return "mock-provider"
        
    def chat_completion(self, messages, tools=None, tool_choice=None):
        return self.mock(messages, tools, tool_choice)

def test_run_command_nudge():
    """Test that a nudge is added after a successful run_command."""
    provider = MockProvider()
    
    # Mock the first response to call run_command
    mock_msg = MagicMock()
    mock_msg.content = None
    mock_tool_call = MagicMock()
    mock_tool_call.id = "call_1"
    mock_tool_call.function.name = "run_command"
    mock_tool_call.function.arguments = json.dumps({"sandbox_id": "sb1", "command": "apt install nginx"})
    mock_msg.tool_calls = [mock_tool_call]
    mock_msg.model_dump.return_value = {
        "role": "assistant",
        "tool_calls": [{
            "id": "call_1",
            "type": "function",
            "function": {"name": "run_command", "arguments": mock_tool_call.function.arguments}
        }]
    }
    
    provider.mock.return_value.choices = [MagicMock(message=mock_msg)]
    
    def tool_handler(name, args):
        if name == "run_command":
            return {"success": True, "output": "installed"}
        return {"error": "unknown tool"}
        
    agent = AgentLoop(
        provider=provider,
        system_prompt="You are a helper.",
        tools=[{"name": "run_command"}],
        tool_handler=tool_handler
    )
    
    agent.add_user_message("Install nginx")
    agent.step()
    
    # Check if a nudge message was added to history
    # messages: [system, user, assistant(tool_call), tool_result, nudge]
    # Actually, it might be added after the tool result.
    
    nudge_msg = agent.messages[-1]
    assert nudge_msg["role"] == "system"
    assert "Ansible playbook" in nudge_msg["content"]
    assert "add_task" in nudge_msg["content"]

def test_no_nudge_on_error():
    """Test that no nudge is added if run_command fails."""
    provider = MockProvider()
    
    mock_msg = MagicMock()
    mock_msg.content = None
    mock_tool_call = MagicMock()
    mock_tool_call.id = "call_1"
    mock_tool_call.function.name = "run_command"
    mock_tool_call.function.arguments = json.dumps({"sandbox_id": "sb1", "command": "bad command"})
    mock_msg.tool_calls = [mock_tool_call]
    mock_msg.model_dump.return_value = {
        "role": "assistant",
        "tool_calls": [{
            "id": "call_1",
            "type": "function",
            "function": {"name": "run_command", "arguments": mock_tool_call.function.arguments}
        }]
    }
    
    provider.mock.return_value.choices = [MagicMock(message=mock_msg)]
    
    def tool_handler(name, args):
        return {"error": "command failed"}
        
    agent = AgentLoop(
        provider=provider,
        system_prompt="You are a helper.",
        tools=[{"name": "run_command"}],
        tool_handler=tool_handler
    )
    
    agent.add_user_message("Run bad command")
    agent.step()
    
    # messages: [system, user, assistant(tool_call), tool_result]
    # No nudge should be there.
    assert agent.messages[-1]["role"] == "tool"
