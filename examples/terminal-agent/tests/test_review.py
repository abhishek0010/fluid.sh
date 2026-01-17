import pytest
from unittest.mock import MagicMock, patch
from agent import AgentLoop, AgentResponse
from session_manager import SessionState, RequestReviewTool
from tools import ToolRegistry, ToolExecutionResult

def test_request_review_tool():
    session_state = SessionState()
    tool = RequestReviewTool(session_state)
    
    result = tool.execute(reason="Testing review")
    
    assert result.success is True
    assert result.data["status"] == "awaiting_review"
    assert result.data["reason"] == "Testing review"
    assert "summary" in result.data

def test_agent_loop_awaiting_input_flag():
    # Mock provider
    provider = MagicMock()
    
    # Mock response with request_review tool call
    mock_tool_call = MagicMock()
    mock_tool_call.id = "call_1"
    mock_tool_call.function.name = "request_review"
    mock_tool_call.function.arguments = '{"reason": "test"}'
    
    mock_message = MagicMock()
    mock_message.tool_calls = [mock_tool_call]
    mock_message.content = None
    mock_message.model_dump.return_value = {"role": "assistant", "tool_calls": []}
    
    provider.chat_completion.return_value.choices = [MagicMock(message=mock_message)]
    
    # Tool handler
    def tool_handler(name, args):
        if name == "request_review":
            return {"status": "awaiting_review"}
        return {}

    agent = AgentLoop(
        provider=provider,
        system_prompt="test",
        tools=[{"name": "request_review"}],
        tool_handler=tool_handler
    )
    
    response = agent.step()
    
    assert response.awaiting_input is True
    assert any(tr.name == "request_review" for tr in response.tool_results)

def test_agent_run_stops_on_review():
    # Mock provider
    provider = MagicMock()
    
    # First turn: request review
    mock_tool_call = MagicMock()
    mock_tool_call.id = "call_1"
    mock_tool_call.function.name = "request_review"
    mock_tool_call.function.arguments = '{"reason": "test"}'
    
    msg1 = MagicMock()
    msg1.tool_calls = [mock_tool_call]
    msg1.content = None
    msg1.model_dump.return_value = {"role": "assistant", "tool_calls": [{"id": "call_1", "function": {"name": "request_review", "arguments": '{"reason": "test"}'}}]}
    
    provider.chat_completion.side_effect = [
        MagicMock(choices=[MagicMock(message=msg1)]),
        # This shouldn't be called because run should stop
        MagicMock(choices=[MagicMock(message=MagicMock(content="I continued", tool_calls=None))])
    ]
    
    def tool_handler(name, args):
        return {"status": "awaiting_review"}

    agent = AgentLoop(
        provider=provider,
        system_prompt="test",
        tools=[{"name": "request_review"}],
        tool_handler=tool_handler
    )
    
    responses = agent.run("hello")
    
    assert len(responses) == 1
    assert responses[0].awaiting_input is True
    assert provider.chat_completion.call_count == 1
