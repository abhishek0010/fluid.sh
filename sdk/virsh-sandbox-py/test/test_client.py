# coding: utf-8

"""
Tests for the VirshSandbox unified client.

This module tests that all client methods properly return dictionaries
instead of response objects.
"""

import unittest
from typing import Any, Dict, List
from unittest.mock import MagicMock, patch

from virsh_sandbox.client import (AccessOperations, AnsibleOperations,
                                  AuditOperations, CommandOperations,
                                  FileOperations, HealthOperations,
                                  HumanOperations, PlanOperations,
                                  SandboxOperations, TmuxOperations,
                                  VirshSandbox, VMsOperations, _to_dict)


class MockResponse:
    """Mock response object with to_dict method."""

    def __init__(self, data: Dict[str, Any]):
        self._data = data

    def to_dict(self) -> Dict[str, Any]:
        return self._data


class TestToDictHelper(unittest.TestCase):
    """Test the _to_dict helper function."""

    def test_to_dict_with_none(self) -> None:
        """Test _to_dict returns None for None input."""
        self.assertIsNone(_to_dict(None))

    def test_to_dict_with_dict(self) -> None:
        """Test _to_dict returns dict unchanged."""
        data = {"key": "value"}
        result = _to_dict(data)
        self.assertEqual(result, data)
        self.assertIsInstance(result, dict)

    def test_to_dict_with_response_object(self) -> None:
        """Test _to_dict converts response object to dict."""
        mock_response = MockResponse({"sandbox_id": "123", "status": "running"})
        result = _to_dict(mock_response)
        self.assertEqual(result, {"sandbox_id": "123", "status": "running"})
        self.assertIsInstance(result, dict)

    def test_to_dict_with_list_of_responses(self) -> None:
        """Test _to_dict converts list of response objects."""
        mock_responses = [
            MockResponse({"id": "1", "name": "session1"}),
            MockResponse({"id": "2", "name": "session2"}),
        ]
        result = _to_dict(mock_responses)
        self.assertEqual(
            result,
            [
                {"id": "1", "name": "session1"},
                {"id": "2", "name": "session2"},
            ],
        )
        self.assertIsInstance(result, list)
        for item in result:
            self.assertIsInstance(item, dict)

    def test_to_dict_with_list_of_dicts(self) -> None:
        """Test _to_dict handles list of dicts."""
        data = [{"a": 1}, {"b": 2}]
        result = _to_dict(data)
        self.assertEqual(result, data)

    def test_to_dict_with_primitive(self) -> None:
        """Test _to_dict returns primitives unchanged."""
        self.assertEqual(_to_dict("string"), "string")
        self.assertEqual(_to_dict(123), 123)
        self.assertEqual(_to_dict(True), True)


