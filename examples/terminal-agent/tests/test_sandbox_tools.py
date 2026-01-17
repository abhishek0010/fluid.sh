import pytest
from unittest.mock import MagicMock
import sys

# Mock virsh_sandbox module if needed before importing sandbox_tools
# This ensures that even if the package isn't installed, the test can run
if "virsh_sandbox" not in sys.modules:
    sys.modules["virsh_sandbox"] = MagicMock()

from sandbox_tools import CreateSandboxTool
from session_manager import SessionState

def test_create_sandbox_tool_init():
    mock_client = MagicMock()
    session = SessionState()
    tool = CreateSandboxTool(mock_client, session)
    assert tool.name == "create_sandbox"
    assert tool.description
    assert "source_vm_name" in tool.parameters["properties"]

def test_create_sandbox_success():
    mock_client = MagicMock()
    session = SessionState()
    
    # Setup mock response
    mock_response = MagicMock()
    mock_sandbox = MagicMock()
    
    # Configure to_dict to return a dict
    mock_sandbox.to_dict.return_value = {
        "id": "sbx-123", 
        "ip_address": "192.168.1.100",
        "name": "my-sandbox",
        "source_vm_name": "base-vm"
    }
    
    mock_response.sandbox = mock_sandbox
    mock_client.sandbox.create_sandbox.return_value = mock_response

    tool = CreateSandboxTool(mock_client, session)
    result = tool.execute(source_vm_name="base-vm", vm_name="my-sbx")

    assert result.success is True
    assert result.data["id"] == "sbx-123"
    assert result.data["ip_address"] == "192.168.1.100"
    
    # Verify session update
    assert len(session.sandboxes) == 1
    assert session.sandboxes[0].sandbox_id == "sbx-123"
    
    # Verify call
    mock_client.sandbox.create_sandbox.assert_called_once()
    call_kwargs = mock_client.sandbox.create_sandbox.call_args.kwargs
    assert call_kwargs["source_vm_name"] == "base-vm"
    assert call_kwargs["vm_name"] == "my-sbx"
    assert call_kwargs["auto_start"] is True
    assert call_kwargs["wait_for_ip"] is True

def test_create_sandbox_fallback_serialization():
    """Test fallback when to_dict doesn't exist."""
    mock_client = MagicMock()
    session = SessionState()
    
    mock_response = MagicMock()
    mock_sandbox = MagicMock()
    # Remove to_dict
    if hasattr(mock_sandbox, "to_dict"):
        del mock_sandbox.to_dict
    
    # Add model_dump for Pydantic v2 simulation
    mock_sandbox.model_dump.return_value = {"id": "sbx-pydantic"}
    
    mock_response.sandbox = mock_sandbox
    mock_client.sandbox.create_sandbox.return_value = mock_response

    tool = CreateSandboxTool(mock_client, session)
    result = tool.execute(source_vm_name="base-vm")

    assert result.success is True
    assert result.data["id"] == "sbx-pydantic"
    assert len(session.sandboxes) == 1

def test_create_sandbox_error():
    mock_client = MagicMock()
    session = SessionState()
    mock_client.sandbox.create_sandbox.side_effect = Exception("API connection failed")

    tool = CreateSandboxTool(mock_client, session)
    result = tool.execute(source_vm_name="base-vm")

    assert result.success is False
    assert "API connection failed" in result.error_message
    assert len(session.sandboxes) == 0


from sandbox_tools import RunCommandTool

def test_run_command_tool_init():
    mock_client = MagicMock()
    session = SessionState()
    tool = RunCommandTool(mock_client, session)
    assert tool.name == "run_command"
    assert tool.description
    assert "sandbox_id" in tool.parameters["properties"]
    assert "command" in tool.parameters["properties"]

def test_run_command_success():
    mock_client = MagicMock()
    session = SessionState()
    
    mock_response = MagicMock()
    mock_result = MagicMock()
    mock_result.to_dict.return_value = {"stdout": "hello", "stderr": "", "exit_code": 0}
    mock_response.result = mock_result
    mock_client.sandbox.run_command.return_value = mock_response

    tool = RunCommandTool(mock_client, session)
    result = tool.execute(sandbox_id="sbx-123", command="echo hello")

    assert result.success is True
    assert result.data["stdout"] == "hello"
    
    # Verify session update
    assert len(session.commands) == 1
    assert session.commands[0].command == "echo hello"
    assert session.commands[0].sandbox_id == "sbx-123"
    
    mock_client.sandbox.run_command.assert_called_once_with(
        sandbox_id="sbx-123",
        cmd="echo hello",
        request_timeout=30.0,
    )

def test_run_command_chained_disallowed():
    mock_client = MagicMock()
    session = SessionState()
    tool = RunCommandTool(mock_client, session)
    
    for op in ["&&", "||", ";", "|", "`"]:
        result = tool.execute(sandbox_id="sbx-123", command=f"echo hello {op} echo world")
        assert result.success is False
        assert "Chained commands are not allowed" in result.error_message
        assert len(session.commands) == 0

def test_run_command_error():
    mock_client = MagicMock()
    session = SessionState()
    mock_client.sandbox.run_command.side_effect = Exception("Command failed")

    tool = RunCommandTool(mock_client, session)
    result = tool.execute(sandbox_id="sbx-123", command="invalid command")

    assert result.success is False
    assert "Command failed" in result.error_message
    assert len(session.commands) == 0
