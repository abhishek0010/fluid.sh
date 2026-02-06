# coding: utf-8

from __future__ import annotations

"""
Unified Fluid Client

This module provides a unified client wrapper for the Fluid SDK,
offering a cleaner interface with flattened parameters instead of request objects.

Example:
    from fluid import Fluid

    client = Fluid(host="http://localhost:8080")
    # Create a sandbox with simple parameters
    client.sandbox.create_sandbox(source_vm_name="ubuntu-base")
    # Run a command
    client.command.run_command(command="ls", args=["-la"])
"""

from typing import Dict, List, Optional, Tuple, Union

from fluid.api.access_api import AccessApi
from fluid.api.ansible_api import AnsibleApi
from fluid.api.ansible_playbooks_api import AnsiblePlaybooksApi
from fluid.api.health_api import HealthApi
from fluid.api.sandbox_api import SandboxApi
from fluid.api.vms_api import VMsApi
from fluid.api_client import ApiClient
from fluid.configuration import Configuration
from fluid.models.fluid_remote_internal_ansible_job import \
    FluidRemoteInternalAnsibleJob
from fluid.models.fluid_remote_internal_ansible_job_request import \
    FluidRemoteInternalAnsibleJobRequest
from fluid.models.fluid_remote_internal_ansible_job_response import \
    FluidRemoteInternalAnsibleJobResponse
from fluid.models.fluid_remote_internal_rest_create_sandbox_request import \
    FluidRemoteInternalRestCreateSandboxRequest
from fluid.models.fluid_remote_internal_rest_create_sandbox_response import \
    FluidRemoteInternalRestCreateSandboxResponse
from fluid.models.fluid_remote_internal_rest_destroy_sandbox_response import \
    FluidRemoteInternalRestDestroySandboxResponse
from fluid.models.fluid_remote_internal_rest_diff_request import \
    FluidRemoteInternalRestDiffRequest
from fluid.models.fluid_remote_internal_rest_diff_response import \
    FluidRemoteInternalRestDiffResponse
from fluid.models.fluid_remote_internal_rest_discover_ip_response import \
    FluidRemoteInternalRestDiscoverIPResponse
from fluid.models.fluid_remote_internal_rest_get_sandbox_response import \
    FluidRemoteInternalRestGetSandboxResponse
from fluid.models.fluid_remote_internal_rest_health_response import \
    FluidRemoteInternalRestHealthResponse
from fluid.models.fluid_remote_internal_rest_inject_ssh_key_request import \
    FluidRemoteInternalRestInjectSSHKeyRequest
from fluid.models.fluid_remote_internal_rest_list_sandbox_commands_response import \
    FluidRemoteInternalRestListSandboxCommandsResponse
from fluid.models.fluid_remote_internal_rest_list_sandboxes_response import \
    FluidRemoteInternalRestListSandboxesResponse
from fluid.models.fluid_remote_internal_rest_list_vms_response import \
    FluidRemoteInternalRestListVMsResponse
from fluid.models.fluid_remote_internal_rest_publish_request import \
    FluidRemoteInternalRestPublishRequest
from fluid.models.fluid_remote_internal_rest_run_command_request import \
    FluidRemoteInternalRestRunCommandRequest
from fluid.models.fluid_remote_internal_rest_run_command_response import \
    FluidRemoteInternalRestRunCommandResponse
from fluid.models.fluid_remote_internal_rest_snapshot_request import \
    FluidRemoteInternalRestSnapshotRequest
from fluid.models.fluid_remote_internal_rest_snapshot_response import \
    FluidRemoteInternalRestSnapshotResponse
from fluid.models.fluid_remote_internal_rest_start_sandbox_request import \
    FluidRemoteInternalRestStartSandboxRequest
from fluid.models.fluid_remote_internal_rest_start_sandbox_response import \
    FluidRemoteInternalRestStartSandboxResponse
