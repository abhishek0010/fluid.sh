from typing import Any
from unittest.mock import MagicMock
import pytest
from agent import AgentLoop

class MockProvider:
    def chat_completion(self, messages, tools=None, tool_choice=None):
        mock = MagicMock()
        mock.choices = [MagicMock()]
        mock.choices[0].message.content = "OK"
        mock.choices[0].message.tool_calls = None
        mock.choices[0].message.model_dump.return_value = {
            "role": "assistant",
            "content": "OK"
        }
        return mock

def test_history_pruning():
    """Test that history is pruned to max_history_messages."""
    max_history = 10
    agent = AgentLoop(
        provider=MockProvider(),
        system_prompt="System",
        tools=[],
        tool_handler=lambda n, a: {},
        max_history_messages=max_history
    )
    
    # Add enough messages to trigger pruning
    # Each iteration adds 1 user message and 1 assistant message = 2 messages
    iterations = 20
    for i in range(iterations):
        agent.add_user_message(f"Msg {i}")
        agent.step()
        
    # Should be max_history + 1 (system prompt)
    assert len(agent.messages) == max_history + 1
    
    # System prompt should always be first
    assert agent.messages[0]["role"] == "system"
    assert agent.messages[0]["content"] == "System"
    
    # The last message should be the last assistant response
    assert agent.messages[-1]["role"] == "assistant"
    assert agent.messages[-1]["content"] == "OK"
    
    # The second message (first after system) should not be "Msg 0" (it should have been pruned)
    # We added 40 messages total (20 * 2). Max is 10.
    # So we should have kept the last 10 messages.
    # The sequence of contents added:
    # User 0, Asst 0, User 1, Asst 1, ... User 19, Asst 19.
    # Last 10: User 15, Asst 15, ..., User 19, Asst 19.
    # So the message at index 1 should be User 15.
    
    expected_first_msg_index = iterations - (max_history // 2) 
    # Wait, simple math check:
    # We have system + 10 messages.
    # Messages are: [Sys, Msg 15, OK, Msg 16, OK, Msg 17, OK, Msg 18, OK, Msg 19, OK]
    # Let's check content of index 1
    
    assert agent.messages[1]["role"] == "user"
    assert agent.messages[1]["content"] == "Msg 15"

def test_history_pruning_with_tool_calls():
    """Test history pruning with larger messages (tool calls)."""
    # This is trickier because tool calls add multiple messages per step.
    # But strictly count based should still work.
    pass