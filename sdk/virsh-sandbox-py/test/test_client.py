# coding: utf-8

"""
Tests for the VirshSandbox unified client.

This module tests that all client methods properly return Pydantic model
objects with full IDE autocomplete support.
"""

import unittest
from unittest.mock import MagicMock, patch

from pydantic import BaseModel

from virsh_sandbox.client import (AccessOperations, AnsibleOperations,
                                  AuditOperations, CommandOperations,
                                  FileOperations, HealthOperations,
                                  HumanOperations, PlanOperations,
                                  SandboxOperations, TmuxOperations,
                                  VirshSandbox, VMsOperations)


class MockPydanticModel(BaseModel):
    """Mock Pydantic model for testing."""

    id: str = ""
    name: str = ""


class TestAccessOperations(unittest.TestCase):
    """Test AccessOperations wrapper methods return Pydantic models."""

    def test_v1_access_ca_pubkey_get_returns_model(self) -> None:
        """Test v1_access_ca_pubkey_get returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_ca_pubkey_get.return_value = MockPydanticModel(id="test")
        ops = AccessOperations(mock_api)
        result = ops.v1_access_ca_pubkey_get()
        self.assertIsInstance(result, BaseModel)

    def test_v1_access_certificate_cert_id_delete_returns_model(self) -> None:
        """Test delete certificate returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_certificate_cert_id_delete.return_value = MockPydanticModel()
        ops = AccessOperations(mock_api)
        result = ops.v1_access_certificate_cert_id_delete(cert_id="test-cert")
        self.assertIsInstance(result, BaseModel)

    def test_v1_access_certificate_cert_id_get_returns_model(self) -> None:
        """Test get certificate returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_certificate_cert_id_get.return_value = MockPydanticModel()
        ops = AccessOperations(mock_api)
        result = ops.v1_access_certificate_cert_id_get(cert_id="test-cert")
        self.assertIsInstance(result, BaseModel)

    def test_v1_access_certificates_get_returns_model(self) -> None:
        """Test list certificates returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_certificates_get.return_value = MockPydanticModel()
        ops = AccessOperations(mock_api)
        result = ops.v1_access_certificates_get()
        self.assertIsInstance(result, BaseModel)

    def test_v1_access_request_post_returns_model(self) -> None:
        """Test request access returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_request_post.return_value = MockPydanticModel()
        ops = AccessOperations(mock_api)
        result = ops.v1_access_request_post()
        self.assertIsInstance(result, BaseModel)

    def test_v1_access_session_end_post_returns_model(self) -> None:
        """Test session end returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_session_end_post.return_value = MockPydanticModel()
        ops = AccessOperations(mock_api)
        result = ops.v1_access_session_end_post()
        self.assertIsInstance(result, BaseModel)

    def test_v1_access_session_start_post_returns_model(self) -> None:
        """Test session start returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_session_start_post.return_value = MockPydanticModel()
        ops = AccessOperations(mock_api)
        result = ops.v1_access_session_start_post()
        self.assertIsInstance(result, BaseModel)

    def test_v1_access_sessions_get_returns_model(self) -> None:
        """Test list sessions returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.v1_access_sessions_get.return_value = MockPydanticModel()
        ops = AccessOperations(mock_api)
        result = ops.v1_access_sessions_get()
        self.assertIsInstance(result, BaseModel)


class TestSandboxOperations(unittest.TestCase):
    """Test SandboxOperations wrapper methods return Pydantic models."""

    def test_create_sandbox_returns_model(self) -> None:
        """Test create_sandbox returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.create_sandbox.return_value = MockPydanticModel(id="SBX-123")
        ops = SandboxOperations(mock_api)
        result = ops.create_sandbox()
        self.assertIsInstance(result, BaseModel)

    def test_create_sandbox_session_returns_model(self) -> None:
        """Test create_sandbox_session returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.create_sandbox_session.return_value = MockPydanticModel()
        ops = SandboxOperations(mock_api)
        result = ops.create_sandbox_session()
        self.assertIsInstance(result, BaseModel)

    def test_start_sandbox_returns_model(self) -> None:
        """Test start_sandbox returns a Pydantic model."""
        mock_api = MagicMock()
        mock_api.start_sandbox.return_value = MockPydanticModel()
        ops = SandboxOperations(mock_api)
        result = ops.start_sandbox(id="SBX-123")
        self.assertIsInstance(result, BaseModel)