from fluid.models.internal_ansible_add_task_request import \
    InternalAnsibleAddTaskRequest
from fluid.models.internal_ansible_add_task_response import \
    InternalAnsibleAddTaskResponse
from fluid.models.internal_ansible_create_playbook_request import \
    InternalAnsibleCreatePlaybookRequest
from fluid.models.internal_ansible_create_playbook_response import \
    InternalAnsibleCreatePlaybookResponse
from fluid.models.internal_ansible_export_playbook_response import \
    InternalAnsibleExportPlaybookResponse
from fluid.models.internal_ansible_get_playbook_response import \
    InternalAnsibleGetPlaybookResponse
from fluid.models.internal_ansible_list_playbooks_response import \
    InternalAnsibleListPlaybooksResponse
from fluid.models.internal_ansible_reorder_tasks_request import \
    InternalAnsibleReorderTasksRequest
from fluid.models.internal_ansible_update_task_request import \
    InternalAnsibleUpdateTaskRequest
from fluid.models.internal_ansible_update_task_response import \
    InternalAnsibleUpdateTaskResponse
from fluid.models.internal_rest_ca_public_key_response import \
    InternalRestCaPublicKeyResponse
from fluid.models.internal_rest_certificate_response import \
    InternalRestCertificateResponse
from fluid.models.internal_rest_list_certificates_response import \
    InternalRestListCertificatesResponse
from fluid.models.internal_rest_list_sessions_response import \
    InternalRestListSessionsResponse
from fluid.models.internal_rest_request_access_request import \
    InternalRestRequestAccessRequest
from fluid.models.internal_rest_request_access_response import \
    InternalRestRequestAccessResponse
from fluid.models.internal_rest_revoke_certificate_request import \
    InternalRestRevokeCertificateRequest
from fluid.models.internal_rest_revoke_certificate_response import \
    InternalRestRevokeCertificateResponse
from fluid.models.internal_rest_session_end_request import \
    InternalRestSessionEndRequest
from fluid.models.internal_rest_session_end_response import \
    InternalRestSessionEndResponse
from fluid.models.internal_rest_session_start_request import \
    InternalRestSessionStartRequest
from fluid.models.internal_rest_session_start_response import \
    InternalRestSessionStartResponse


