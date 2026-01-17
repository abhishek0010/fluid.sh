
import pytest
from unittest.mock import MagicMock, AsyncMock, patch
import os
from mcp_manager import MCPManager
from main import discover_mcp_servers, create_agent

@pytest.mark.asyncio
async def test_discover_slack_mcp():
    mcp_manager = MCPManager()
    
    with patch.dict(os.environ, {"SLACK_BOT_TOKEN": "xoxb-test", "SLACK_APP_TOKEN": "xapp-test"}):
        with patch.object(mcp_manager, "add_server", new_callable=AsyncMock) as mock_add:
            mock_add.return_value = []
            await discover_mcp_servers(mcp_manager)
            
            mock_add.assert_called_once()
            args, kwargs = mock_add.call_args
            assert args[0] == "slack"
            assert args[1] == "npx"
            assert "--y" in args[2] or "-y" in args[2]
            assert "@modelcontextprotocol/server-slack" in args[2]
            assert kwargs["env"]["SLACK_BOT_TOKEN"] == "xoxb-test"
            assert kwargs["env"]["SLACK_APP_TOKEN"] == "xapp-test"

@pytest.mark.asyncio
async def test_discover_github_mcp():
    mcp_manager = MCPManager()
    
    with patch.dict(os.environ, {"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp-test"}):
        with patch.object(mcp_manager, "add_server", new_callable=AsyncMock) as mock_add:
            mock_add.return_value = []
            await discover_mcp_servers(mcp_manager)
            
            # Should be called for GitHub (and Slack if tokens were present, but they aren't here)
            mock_add.assert_called_once()
            args, kwargs = mock_add.call_args
            assert args[0] == "github"
            assert args[1] == "npx"
            assert "@modelcontextprotocol/server-github" in args[2]
            assert kwargs["env"]["GITHUB_PERSONAL_ACCESS_TOKEN"] == "ghp-test"

from tools import ToolDefinition

@pytest.mark.asyncio
async def test_mcp_tools_registered_in_agent():
    mock_provider = MagicMock()
    mock_tool = MagicMock()
    mock_tool.name = "mcp_slack_post_message"
    mock_tool.description = "Post a message"
    mock_tool.parameters = {"type": "object"}
    
    # Mock get_definition to return a real ToolDefinition object
    mock_tool.get_definition.return_value = ToolDefinition(
        name="mcp_slack_post_message",
        description="Post a message",
        parameters={"type": "object"}
    )
    
    with patch.dict(os.environ, {"LLM_API_KEY": "test-key"}):
        with patch("main.discover_mcp_servers", new_callable=AsyncMock):
            with patch("main.create_provider", return_value=mock_provider):
                with patch("main.MCPManager") as MockMCPManager:
                    mock_mcp = MockMCPManager.return_value
                    mock_mcp.all_tools = [mock_tool]
                    # Mock register_tools to actually add the tool to registry
                    def mock_register(reg):
                        reg.register(mock_tool)
                    mock_mcp.register_tools.side_effect = mock_register
                    
                    agent, provider_type, model, mcp_manager = await create_agent()
                    
                    # Check if the tool is in the agent's tools
                    tool_names = [t["function"]["name"] for t in agent.tools]
                    assert "mcp_slack_post_message" in tool_names

@pytest.mark.asyncio
async def test_mcp_manager_disconnect_all():
    mcp_manager = MCPManager()
    mock_client = AsyncMock()
    mcp_manager.clients["test"] = mock_client
    
    await mcp_manager.disconnect_all()
    
    mock_client.disconnect.assert_called_once()
    assert len(mcp_manager.clients) == 0