class TestVirshSandboxClient(unittest.TestCase):
    """Test the main VirshSandbox client."""

    @patch("virsh_sandbox.client.ApiClient")
    @patch("virsh_sandbox.client.Configuration")
    def test_client_initialization(
        self, mock_config: MagicMock, mock_api_client: MagicMock
    ) -> None:
        """Test client initializes with default parameters."""
        client = VirshSandbox(host="http://localhost:8080")
        self.assertIsNotNone(client)

    @patch("virsh_sandbox.client.ApiClient")
    @patch("virsh_sandbox.client.Configuration")
    def test_client_properties_return_operations(
        self, mock_config: MagicMock, mock_api_client: MagicMock
    ) -> None:
        """Test client properties return Operation instances."""
        client = VirshSandbox(host="http://localhost:8080")
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

    @patch("virsh_sandbox.client.ApiClient")
    @patch("virsh_sandbox.client.Configuration")
    def test_client_context_manager(
        self, mock_config: MagicMock, mock_api_client: MagicMock
    ) -> None:
        """Test client works as context manager."""
        with VirshSandbox(host="http://localhost:8080") as client:
            self.assertIsNotNone(client)

    @patch("virsh_sandbox.client.ApiClient")
    @patch("virsh_sandbox.client.Configuration")
    def test_client_with_separate_tmux_host(
        self, mock_config: MagicMock, mock_api_client: MagicMock
    ) -> None:
        """Test client with separate tmux host."""
        client = VirshSandbox(
            host="http://localhost:8080", tmux_host="http://localhost:8081"
        )
        self.assertIsNotNone(client)


class TestPydanticModelReturns(unittest.TestCase):
    """Test that Pydantic models provide IDE autocomplete benefits."""

    def test_model_has_fields(self) -> None:
        """Test that returned models have accessible fields."""
        model = MockPydanticModel(id="test-id", name="test-name")
        self.assertEqual(model.id, "test-id")
        self.assertEqual(model.name, "test-name")

    def test_model_dump_converts_to_dict(self) -> None:
        """Test that model_dump() converts to dict when needed."""
        model = MockPydanticModel(id="test-id", name="test-name")
        result = model.model_dump()
        self.assertIsInstance(result, dict)
        self.assertEqual(result["id"], "test-id")
        self.assertEqual(result["name"], "test-name")

    def test_model_has_field_info(self) -> None:
        """Test that Pydantic models expose field information for IDE autocomplete."""
        # model_fields contains field definitions for IDE introspection
        self.assertIn("id", MockPydanticModel.model_fields)
        self.assertIn("name", MockPydanticModel.model_fields)