class TestAccessOperations(unittest.TestCase):
    """Test AccessOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = AccessOperations(self.mock_api)

    def test_v1_access_ca_pubkey_get_returns_dict(self) -> None:
        """Test v1_access_ca_pubkey_get returns dict."""
        self.mock_api.v1_access_ca_pubkey_get.return_value = MockResponse(
            {"public_key": "ssh-rsa AAAA..."}
        )
        result = self.ops.v1_access_ca_pubkey_get()
        self.assertIsInstance(result, dict)
        self.assertEqual(result["public_key"], "ssh-rsa AAAA...")

    def test_v1_access_certificate_cert_id_delete_returns_dict(self) -> None:
        """Test v1_access_certificate_cert_id_delete returns dict."""
        self.mock_api.v1_access_certificate_cert_id_delete.return_value = MockResponse(
            {"status": "revoked"}
        )
        result = self.ops.v1_access_certificate_cert_id_delete(
            cert_id="cert-123", reason="test"
        )
        self.assertIsInstance(result, dict)

    def test_v1_access_certificate_cert_id_get_returns_dict(self) -> None:
        """Test v1_access_certificate_cert_id_get returns dict."""
        self.mock_api.v1_access_certificate_cert_id_get.return_value = MockResponse(
            {"cert_id": "cert-123", "status": "active"}
        )
        result = self.ops.v1_access_certificate_cert_id_get(cert_id="cert-123")
        self.assertIsInstance(result, dict)

    def test_v1_access_certificates_get_returns_dict(self) -> None:
        """Test v1_access_certificates_get returns dict."""
        self.mock_api.v1_access_certificates_get.return_value = MockResponse(
            {"certificates": []}
        )
        result = self.ops.v1_access_certificates_get()
        self.assertIsInstance(result, dict)

    def test_v1_access_request_post_returns_dict(self) -> None:
        """Test v1_access_request_post returns dict."""
        self.mock_api.v1_access_request_post.return_value = MockResponse(
            {"certificate": "...", "expires_at": "2024-01-01"}
        )
        result = self.ops.v1_access_request_post(
            public_key="ssh-rsa ...", sandbox_id="sbx-123"
        )
        self.assertIsInstance(result, dict)

    def test_v1_access_session_end_post_returns_dict(self) -> None:
        """Test v1_access_session_end_post returns dict."""
        self.mock_api.v1_access_session_end_post.return_value = MockResponse(
            {"status": "ended"}
        )
        result = self.ops.v1_access_session_end_post(session_id="sess-123")
        self.assertIsInstance(result, dict)

    def test_v1_access_session_start_post_returns_dict(self) -> None:
        """Test v1_access_session_start_post returns dict."""
        self.mock_api.v1_access_session_start_post.return_value = MockResponse(
            {"session_id": "sess-123"}
        )
        result = self.ops.v1_access_session_start_post(certificate_id="cert-123")
        self.assertIsInstance(result, dict)

    def test_v1_access_sessions_get_returns_dict(self) -> None:
        """Test v1_access_sessions_get returns dict."""
        self.mock_api.v1_access_sessions_get.return_value = MockResponse(
            {"sessions": []}
        )
        result = self.ops.v1_access_sessions_get()
        self.assertIsInstance(result, dict)


class TestAnsibleOperations(unittest.TestCase):
    """Test AnsibleOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = AnsibleOperations(self.mock_api)

    def test_create_ansible_job_returns_dict(self) -> None:
        """Test create_ansible_job returns dict."""
        self.mock_api.create_ansible_job.return_value = MockResponse(
            {"job_id": "job-123", "status": "pending"}
        )
        result = self.ops.create_ansible_job(playbook="test.yml", vm_name="test-vm")
        self.assertIsInstance(result, dict)
        self.assertEqual(result["job_id"], "job-123")

    def test_get_ansible_job_returns_dict(self) -> None:
        """Test get_ansible_job returns dict."""
        self.mock_api.get_ansible_job.return_value = MockResponse(
            {"job_id": "job-123", "status": "completed", "output": "..."}
        )
        result = self.ops.get_ansible_job(job_id="job-123")
        self.assertIsInstance(result, dict)


class TestAuditOperations(unittest.TestCase):
    """Test AuditOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = AuditOperations(self.mock_api)

    def test_get_audit_stats_returns_dict(self) -> None:
        """Test get_audit_stats returns dict."""
        self.mock_api.get_audit_stats.return_value = MockResponse(
            {"total_events": 100, "by_type": {}}
        )
        result = self.ops.get_audit_stats()
        self.assertIsInstance(result, dict)

    def test_query_audit_log_returns_dict(self) -> None:
        """Test query_audit_log returns dict."""
        self.mock_api.query_audit_log.return_value = MockResponse({"entries": []})
        result = self.ops.query_audit_log()
        self.assertIsInstance(result, dict)


class TestCommandOperations(unittest.TestCase):
    """Test CommandOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = CommandOperations(self.mock_api)

    def test_get_allowed_commands_returns_dict(self) -> None:
        """Test get_allowed_commands returns dict."""
        self.mock_api.get_allowed_commands.return_value = MockResponse(
            {"commands": ["ls", "cat", "grep"]}
        )
        result = self.ops.get_allowed_commands()
        self.assertIsInstance(result, dict)

    def test_run_command_returns_dict(self) -> None:
        """Test run_command returns dict."""
        self.mock_api.run_command.return_value = MockResponse(
            {"exit_code": 0, "stdout": "file.txt", "stderr": ""}
        )
        result = self.ops.run_command(command="ls", args=["-la"])
        self.assertIsInstance(result, dict)
        self.assertEqual(result["exit_code"], 0)


