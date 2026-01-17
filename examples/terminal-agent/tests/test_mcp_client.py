
import pytest
from unittest.mock import MagicMock, AsyncMock, patch
from mcp_client import BaseMCPClient, MCPTool
import asyncio

@pytest.mark.asyncio
async def test_mcp_client_connect():
    mock_session = AsyncMock()
    
    # Define a simple Tool class to mimic mcp.types.Tool
    from dataclasses import dataclass
    @dataclass
    class MockTool:
        name: str
        description: str
        inputSchema: dict

    mock_session.list_tools.return_value = MagicMock(tools=[
        MockTool(name="test_tool", description="A test tool", inputSchema={"type": "object"})
    ])
    
    with patch("mcp_client.stdio_client") as mock_stdio:
        mock_stdio.return_value.__aenter__.return_value = (AsyncMock(), AsyncMock())
        with patch("mcp_client.ClientSession", return_value=mock_session):
            client = BaseMCPClient("test-command", ["arg1"])
            tools = await client.connect()
            
            assert len(tools) == 1
            assert tools[0].name == "mcp_test_tool"
            assert tools[0].description == "A test tool"

def test_mcp_tool_execute():
    mock_session = AsyncMock()
    # Mock result with content as Pydantic models (simplified)
    mock_content = MagicMock()
    mock_content.model_dump.return_value = {"type": "text", "text": "success"}
    
    mock_result = MagicMock()
    mock_result.is_error = False
    mock_result.content = [mock_content]
    
    mock_session.call_tool.return_value = mock_result
    
    tool = MCPTool(mock_session, "test_tool", "desc", {"type": "object"})
    
    # We need to mock the event loop's run_until_complete if we want to test strictly
    with patch("asyncio.get_event_loop") as mock_loop:
        mock_loop.return_value.run_until_complete.return_value = mock_result
        result = tool.execute(arg1="val1")
        
        assert result.success
        assert result.data["content"][0]["text"] == "success"
