from unittest.mock import patch, MagicMock
from ansible_tools import (
    InitPlaybookTool,
    AddTaskTool,
    ViewPlaybookTool,
    ValidatePlaybookTool,
    DryRunPlaybookTool,
    RunPlaybookTool,
    PlaybookManager
)

def test_init_playbook_tool():
    manager = PlaybookManager()
    tool = InitPlaybookTool(manager)
    
    assert tool.name == "init_playbook"
    assert "name" in tool.parameters["properties"]
    
    # Test execution
    result = tool.execute(name="test_playbook", hosts="localhost")
    assert result.success
    assert len(manager.playbook) == 1
    assert manager.playbook[0]["name"] == "test_playbook"
    assert manager.playbook[0]["hosts"] == "localhost"
    assert manager.playbook[0]["tasks"] == []

def test_add_task_tool():
    manager = PlaybookManager()
    manager.init_playbook("test", "all")
    tool = AddTaskTool(manager)
    
    assert tool.name == "add_task"
    assert "module" in tool.parameters["properties"]
    
    # Test execution
    result = tool.execute(
        name="Install nginx", 
        module="apt", 
        args={"name": "nginx", "state": "present"}
    )
    assert result.success
    assert len(manager.playbook[0]["tasks"]) == 1
    task = manager.playbook[0]["tasks"][0]
    assert task["name"] == "Install nginx"
    assert task["apt"] == {"name": "nginx", "state": "present"}

def test_add_task_without_init():
    manager = PlaybookManager()
    tool = AddTaskTool(manager)
    
    result = tool.execute(
        name="Task", 
        module="shell", 
        args={"cmd": "echo hi"}
    )
    assert not result.success
    assert "No playbook initialized" in result.error_message

def test_playbook_manager_yaml():
    manager = PlaybookManager()
    manager.init_playbook("test", "all")
    yaml_str = manager.to_yaml()
    assert "name: test" in yaml_str
    assert "hosts: all" in yaml_str

def test_view_playbook_tool():
    manager = PlaybookManager()
    manager.init_playbook("test", "all")
    tool = ViewPlaybookTool(manager)
    
    assert tool.name == "view_playbook"
    
    result = tool.execute()
    assert result.success
    assert "playbook_yaml" in result.data
    assert "name: test" in result.data["playbook_yaml"]

def test_validate_playbook_tool():
    manager = PlaybookManager()
    tool = ValidatePlaybookTool(manager)
    
    # Test empty playbook
    result = tool.execute()
    assert not result.success
    assert "Playbook is empty" in result.error_message
    
    # Test valid playbook (satisfying ansible-lint)
    manager.init_playbook("Test Playbook", "all")
    manager.add_task("Ping hosts", "ansible.builtin.ping", {})
    result = tool.execute()
    assert result.success
    assert result.data["is_valid"] is True
    assert "playbook_yaml" in result.data

def test_playbook_manager_validation():
    manager = PlaybookManager()
    
    # Empty
    valid, errors = manager.validate_playbook()
    assert not valid
    assert len(errors) > 0
    
    # Valid (satisfying ansible-lint)
    manager.init_playbook("Test Playbook", "all")
    manager.add_task("Test task", "ansible.builtin.command", {"cmd": "ls"})
    # Manually add changed_when to satisfy lint for this test
    manager.playbook[0]["tasks"][0]["changed_when"] = False
    valid, errors = manager.validate_playbook()
    assert valid, f"Validation failed with errors: {errors}"
    assert len(errors) == 0
    
    # Invalid task (missing module) - bypass add_task to create invalid state
    manager.playbook[0]["tasks"] = [{"name": "invalid task"}]
    valid, errors = manager.validate_playbook()
    assert not valid
    # Since we use external tools now, the error message might be from them
    assert any("no module specified" in e or "syntax" in e.lower() or "lint" in e.lower() for e in errors)
    
    # Invalid task (missing name)
    manager.playbook[0]["tasks"] = [{"shell": "ls"}]
    valid, errors = manager.validate_playbook()
    assert not valid
    assert any("missing 'name'" in e or "syntax" in e.lower() or "lint" in e.lower() for e in errors)


def test_dry_run_playbook_tool():
    manager = PlaybookManager()
    manager.init_playbook("test", "all")
    tool = DryRunPlaybookTool(manager)
    
    assert tool.name == "dry_run_playbook"
    
    with patch("subprocess.run") as mock_run:
        # Mock success
        mock_run.return_value = MagicMock(returncode=0, stdout="PLAY RECAP...", stderr="")
        
        result = tool.execute(target_host="1.2.3.4")
        
        assert result.success
        assert "stdout" in result.data
        assert result.data["stdout"] == "PLAY RECAP..."
        
        # Verify subprocess call
        args, kwargs = mock_run.call_args
        cmd = args[0]
        assert "ansible-playbook" in cmd
        assert "--check" in cmd
        assert "-i" in cmd
        assert "1.2.3.4," in cmd
        assert kwargs["env"]["ANSIBLE_HOST_KEY_CHECKING"] == "False"
        
        # Mock failure
        mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="Connection failed")
        
        result = tool.execute(target_host="1.2.3.4")
        assert not result.success
        assert "Connection failed" in result.error_message


def test_run_playbook_tool():
    manager = PlaybookManager()
    manager.init_playbook("test", "all")
    tool = RunPlaybookTool(manager)
    
    assert tool.name == "run_playbook"
    
    # Test without confirmation
    result = tool.execute(target_host="1.2.3.4", confirm=False)
    assert not result.success
    assert "not confirmed" in result.error_message.lower()

    with patch("subprocess.run") as mock_run:
        # Mock success
        mock_run.return_value = MagicMock(returncode=0, stdout="PLAY RECAP...", stderr="")
        
        result = tool.execute(target_host="1.2.3.4", confirm=True)
        
        assert result.success
        assert "stdout" in result.data
        assert result.data["stdout"] == "PLAY RECAP..."
        
        # Verify subprocess call (no --check)
        args, kwargs = mock_run.call_args
        cmd = args[0]
        assert "ansible-playbook" in cmd
        assert "--check" not in cmd
        assert "-i" in cmd
        assert "1.2.3.4," in cmd
