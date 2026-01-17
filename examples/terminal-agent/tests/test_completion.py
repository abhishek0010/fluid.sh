
from session_manager import SessionState, TaskCompletionTool
from ansible_tools import PlaybookManager

def test_task_completion_tool():
    playbook_manager = PlaybookManager()
    session_state = SessionState(playbook_manager=playbook_manager)
    
    # Add some dummy state
    session_state.add_sandbox({"id": "sb-1", "vm_name": "test-vm", "source_vm_name": "ubuntu"})
    
    tool = TaskCompletionTool(session_state)
    assert tool.name == "task_complete"
    
    result = tool.execute(summary="All work done.")
    
    assert result.success
    assert result.data["status"] == "task_complete"
    assert result.data["summary"] == "All work done."
    assert result.data["session_stats"]["sandboxes_created"] == 1