class TestFileOperations(unittest.TestCase):
    """Test FileOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = FileOperations(self.mock_api)

    def test_check_file_exists_returns_dict(self) -> None:
        """Test check_file_exists returns dict."""
        self.mock_api.check_file_exists.return_value = MockResponse({"exists": True})
        result = self.ops.check_file_exists()
        self.assertIsInstance(result, dict)

    def test_copy_file_returns_dict(self) -> None:
        """Test copy_file returns dict."""
        self.mock_api.copy_file.return_value = MockResponse({"success": True})
        result = self.ops.copy_file(source="/src", destination="/dst")
        self.assertIsInstance(result, dict)

    def test_delete_file_returns_dict(self) -> None:
        """Test delete_file returns dict."""
        self.mock_api.delete_file.return_value = MockResponse({"deleted": True})
        result = self.ops.delete_file(path="/tmp/test.txt")
        self.assertIsInstance(result, dict)

    def test_edit_file_returns_dict(self) -> None:
        """Test edit_file returns dict."""
        self.mock_api.edit_file.return_value = MockResponse(
            {"modified": True, "matches": 1}
        )
        result = self.ops.edit_file(path="/tmp/test.txt", old_text="a", new_text="b")
        self.assertIsInstance(result, dict)

    def test_get_file_hash_returns_dict(self) -> None:
        """Test get_file_hash returns dict."""
        self.mock_api.get_file_hash.return_value = MockResponse(
            {"hash": "abc123", "algorithm": "sha256"}
        )
        result = self.ops.get_file_hash()
        self.assertIsInstance(result, dict)

    def test_list_directory_returns_dict(self) -> None:
        """Test list_directory returns dict."""
        self.mock_api.list_directory.return_value = MockResponse(
            {"files": [{"name": "test.txt", "size": 100}]}
        )
        result = self.ops.list_directory(path="/tmp")
        self.assertIsInstance(result, dict)

    def test_read_file_returns_dict(self) -> None:
        """Test read_file returns dict."""
        self.mock_api.read_file.return_value = MockResponse(
            {"content": "Hello World", "lines": 1}
        )
        result = self.ops.read_file(path="/tmp/test.txt")
        self.assertIsInstance(result, dict)

    def test_write_file_returns_dict(self) -> None:
        """Test write_file returns dict."""
        self.mock_api.write_file.return_value = MockResponse(
            {"written": True, "bytes": 11}
        )
        result = self.ops.write_file(path="/tmp/test.txt", content="Hello World")
        self.assertIsInstance(result, dict)


class TestHealthOperations(unittest.TestCase):
    """Test HealthOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = HealthOperations(self.mock_api)

    def test_get_health_returns_dict(self) -> None:
        """Test get_health returns dict."""
        self.mock_api.get_health.return_value = MockResponse(
            {"status": "healthy", "components": {}}
        )
        result = self.ops.get_health()
        self.assertIsInstance(result, dict)
        self.assertEqual(result["status"], "healthy")