class TestIntegrationStyleFieldAccess(unittest.TestCase):
    """Integration-style tests that verify Pydantic field access patterns."""

    def test_create_sandbox_field_access(self) -> None:
        """Test create_sandbox returns model with accessible fields."""
        from virsh_sandbox.models.virsh_sandbox_internal_rest_create_sandbox_response import \
            VirshSandboxInternalRestCreateSandboxResponse
        from virsh_sandbox.models.virsh_sandbox_internal_store_sandbox import \
            VirshSandboxInternalStoreSandbox

        # Setup mock
        mock_api = MagicMock()
        mock_sandbox = VirshSandboxInternalStoreSandbox(
            id="SBX-123",
            agent_id="test-agent",
            base_image="ubuntu-base.qcow2",
            sandbox_name="test-vm",
            state="CREATED",
        )
        mock_response = VirshSandboxInternalRestCreateSandboxResponse(
            sandbox=mock_sandbox, ip_address="192.168.122.100"
        )
        mock_api.create_sandbox.return_value = mock_response

        # Execute
        ops = SandboxOperations(mock_api)
        result = ops.create_sandbox(source_vm_name="test-vm")

        # Verify direct field access works as documented
        self.assertIsInstance(result, VirshSandboxInternalRestCreateSandboxResponse)
        self.assertEqual(result.ip_address, "192.168.122.100")

        # Verify nested model access
        self.assertIsNotNone(result.sandbox)
        self.assertEqual(result.sandbox.id, "SBX-123")
        self.assertEqual(result.sandbox.agent_id, "test-agent")
        self.assertEqual(result.sandbox.base_image, "ubuntu-base.qcow2")
        self.assertEqual(result.sandbox.sandbox_name, "test-vm")
        self.assertEqual(result.sandbox.state, "CREATED")

        # Verify model_dump() conversion works
        as_dict = result.model_dump()
        self.assertIsInstance(as_dict, dict)
        self.assertEqual(as_dict["ip_address"], "192.168.122.100")
        self.assertEqual(as_dict["sandbox"]["id"], "SBX-123")

    def test_start_sandbox_field_access(self) -> None:
        """Test start_sandbox returns model with accessible fields."""
        from virsh_sandbox.models.virsh_sandbox_internal_rest_start_sandbox_response import \
            VirshSandboxInternalRestStartSandboxResponse

        # Setup mock
        mock_api = MagicMock()
        mock_response = VirshSandboxInternalRestStartSandboxResponse(
            message="Sandbox started successfully", ip_address="192.168.122.101"
        )
        mock_api.start_sandbox.return_value = mock_response

        # Execute
        ops = SandboxOperations(mock_api)
        result = ops.start_sandbox(id="SBX-123", wait_for_ip=True)

        # Verify field access works
        self.assertIsInstance(result, VirshSandboxInternalRestStartSandboxResponse)
        self.assertEqual(result.message, "Sandbox started successfully")
        self.assertEqual(result.ip_address, "192.168.122.101")

        # Verify model_dump() works
        as_dict = result.model_dump()
        self.assertIsInstance(as_dict, dict)
        self.assertEqual(as_dict["message"], "Sandbox started successfully")
        self.assertEqual(as_dict["ip_address"], "192.168.122.101")

    def test_list_sandboxes_field_access(self) -> None:
        """Test list_sandboxes returns model with list of sandbox models."""
        from virsh_sandbox.models.virsh_sandbox_internal_rest_list_sandboxes_response import \
            VirshSandboxInternalRestListSandboxesResponse
        from virsh_sandbox.models.virsh_sandbox_internal_store_sandbox import \
            VirshSandboxInternalStoreSandbox

        # Setup mock with multiple sandboxes
        mock_api = MagicMock()
        mock_sandboxes = [
            VirshSandboxInternalStoreSandbox(
                id="SBX-001", agent_id="agent-1", state="RUNNING"
            ),
            VirshSandboxInternalStoreSandbox(
                id="SBX-002", agent_id="agent-2", state="STOPPED"
            ),
        ]
        mock_response = VirshSandboxInternalRestListSandboxesResponse(
            sandboxes=mock_sandboxes, total=2, limit=10, offset=0
        )
        mock_api.list_sandboxes.return_value = mock_response

        # Execute
        ops = SandboxOperations(mock_api)
        result = ops.list_sandboxes()

        # Verify field access on response
        self.assertIsInstance(result, VirshSandboxInternalRestListSandboxesResponse)
        self.assertEqual(result.total, 2)
        self.assertEqual(result.limit, 10)
        self.assertEqual(result.offset, 0)

        # Verify list contains proper models
        self.assertIsNotNone(result.sandboxes)
        self.assertEqual(len(result.sandboxes), 2)

        # Verify field access on list items
        self.assertEqual(result.sandboxes[0].id, "SBX-001")
        self.assertEqual(result.sandboxes[0].agent_id, "agent-1")
        self.assertEqual(result.sandboxes[0].state, "RUNNING")

        self.assertEqual(result.sandboxes[1].id, "SBX-002")
        self.assertEqual(result.sandboxes[1].agent_id, "agent-2")
        self.assertEqual(result.sandboxes[1].state, "STOPPED")

        # Verify model_dump() works with nested lists
        as_dict = result.model_dump()
        self.assertIsInstance(as_dict, dict)
        self.assertEqual(as_dict["total"], 2)
        self.assertIsInstance(as_dict["sandboxes"], list)
        self.assertEqual(len(as_dict["sandboxes"]), 2)
        self.assertEqual(as_dict["sandboxes"][0]["id"], "SBX-001")

    def test_get_health_field_access(self) -> None:
        """Test health check returns model with accessible fields."""
        from virsh_sandbox.models.tmux_client_internal_types_health_response import \
            TmuxClientInternalTypesHealthResponse

        # Setup mock
        mock_api = MagicMock()
        mock_response = TmuxClientInternalTypesHealthResponse(
            status="healthy", version="1.0.0"
        )
        mock_api.get_health.return_value = mock_response

        # Execute
        ops = HealthOperations(mock_api)
        result = ops.get_health()

        # Verify field access works
        self.assertIsInstance(result, TmuxClientInternalTypesHealthResponse)
        self.assertEqual(result.status, "healthy")
        self.assertEqual(result.version, "1.0.0")

        # Verify model_dump() works
        as_dict = result.model_dump()
        self.assertIsInstance(as_dict, dict)
        self.assertEqual(as_dict["status"], "healthy")

    def test_run_command_field_access(self) -> None:
        """Test run_command returns model with accessible fields."""
        from virsh_sandbox.models.tmux_client_internal_types_run_command_response import \
            TmuxClientInternalTypesRunCommandResponse

        # Setup mock
        mock_api = MagicMock()
        mock_response = TmuxClientInternalTypesRunCommandResponse(
            stdout="command output", stderr="", exit_code=0, duration_ms=150
        )
        mock_api.run_command.return_value = mock_response

        # Execute
        ops = CommandOperations(mock_api)
        result = ops.run_command(command="ls", args=["-la"])

        # Verify field access works
        self.assertIsInstance(result, TmuxClientInternalTypesRunCommandResponse)
        self.assertEqual(result.stdout, "command output")
        self.assertEqual(result.stderr, "")
        self.assertEqual(result.exit_code, 0)
        self.assertEqual(result.duration_ms, 150)

        # Verify model_dump() works
        as_dict = result.model_dump()
        self.assertIsInstance(as_dict, dict)
        self.assertEqual(as_dict["stdout"], "command output")
        self.assertEqual(as_dict["exit_code"], 0)

    def test_read_file_field_access(self) -> None:
        """Test read_file returns model with accessible fields."""
        from virsh_sandbox.models.tmux_client_internal_types_read_file_response import \
            TmuxClientInternalTypesReadFileResponse

        # Setup mock
        mock_api = MagicMock()
        mock_response = TmuxClientInternalTypesReadFileResponse(
            content="file contents here",
            lines=1,
            truncated=False,
            size_bytes=18,
        )
        mock_api.read_file.return_value = mock_response

        # Execute
        ops = FileOperations(mock_api)
        result = ops.read_file(path="/tmp/test.txt")

        # Verify field access works
        self.assertIsInstance(result, TmuxClientInternalTypesReadFileResponse)
        self.assertEqual(result.content, "file contents here")
        self.assertEqual(result.lines, 1)
        self.assertEqual(result.truncated, False)
        self.assertEqual(result.size_bytes, 18)

        # Verify model_dump() works
        as_dict = result.model_dump()
        self.assertIsInstance(as_dict, dict)
        self.assertEqual(as_dict["content"], "file contents here")
        self.assertEqual(as_dict["lines"], 1)

    def test_create_sandbox_session_field_access(self) -> None:
        """Test create_sandbox_session returns model with nested fields."""
        from virsh_sandbox.models.internal_api_create_sandbox_session_response import \
            InternalApiCreateSandboxSessionResponse

        # Setup mock
        mock_api = MagicMock()
        mock_response = InternalApiCreateSandboxSessionResponse(
            session_name="session-abc123",
            certificate_id="cert-xyz789",
            tmux_url="http://tmux:8081",
        )
        mock_api.create_sandbox_session.return_value = mock_response

        # Execute
        ops = SandboxOperations(mock_api)
        result = ops.create_sandbox_session(
            sandbox_id="SBX-123", session_name="test-session"
        )

        # Verify field access works
        self.assertIsInstance(result, InternalApiCreateSandboxSessionResponse)
        self.assertEqual(result.session_name, "session-abc123")
        self.assertEqual(result.certificate_id, "cert-xyz789")
        self.assertEqual(result.tmux_url, "http://tmux:8081")

        # Verify model_dump() works
        as_dict = result.model_dump()
        self.assertIsInstance(as_dict, dict)
        self.assertEqual(as_dict["session_name"], "session-abc123")
        self.assertEqual(as_dict["certificate_id"], "cert-xyz789")

    def test_list_tmux_sessions_list_return_type(self) -> None:
        """Test list_tmux_sessions returns list of Pydantic models."""
        from virsh_sandbox.models.tmux_client_internal_types_session_info import \
            TmuxClientInternalTypesSessionInfo

        # Setup mock
        mock_api = MagicMock()
        mock_sessions = [
            TmuxClientInternalTypesSessionInfo(
                name="session-1", windows=2, attached=True
            ),
            TmuxClientInternalTypesSessionInfo(
                name="session-2", windows=1, attached=False
            ),
        ]
        mock_api.list_tmux_sessions.return_value = mock_sessions

        # Execute
        ops = TmuxOperations(mock_api)
        result = ops.list_tmux_sessions()

        # Verify list return type
        self.assertIsInstance(result, list)
        self.assertEqual(len(result), 2)

        # Verify each item is a Pydantic model with accessible fields
        self.assertIsInstance(result[0], TmuxClientInternalTypesSessionInfo)
        self.assertEqual(result[0].name, "session-1")
        self.assertEqual(result[0].windows, 2)
        self.assertEqual(result[0].attached, True)

        self.assertIsInstance(result[1], TmuxClientInternalTypesSessionInfo)
        self.assertEqual(result[1].name, "session-2")
        self.assertEqual(result[1].windows, 1)
        self.assertEqual(result[1].attached, False)

        # Verify model_dump() works on list items
        item_dict = result[0].model_dump()
        self.assertIsInstance(item_dict, dict)
        self.assertEqual(item_dict["name"], "session-1")


if __name__ == "__main__":
    unittest.main()