class AccessOperations:
    """Wrapper for AccessApi with simplified method signatures."""

    def __init__(self, api: AccessApi):
        self._api = api

    def get_ca_public_key(self) -> InternalRestCaPublicKeyResponse:
        """Get the SSH CA public key

        Returns:
            InternalRestCaPublicKeyResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_ca_public_key()

    def get_certificate(
        self,
        cert_id: str,
    ) -> InternalRestCertificateResponse:
        """Get certificate details

        Args:
            cert_id: str

        Returns:
            InternalRestCertificateResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_certificate(cert_id=cert_id)

    def list_certificates(
        self,
        sandbox_id: Optional[str] = None,
        user_id: Optional[str] = None,
        status: Optional[str] = None,
        active_only: Optional[bool] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> InternalRestListCertificatesResponse:
        """List certificates

        Args:
            sandbox_id: Optional[str]
            user_id: Optional[str]
            status: Optional[str]
            active_only: Optional[bool]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            InternalRestListCertificatesResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_certificates(
            sandbox_id=sandbox_id,
            user_id=user_id,
            status=status,
            active_only=active_only,
            limit=limit,
            offset=offset,
        )

    def list_sessions(
        self,
        sandbox_id: Optional[str] = None,
        certificate_id: Optional[str] = None,
        user_id: Optional[str] = None,
        active_only: Optional[bool] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> InternalRestListSessionsResponse:
        """List sessions

        Args:
            sandbox_id: Optional[str]
            certificate_id: Optional[str]
            user_id: Optional[str]
            active_only: Optional[bool]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            InternalRestListSessionsResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_sessions(
            sandbox_id=sandbox_id,
            certificate_id=certificate_id,
            user_id=user_id,
            active_only=active_only,
            limit=limit,
            offset=offset,
        )

    def record_session_end(
        self,
        reason: Optional[str] = None,
        session_id: Optional[str] = None,
    ) -> InternalRestSessionEndResponse:
        """Record session end

        Args:
            reason: reason
            session_id: session_id

        Returns:
            InternalRestSessionEndResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestSessionEndRequest(
            reason=reason,
            session_id=session_id,
        )
        return self._api.record_session_end(request=request)

    def record_session_start(
        self,
        certificate_id: Optional[str] = None,
        source_ip: Optional[str] = None,
    ) -> InternalRestSessionStartResponse:
        """Record session start

        Args:
            certificate_id: certificate_id
            source_ip: source_ip

        Returns:
            InternalRestSessionStartResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestSessionStartRequest(
            certificate_id=certificate_id,
            source_ip=source_ip,
        )
        return self._api.record_session_start(request=request)

    def request_access(
        self,
        public_key: Optional[str] = None,
        sandbox_id: Optional[str] = None,
        ttl_minutes: Optional[int] = None,
        user_id: Optional[str] = None,
    ) -> InternalRestRequestAccessResponse:
        """Request SSH access to a sandbox

        Args:
            public_key: PublicKey is the user
            sandbox_id: SandboxID is the target sandbox.
            ttl_minutes: TTLMinutes is the requested access duration (1-10 minutes).
            user_id: UserID identifies the requesting user.

        Returns:
            InternalRestRequestAccessResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestRequestAccessRequest(
            public_key=public_key,
            sandbox_id=sandbox_id,
            ttl_minutes=ttl_minutes,
            user_id=user_id,
        )
        return self._api.request_access(request=request)

    def revoke_certificate(
        self,
        cert_id: str,
        reason: Optional[str] = None,
    ) -> InternalRestRevokeCertificateResponse:
        """Revoke a certificate

        Args:
            cert_id: str
            reason: reason

        Returns:
            InternalRestRevokeCertificateResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestRevokeCertificateRequest(
            reason=reason,
        )
        return self._api.revoke_certificate(cert_id=cert_id, request=request)


class AnsibleOperations:
    """Wrapper for AnsibleApi with simplified method signatures."""

    def __init__(self, api: AnsibleApi):
        self._api = api

    def create_ansible_job(
        self,
        check: Optional[bool] = None,
        playbook: Optional[str] = None,
        vm_name: Optional[str] = None,
    ) -> FluidRemoteInternalAnsibleJobResponse:
        """Create Ansible job

        Args:
            check: check
            playbook: playbook
            vm_name: vm_name

        Returns:
            FluidRemoteInternalAnsibleJobResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = FluidRemoteInternalAnsibleJobRequest(
            check=check,
            playbook=playbook,
            vm_name=vm_name,
        )
        return self._api.create_ansible_job(request=request)

    def get_ansible_job(
        self,
        job_id: str,
    ) -> FluidRemoteInternalAnsibleJob:
        """Get Ansible job

        Args:
            job_id: str

        Returns:
            FluidRemoteInternalAnsibleJob: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_ansible_job(job_id=job_id)

    def stream_ansible_job_output(
        self,
        job_id: str,
    ) -> None:
        """Stream Ansible job output

        Args:
            job_id: str
        """
        return self._api.stream_ansible_job_output(job_id=job_id)


class AnsiblePlaybooksOperations:
    """Wrapper for AnsiblePlaybooksApi with simplified method signatures."""

    def __init__(self, api: AnsiblePlaybooksApi):
        self._api = api

    def add_playbook_task(
        self,
        playbook_name: str,
        module: Optional[str] = None,
        name: Optional[str] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> InternalAnsibleAddTaskResponse:
        """Add task to playbook

        Args:
            playbook_name: str
            module: module
            name: name
            params: params

        Returns:
            InternalAnsibleAddTaskResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalAnsibleAddTaskRequest(
            module=module,
            name=name,
            params=params,
        )
        return self._api.add_playbook_task(playbook_name=playbook_name, request=request)

    def create_playbook(
        self,
        become: Optional[bool] = None,
        hosts: Optional[str] = None,
        name: Optional[str] = None,
    ) -> InternalAnsibleCreatePlaybookResponse:
        """Create playbook

        Args:
            become: become
            hosts: hosts
            name: name

        Returns:
            InternalAnsibleCreatePlaybookResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalAnsibleCreatePlaybookRequest(
            become=become,
            hosts=hosts,
            name=name,
        )
        return self._api.create_playbook(request=request)

    def delete_playbook(
        self,
        playbook_name: str,
    ) -> None:
        """Delete playbook

        Args:
            playbook_name: str
        """
        return self._api.delete_playbook(playbook_name=playbook_name)

    def delete_playbook_task(
        self,
        playbook_name: str,
        task_id: str,
    ) -> None:
        """Delete task

        Args:
            playbook_name: str
            task_id: str
        """
        return self._api.delete_playbook_task(
            playbook_name=playbook_name, task_id=task_id
        )

    def export_playbook(
        self,
        playbook_name: str,
    ) -> InternalAnsibleExportPlaybookResponse:
        """Export playbook

        Args:
            playbook_name: str

        Returns:
            InternalAnsibleExportPlaybookResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.export_playbook(playbook_name=playbook_name)

    def get_playbook(
        self,
        playbook_name: str,
    ) -> InternalAnsibleGetPlaybookResponse:
        """Get playbook

        Args:
            playbook_name: str

        Returns:
            InternalAnsibleGetPlaybookResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_playbook(playbook_name=playbook_name)

    def list_playbooks(self) -> InternalAnsibleListPlaybooksResponse:
        """List playbooks

        Returns:
            InternalAnsibleListPlaybooksResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_playbooks()

    def reorder_playbook_tasks(
        self,
        playbook_name: str,
        task_ids: Optional[List[str]] = None,
    ) -> None:
        """Reorder tasks

        Args:
            playbook_name: str
            task_ids: task_ids
        """
        request = InternalAnsibleReorderTasksRequest(
            task_ids=task_ids,
        )
        return self._api.reorder_playbook_tasks(
            playbook_name=playbook_name, request=request
        )

    def update_playbook_task(
        self,
        playbook_name: str,
        task_id: str,
        module: Optional[str] = None,
        name: Optional[str] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> InternalAnsibleUpdateTaskResponse:
        """Update task

        Args:
            playbook_name: str
            task_id: str
            module: module
            name: name
            params: params

        Returns:
            InternalAnsibleUpdateTaskResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalAnsibleUpdateTaskRequest(
            module=module,
            name=name,
            params=params,
        )
        return self._api.update_playbook_task(
            playbook_name=playbook_name, task_id=task_id, request=request
        )


class HealthOperations:
    """Wrapper for HealthApi with simplified method signatures."""

    def __init__(self, api: HealthApi):
        self._api = api

    def get_health(self) -> FluidRemoteInternalRestHealthResponse:
        """Health check

        Returns:
            FluidRemoteInternalRestHealthResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_health()


class SandboxOperations:
    """Wrapper for SandboxApi with simplified method signatures."""

    def __init__(self, api: SandboxApi):
        self._api = api

    def create_sandbox(
        self,
        agent_id: Optional[str] = None,
        auto_start: Optional[bool] = None,
        cpu: Optional[int] = None,
        memory_mb: Optional[int] = None,
        source_vm_name: Optional[str] = None,
        ttl_seconds: Optional[int] = None,
        vm_name: Optional[str] = None,
        wait_for_ip: Optional[bool] = None,
        request_timeout: Union[None, float, Tuple[float, float]] = None,
    ) -> FluidRemoteInternalRestCreateSandboxResponse:
        """Create a new sandbox

        Args:
            agent_id: required
            auto_start: optional; if true, start the VM immediately after creation
            cpu: optional; default from service config if <=0
            memory_mb: optional; default from service config if <=0
            source_vm_name: required; name of existing VM in libvirt to clone from
            ttl_seconds: optional; TTL for auto garbage collection
            vm_name: optional; generated if empty
            wait_for_ip: optional; if true and auto_start, wait for IP discovery. When True, consider setting request_timeout to accommodate IP discovery (server default is 120s)
            request_timeout: HTTP request timeout in seconds. Can be a single float for total timeout, or a tuple (connect_timeout, read_timeout). For operations with wait_for_ip=True, set this to at least 180 seconds.

        Returns:
            FluidRemoteInternalRestCreateSandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = FluidRemoteInternalRestCreateSandboxRequest(
            agent_id=agent_id,
            auto_start=auto_start,
            cpu=cpu,
            memory_mb=memory_mb,
            source_vm_name=source_vm_name,
            ttl_seconds=ttl_seconds,
            vm_name=vm_name,
            wait_for_ip=wait_for_ip,
        )
        return self._api.create_sandbox(
            request=request, _request_timeout=request_timeout
        )

    def create_snapshot(
        self,
        id: str,
        external: Optional[bool] = None,
        name: Optional[str] = None,
    ) -> FluidRemoteInternalRestSnapshotResponse:
        """Create snapshot

        Args:
            id: str
            external: optional; default false (internal snapshot)
            name: required

        Returns:
            FluidRemoteInternalRestSnapshotResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = FluidRemoteInternalRestSnapshotRequest(
            external=external,
            name=name,
        )
        return self._api.create_snapshot(id=id, request=request)

    def destroy_sandbox(
        self,
        id: str,
    ) -> FluidRemoteInternalRestDestroySandboxResponse:
        """Destroy sandbox

        Args:
            id: str

        Returns:
            FluidRemoteInternalRestDestroySandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.destroy_sandbox(id=id)

    def diff_snapshots(
        self,
        id: str,
        from_snapshot: Optional[str] = None,
        to_snapshot: Optional[str] = None,
    ) -> FluidRemoteInternalRestDiffResponse:
        """Diff snapshots

        Args:
            id: str
            from_snapshot: required
            to_snapshot: required

        Returns:
            FluidRemoteInternalRestDiffResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = FluidRemoteInternalRestDiffRequest(
            from_snapshot=from_snapshot,
            to_snapshot=to_snapshot,
        )
        return self._api.diff_snapshots(id=id, request=request)

    def discover_sandbox_ip(
        self,
        id: str,
    ) -> FluidRemoteInternalRestDiscoverIPResponse:
        """Discover sandbox IP

        Args:
            id: str

        Returns:
            FluidRemoteInternalRestDiscoverIPResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.discover_sandbox_ip(id=id)

    def generate_configuration(
        self,
        id: str,
        tool: str,
    ) -> None:
        """Generate configuration

        Args:
            id: str
            tool: str
        """
        return self._api.generate_configuration(id=id, tool=tool)

    def get_sandbox(
        self,
        id: str,
        include_commands: Optional[bool] = None,
    ) -> FluidRemoteInternalRestGetSandboxResponse:
        """Get sandbox details

        Args:
            id: str
            include_commands: Optional[bool]

        Returns:
            FluidRemoteInternalRestGetSandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_sandbox(id=id, include_commands=include_commands)

    def inject_ssh_key(
        self,
        id: str,
        public_key: Optional[str] = None,
        username: Optional[str] = None,
    ) -> None:
        """Inject SSH key into sandbox

        Args:
            id: str
            public_key: required
            username: required (explicit); typical: \
        """
        request = FluidRemoteInternalRestInjectSSHKeyRequest(
            public_key=public_key,
            username=username,
        )
        return self._api.inject_ssh_key(id=id, request=request)

    def list_sandbox_commands(
        self,
        id: str,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> FluidRemoteInternalRestListSandboxCommandsResponse:
        """List sandbox commands

        Args:
            id: str
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            FluidRemoteInternalRestListSandboxCommandsResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_sandbox_commands(id=id, limit=limit, offset=offset)

    def list_sandboxes(
        self,
        agent_id: Optional[str] = None,
        job_id: Optional[str] = None,
        base_image: Optional[str] = None,
        state: Optional[str] = None,
        vm_name: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> FluidRemoteInternalRestListSandboxesResponse:
        """List sandboxes

        Args:
            agent_id: Optional[str]
            job_id: Optional[str]
            base_image: Optional[str]
            state: Optional[str]
            vm_name: Optional[str]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            FluidRemoteInternalRestListSandboxesResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_sandboxes(
            agent_id=agent_id,
            job_id=job_id,
            base_image=base_image,
            state=state,
            vm_name=vm_name,
            limit=limit,
            offset=offset,
        )

    def publish_changes(
        self,
        id: str,
        job_id: Optional[str] = None,
        message: Optional[str] = None,
        reviewers: Optional[List[str]] = None,
    ) -> None:
        """Publish changes

        Args:
            id: str
            job_id: required
            message: optional commit/PR message
            reviewers: optional
        """
        request = FluidRemoteInternalRestPublishRequest(
            job_id=job_id,
            message=message,
            reviewers=reviewers,
        )
        return self._api.publish_changes(id=id, request=request)

    def run_sandbox_command(
        self,
        id: str,
        command: Optional[str] = None,
        env: Optional[Dict[str, str]] = None,
        private_key_path: Optional[str] = None,
        timeout_sec: Optional[int] = None,
        user: Optional[str] = None,
        request_timeout: Union[None, float, Tuple[float, float]] = None,
    ) -> FluidRemoteInternalRestRunCommandResponse:
        """Run command in sandbox

        Args:
            id: str
            command: required
            env: optional
            private_key_path: optional; if empty, uses managed credentials (requires SSH CA)
            timeout_sec: optional; default from service config
            user: optional; defaults to \
            request_timeout: HTTP request timeout in seconds. Can be a single float for total timeout, or a tuple (connect_timeout, read_timeout). For operations with wait_for_ip=True, set this to at least 180 seconds.

        Returns:
            FluidRemoteInternalRestRunCommandResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = FluidRemoteInternalRestRunCommandRequest(
            command=command,
            env=env,
            private_key_path=private_key_path,
            timeout_sec=timeout_sec,
            user=user,
        )
        return self._api.run_sandbox_command(
            id=id, request=request, _request_timeout=request_timeout
        )

    def start_sandbox(
        self,
        id: str,
        wait_for_ip: Optional[bool] = None,
        request_timeout: Union[None, float, Tuple[float, float]] = None,
    ) -> FluidRemoteInternalRestStartSandboxResponse:
        """Start sandbox

        Args:
            id: str
            wait_for_ip: optional; default false. When True, consider setting request_timeout to accommodate IP discovery (server default is 120s)
            request_timeout: HTTP request timeout in seconds. Can be a single float for total timeout, or a tuple (connect_timeout, read_timeout). For operations with wait_for_ip=True, set this to at least 180 seconds.

        Returns:
            FluidRemoteInternalRestStartSandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = FluidRemoteInternalRestStartSandboxRequest(
            wait_for_ip=wait_for_ip,
        )
        return self._api.start_sandbox(
            id=id, request=request, _request_timeout=request_timeout
        )

    def stream_sandbox_activity(
        self,
        id: str,
    ) -> None:
        """Stream sandbox activity

        Args:
            id: str
        """
        return self._api.stream_sandbox_activity(id=id)


class VMsOperations:
    """Wrapper for VMsApi with simplified method signatures."""

    def __init__(self, api: VMsApi):
        self._api = api

    def list_virtual_machines(self) -> FluidRemoteInternalRestListVMsResponse:
        """List all VMs

        Returns:
            FluidRemoteInternalRestListVMsResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_virtual_machines()


class Fluid:
    """Unified client for the Fluid API.

    This class provides a single entry point for all Fluid API operations.
    All methods use flattened parameters instead of request objects.

    Args:
        host: Base URL for the main Fluid API
        api_key: Optional API key for authentication
        verify_ssl: Whether to verify SSL certificates

    Example:
        >>> from fluid import Fluid
        >>> client = Fluid()
        >>> client.sandbox.create_sandbox(source_vm_name="base-vm")
    """

    def __init__(
        self,
        host: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        access_token: Optional[str] = None,
        username: Optional[str] = None,
        password: Optional[str] = None,
        verify_ssl: bool = True,
        ssl_ca_cert: Optional[str] = None,
        retries: Optional[int] = None,
    ) -> None:
        """Initialize the Fluid client."""
        self._main_config = Configuration(
            host=host,
            api_key={"Authorization": api_key} if api_key else None,
            access_token=access_token,
            username=username,
            password=password,
            ssl_ca_cert=ssl_ca_cert,
            retries=retries,
        )
        self._main_config.verify_ssl = verify_ssl
        self._main_api_client = ApiClient(configuration=self._main_config)

        self._access: Optional[AccessOperations] = None
        self._ansible: Optional[AnsibleOperations] = None
        self._ansible_playbooks: Optional[AnsiblePlaybooksOperations] = None
        self._health: Optional[HealthOperations] = None
        self._sandbox: Optional[SandboxOperations] = None
        self._vms: Optional[VMsOperations] = None

    @property
    def access(self) -> AccessOperations:
        """Access AccessApi operations."""
        if self._access is None:
            api = AccessApi(api_client=self._main_api_client)
            self._access = AccessOperations(api)
        return self._access

    @property
    def ansible(self) -> AnsibleOperations:
        """Access AnsibleApi operations."""
        if self._ansible is None:
            api = AnsibleApi(api_client=self._main_api_client)
            self._ansible = AnsibleOperations(api)
        return self._ansible

    @property
    def ansible_playbooks(self) -> AnsiblePlaybooksOperations:
        """Access AnsiblePlaybooksApi operations."""
        if self._ansible_playbooks is None:
            api = AnsiblePlaybooksApi(api_client=self._main_api_client)
            self._ansible_playbooks = AnsiblePlaybooksOperations(api)
        return self._ansible_playbooks

    @property
    def health(self) -> HealthOperations:
        """Access HealthApi operations."""
        if self._health is None:
            api = HealthApi(api_client=self._main_api_client)
            self._health = HealthOperations(api)
        return self._health

    @property
    def sandbox(self) -> SandboxOperations:
        """Access SandboxApi operations."""
        if self._sandbox is None:
            api = SandboxApi(api_client=self._main_api_client)
            self._sandbox = SandboxOperations(api)
        return self._sandbox

    @property
    def vms(self) -> VMsOperations:
        """Access VMsApi operations."""
        if self._vms is None:
            api = VMsApi(api_client=self._main_api_client)
            self._vms = VMsOperations(api)
        return self._vms

    @property
    def configuration(self) -> Configuration:
        """Get the main API configuration."""
        return self._main_config

    def set_debug(self, debug: bool) -> None:
        """Enable or disable debug mode."""
        self._main_config.debug = debug

    def close(self) -> None:
        """Close the API client connections."""
        if hasattr(self._main_api_client.rest_client, "close"):
            self._main_api_client.rest_client.close()

    def __enter__(self) -> "Fluid":
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        self.close()