class TestHumanOperations(unittest.TestCase):
    """Test HumanOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = HumanOperations(self.mock_api)

    def test_ask_human_returns_dict(self) -> None:
        """Test ask_human returns dict."""
        self.mock_api.ask_human.return_value = MockResponse(
            {"approved": True, "response": "yes"}
        )
        result = self.ops.ask_human(prompt="Continue?")
        self.assertIsInstance(result, dict)

    def test_ask_human_async_returns_dict(self) -> None:
        """Test ask_human_async returns dict."""
        self.mock_api.ask_human_async.return_value = MockResponse(
            {"request_id": "req-123"}
        )
        result = self.ops.ask_human_async(prompt="Continue?")
        self.assertIsInstance(result, dict)

    def test_cancel_approval_returns_dict(self) -> None:
        """Test cancel_approval returns dict."""
        self.mock_api.cancel_approval.return_value = MockResponse({"cancelled": True})
        result = self.ops.cancel_approval(request_id="req-123")
        self.assertIsInstance(result, dict)

    def test_get_pending_approval_returns_dict(self) -> None:
        """Test get_pending_approval returns dict."""
        self.mock_api.get_pending_approval.return_value = MockResponse(
            {"request_id": "req-123", "prompt": "Continue?", "status": "pending"}
        )
        result = self.ops.get_pending_approval(request_id="req-123")
        self.assertIsInstance(result, dict)

    def test_list_pending_approvals_returns_dict(self) -> None:
        """Test list_pending_approvals returns dict."""
        self.mock_api.list_pending_approvals.return_value = MockResponse(
            {"approvals": []}
        )
        result = self.ops.list_pending_approvals()
        self.assertIsInstance(result, dict)

    def test_respond_to_approval_returns_dict(self) -> None:
        """Test respond_to_approval returns dict."""
        self.mock_api.respond_to_approval.return_value = MockResponse(
            {"approved": True}
        )
        result = self.ops.respond_to_approval(request_id="req-123", approved=True)
        self.assertIsInstance(result, dict)


class TestPlanOperations(unittest.TestCase):
    """Test PlanOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = PlanOperations(self.mock_api)

    def test_abort_plan_returns_dict(self) -> None:
        """Test abort_plan returns dict."""
        self.mock_api.abort_plan.return_value = MockResponse({"aborted": True})
        result = self.ops.abort_plan(plan_id="plan-123")
        self.assertIsInstance(result, dict)

    def test_advance_plan_step_returns_dict(self) -> None:
        """Test advance_plan_step returns dict."""
        self.mock_api.advance_plan_step.return_value = MockResponse({"current_step": 2})
        result = self.ops.advance_plan_step(plan_id="plan-123")
        self.assertIsInstance(result, dict)

    def test_create_plan_returns_dict(self) -> None:
        """Test create_plan returns dict."""
        self.mock_api.create_plan.return_value = MockResponse(
            {"plan_id": "plan-123", "status": "created"}
        )
        result = self.ops.create_plan(name="Test Plan", steps=["step1", "step2"])
        self.assertIsInstance(result, dict)

    def test_delete_plan_returns_dict(self) -> None:
        """Test delete_plan returns dict."""
        self.mock_api.delete_plan.return_value = MockResponse({"deleted": True})
        result = self.ops.delete_plan(plan_id="plan-123")
        self.assertIsInstance(result, dict)

    def test_get_plan_returns_dict(self) -> None:
        """Test get_plan returns dict."""
        self.mock_api.get_plan.return_value = MockResponse(
            {"plan_id": "plan-123", "name": "Test Plan", "steps": []}
        )
        result = self.ops.get_plan(plan_id="plan-123")
        self.assertIsInstance(result, dict)

    def test_list_plans_returns_dict(self) -> None:
        """Test list_plans returns dict."""
        self.mock_api.list_plans.return_value = MockResponse({"plans": []})
        result = self.ops.list_plans()
        self.assertIsInstance(result, dict)

    def test_update_plan_returns_dict(self) -> None:
        """Test update_plan returns dict."""
        self.mock_api.update_plan.return_value = MockResponse({"updated": True})
        result = self.ops.update_plan(plan_id="plan-123", step_index=0, result="done")
        self.assertIsInstance(result, dict)


