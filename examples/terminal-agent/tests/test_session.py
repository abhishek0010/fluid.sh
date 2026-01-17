import pytest
from datetime import datetime
from session_manager import SessionState, CommandEntry, SandboxEntry, ViewSessionTool
from ansible_tools import PlaybookManager

def test_session_state_init():
    session = SessionState()
    assert session.sandboxes == []
    assert session.commands == []
    assert isinstance(session.playbook_manager, PlaybookManager)
    assert isinstance(session.start_time, datetime)

def test_add_sandbox():
    session = SessionState()
    sandbox_data = {
        "id": "sb-123",
        "vm_name": "test-vm",
        "source_vm_name": "base-vm",
        "ip_address": "192.168.122.10"
    }
    session.add_sandbox(sandbox_data)
    
    assert len(session.sandboxes) == 1
    entry = session.sandboxes[0]
    assert isinstance(entry, SandboxEntry)
    assert entry.sandbox_id == "sb-123"
    assert entry.vm_name == "test-vm"
    assert entry.source_vm == "base-vm"
    assert entry.ip_address == "192.168.122.10"

def test_add_command():
    session = SessionState()
    result_data = {
        "exit_code": 0,
        "stdout": "hello world",
        "stderr": ""
    }
    session.add_command("sb-123", "echo hello world", result_data)
    
    assert len(session.commands) == 1
    entry = session.commands[0]
    assert isinstance(entry, CommandEntry)
    assert entry.sandbox_id == "sb-123"
    assert entry.command == "echo hello world"
    assert entry.exit_code == 0
    assert entry.stdout == "hello world"

def test_get_summary():
    session = SessionState()
    session.add_sandbox({"id": "sb-1", "vm_name": "vm1"})
    session.add_command("sb-1", "ls", {"exit_code": 0})
    session.playbook_manager.init_playbook("test")
    session.playbook_manager.add_task("task1", "shell", {"cmd": "ls"})
    
    summary = session.get_summary()
    assert summary["sandboxes_created"] == 1
    assert summary["commands_executed"] == 1
    assert summary["playbook_tasks"] == 1
    assert len(summary["current_sandboxes"]) == 1
    assert summary["current_sandboxes"][0]["id"] == "sb-1"

def test_view_session_tool():
    session = SessionState()
    tool = ViewSessionTool(session)
    
    result = tool.execute()
    assert result.success is True
    assert result.data["sandboxes_created"] == 0
    assert result.data["commands_executed"] == 0