class TestSandboxOperations(unittest.TestCase):
    """Test SandboxOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = SandboxOperations(self.mock_api)

    def test_create_sandbox_returns_dict(self) -> None:
        """Test create_sandbox returns dict."""
        self.mock_api.create_sandbox.return_value = MockResponse(
            {
                "sandbox": {
                    "id": "SBX-123",
                    "agent_id": "agent-123",
                    "state": "CREATED",
                }
            }
        )
        result = self.ops.create_sandbox(agent_id="agent-123", source_vm_name="base-vm")
        self.assertIsInstance(result, dict)
        self.assertIn("sandbox", result)
        self.assertIsInstance(result["sandbox"], dict)

    def test_create_sandbox_session_returns_dict(self) -> None:
        """Test create_sandbox_session returns dict."""
        self.mock_api.create_sandbox_session.return_value = MockResponse(
            {"session_name": "sess-123", "certificate": "..."}
        )
        result = self.ops.create_sandbox_session(sandbox_id="SBX-123")
        self.assertIsInstance(result, dict)

    def test_create_snapshot_returns_dict(self) -> None:
        """Test create_snapshot returns dict."""
        self.mock_api.create_snapshot.return_value = MockResponse(
            {"snapshot_id": "snap-123", "name": "test-snapshot"}
        )
        result = self.ops.create_snapshot(id="SBX-123", name="test-snapshot")
        self.assertIsInstance(result, dict)

    def test_diff_snapshots_returns_dict(self) -> None:
        """Test diff_snapshots returns dict."""
        self.mock_api.diff_snapshots.return_value = MockResponse(
            {"changes": [], "summary": {}}
        )
        result = self.ops.diff_snapshots(
            id="SBX-123", from_snapshot="snap-1", to_snapshot="snap-2"
        )
        self.assertIsInstance(result, dict)

    def test_get_sandbox_session_returns_dict(self) -> None:
        """Test get_sandbox_session returns dict."""
        self.mock_api.get_sandbox_session.return_value = MockResponse(
            {"session_name": "sess-123", "sandbox_id": "SBX-123"}
        )
        result = self.ops.get_sandbox_session(session_name="sess-123")
        self.assertIsInstance(result, dict)

    def test_kill_sandbox_session_returns_dict(self) -> None:
        """Test kill_sandbox_session returns dict."""
        self.mock_api.kill_sandbox_session.return_value = MockResponse({"killed": True})
        result = self.ops.kill_sandbox_session(session_name="sess-123")
        self.assertIsInstance(result, dict)

    def test_list_sandbox_sessions_returns_dict(self) -> None:
        """Test list_sandbox_sessions returns dict."""
        self.mock_api.list_sandbox_sessions.return_value = MockResponse(
            {"sessions": []}
        )
        result = self.ops.list_sandbox_sessions()
        self.assertIsInstance(result, dict)

    def test_run_sandbox_command_returns_dict(self) -> None:
        """Test run_sandbox_command returns dict."""
        self.mock_api.run_sandbox_command.return_value = MockResponse(
            {"exit_code": 0, "stdout": "output", "stderr": ""}
        )
        result = self.ops.run_sandbox_command(
            id="SBX-123",
            command="ls -la",
            username="root",
            private_key_path="/path/to/key",
        )
        self.assertIsInstance(result, dict)

    def test_sandbox_api_health_returns_dict(self) -> None:
        """Test sandbox_api_health returns dict."""
        self.mock_api.sandbox_api_health.return_value = MockResponse(
            {"status": "healthy"}
        )
        result = self.ops.sandbox_api_health()
        self.assertIsInstance(result, dict)

    def test_start_sandbox_returns_dict(self) -> None:
        """Test start_sandbox returns dict."""
        self.mock_api.start_sandbox.return_value = MockResponse(
            {"sandbox": {"id": "SBX-123", "state": "RUNNING", "ip_address": "10.0.0.1"}}
        )
        result = self.ops.start_sandbox(id="SBX-123", wait_for_ip=True)
        self.assertIsInstance(result, dict)


class TestTmuxOperations(unittest.TestCase):
    """Test TmuxOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = TmuxOperations(self.mock_api)

    def test_create_tmux_pane_returns_dict(self) -> None:
        """Test create_tmux_pane returns dict."""
        self.mock_api.create_tmux_pane.return_value = MockResponse(
            {"pane_id": "%1", "window_name": "main"}
        )
        result = self.ops.create_tmux_pane(session_name="test-session")
        self.assertIsInstance(result, dict)

    def test_create_tmux_session_returns_dict(self) -> None:
        """Test create_tmux_session returns dict."""
        self.mock_api.create_tmux_session.return_value = MockResponse(
            {"session_name": "test-session"}
        )
        result = self.ops.create_tmux_session()
        self.assertIsInstance(result, dict)

    def test_kill_tmux_pane_returns_dict(self) -> None:
        """Test kill_tmux_pane returns dict."""
        self.mock_api.kill_tmux_pane.return_value = MockResponse({"killed": True})
        result = self.ops.kill_tmux_pane(pane_id="%1")
        self.assertIsInstance(result, dict)

    def test_kill_tmux_session_returns_dict(self) -> None:
        """Test kill_tmux_session returns dict."""
        self.mock_api.kill_tmux_session.return_value = MockResponse({"killed": True})
        result = self.ops.kill_tmux_session(session_name="test-session")
        self.assertIsInstance(result, dict)

    def test_list_tmux_panes_returns_dict(self) -> None:
        """Test list_tmux_panes returns dict."""
        self.mock_api.list_tmux_panes.return_value = MockResponse({"panes": []})
        result = self.ops.list_tmux_panes(session="test-session")
        self.assertIsInstance(result, dict)

    def test_list_tmux_sessions_returns_list_of_dicts(self) -> None:
        """Test list_tmux_sessions returns list of dicts."""
        self.mock_api.list_tmux_sessions.return_value = [
            MockResponse({"name": "session1", "windows": 1}),
            MockResponse({"name": "session2", "windows": 2}),
        ]
        result = self.ops.list_tmux_sessions()
        self.assertIsInstance(result, list)
        for item in result:
            self.assertIsInstance(item, dict)

    def test_list_tmux_windows_returns_list_of_dicts(self) -> None:
        """Test list_tmux_windows returns list of dicts."""
        self.mock_api.list_tmux_windows.return_value = [
            MockResponse({"name": "window1", "index": 0}),
            MockResponse({"name": "window2", "index": 1}),
        ]
        result = self.ops.list_tmux_windows(session="test-session")
        self.assertIsInstance(result, list)
        for item in result:
            self.assertIsInstance(item, dict)

    def test_read_tmux_pane_returns_dict(self) -> None:
        """Test read_tmux_pane returns dict."""
        self.mock_api.read_tmux_pane.return_value = MockResponse(
            {"content": "$ ls\nfile.txt", "lines": 2}
        )
        result = self.ops.read_tmux_pane(pane_id="%1")
        self.assertIsInstance(result, dict)

    def test_release_tmux_session_returns_dict(self) -> None:
        """Test release_tmux_session returns dict."""
        self.mock_api.release_tmux_session.return_value = MockResponse(
            {"released": True}
        )
        result = self.ops.release_tmux_session(session_id="sess-123")
        self.assertIsInstance(result, dict)

    def test_send_keys_to_pane_returns_dict(self) -> None:
        """Test send_keys_to_pane returns dict."""
        self.mock_api.send_keys_to_pane.return_value = MockResponse({"sent": True})
        result = self.ops.send_keys_to_pane(pane_id="%1", key="Enter")
        self.assertIsInstance(result, dict)

    def test_switch_tmux_pane_returns_dict(self) -> None:
        """Test switch_tmux_pane returns dict."""
        self.mock_api.switch_tmux_pane.return_value = MockResponse({"switched": True})
        result = self.ops.switch_tmux_pane(pane_id="%1")
        self.assertIsInstance(result, dict)


class TestVMsOperations(unittest.TestCase):
    """Test VMsOperations returns dictionaries."""

    def setUp(self) -> None:
        self.mock_api = MagicMock()
        self.ops = VMsOperations(self.mock_api)

    def test_list_virtual_machines_returns_dict(self) -> None:
        """Test list_virtual_machines returns dict."""
        self.mock_api.list_virtual_machines.return_value = MockResponse(
            {
                "vms": [
                    {"name": "vm1", "state": "running"},
                    {"name": "vm2", "state": "stopped"},
                ]
            }
        )
        result = self.ops.list_virtual_machines()
        self.assertIsInstance(result, dict)
        self.assertIn("vms", result)


class TestVirshSandboxClient(unittest.TestCase):
    """Test the main VirshSandbox client."""

    def test_client_initialization(self) -> None:
        """Test client initializes correctly."""
        client = VirshSandbox(host="http://localhost:8080")
        self.assertIsNotNone(client)
        self.assertEqual(client.configuration.host, "http://localhost:8080")

    def test_client_with_separate_tmux_host(self) -> None:
        """Test client with separate tmux host."""
        client = VirshSandbox(
            host="http://localhost:8080", tmux_host="http://localhost:8081"
        )
        self.assertEqual(client.configuration.host, "http://localhost:8080")
        self.assertEqual(client.tmux_configuration.host, "http://localhost:8081")

    def test_client_properties_return_operations(self) -> None:
        """Test client properties return correct operation classes."""
        client = VirshSandbox()
        self.assertIsInstance(client.access, AccessOperations)
        self.assertIsInstance(client.ansible, AnsibleOperations)
        self.assertIsInstance(client.audit, AuditOperations)
        self.assertIsInstance(client.command, CommandOperations)
        self.assertIsInstance(client.file, FileOperations)
        self.assertIsInstance(client.health, HealthOperations)
        self.assertIsInstance(client.human, HumanOperations)
        self.assertIsInstance(client.plan, PlanOperations)
        self.assertIsInstance(client.sandbox, SandboxOperations)
        self.assertIsInstance(client.tmux, TmuxOperations)
        self.assertIsInstance(client.vms, VMsOperations)

    def test_client_context_manager(self) -> None:
        """Test client can be used as context manager."""
        with VirshSandbox() as client:
            self.assertIsNotNone(client)

    def test_client_set_debug(self) -> None:
        """Test setting debug mode."""
        client = VirshSandbox()
        client.set_debug(True)
        self.assertTrue(client.configuration.debug)


class TestReturnTypeConsistency(unittest.TestCase):
    """Test that all operations consistently return dicts or lists of dicts."""

    def test_all_sandbox_operations_return_dict(self) -> None:
        """Verify SandboxOperations methods return dict type."""
        mock_api = MagicMock()
        ops = SandboxOperations(mock_api)

        # Set up mock returns
        mock_api.create_sandbox.return_value = MockResponse({"sandbox": {}})
        mock_api.create_sandbox_session.return_value = MockResponse({})
        mock_api.create_snapshot.return_value = MockResponse({})
        mock_api.diff_snapshots.return_value = MockResponse({})
        mock_api.get_sandbox_session.return_value = MockResponse({})
        mock_api.kill_sandbox_session.return_value = MockResponse({})
        mock_api.list_sandbox_sessions.return_value = MockResponse({})
        mock_api.run_sandbox_command.return_value = MockResponse({})
        mock_api.sandbox_api_health.return_value = MockResponse({})
        mock_api.start_sandbox.return_value = MockResponse({})

        # Test all methods
        self.assertIsInstance(ops.create_sandbox(), dict)
        self.assertIsInstance(ops.create_sandbox_session(), dict)
        self.assertIsInstance(ops.create_snapshot(id="test"), dict)
        self.assertIsInstance(ops.diff_snapshots(id="test"), dict)
        self.assertIsInstance(ops.get_sandbox_session(session_name="test"), dict)
        self.assertIsInstance(ops.kill_sandbox_session(session_name="test"), dict)
        self.assertIsInstance(ops.list_sandbox_sessions(), dict)
        self.assertIsInstance(ops.run_sandbox_command(id="test"), dict)
        self.assertIsInstance(ops.sandbox_api_health(), dict)
        self.assertIsInstance(ops.start_sandbox(id="test"), dict)


if __name__ == "__main__":
    unittest.main()
