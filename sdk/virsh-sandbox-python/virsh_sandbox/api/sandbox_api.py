# coding: utf-8

"""
    virsh-sandbox API
    API for managing virtual machine sandboxes using libvirt
"""

from typing import Optional, Dict, Any, List
import asyncio

from virsh_sandbox.api_client import ApiClient
from virsh_sandbox.exceptions import ApiException
{import=from pydantic import Field, StrictStr}
{import=from typing import Optional}
{import=from typing_extensions import Annotated}
{import=from virsh_sandbox.models.internal_rest_create_sandbox_request import InternalRestCreateSandboxRequest}
{import=from virsh_sandbox.models.internal_rest_create_sandbox_response import InternalRestCreateSandboxResponse}
{import=from virsh_sandbox.models.internal_rest_diff_request import InternalRestDiffRequest}
{import=from virsh_sandbox.models.internal_rest_diff_response import InternalRestDiffResponse}
{import=from virsh_sandbox.models.internal_rest_inject_ssh_key_request import InternalRestInjectSSHKeyRequest}
{import=from virsh_sandbox.models.internal_rest_publish_request import InternalRestPublishRequest}
{import=from virsh_sandbox.models.internal_rest_run_command_request import InternalRestRunCommandRequest}
{import=from virsh_sandbox.models.internal_rest_run_command_response import InternalRestRunCommandResponse}
{import=from virsh_sandbox.models.internal_rest_snapshot_request import InternalRestSnapshotRequest}
{import=from virsh_sandbox.models.internal_rest_snapshot_response import InternalRestSnapshotResponse}
{import=from virsh_sandbox.models.internal_rest_start_sandbox_request import InternalRestStartSandboxRequest}
{import=from virsh_sandbox.models.internal_rest_start_sandbox_response import InternalRestStartSandboxResponse}

class SandboxApi:
    """SandboxApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def create_sandbox(
        self,
        request: InternalRestCreateSandboxRequest,
        **kwargs
    ) -> InternalRestCreateSandboxResponse:
        """Create a new sandbox

        Creates a new virtual machine sandbox by cloning from an existing VM

        Args:
            request: Sandbox creation parameters

        Returns:
            InternalRestCreateSandboxResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.create_sandbox_with_http_info(request=request, **kwargs)[0]

    def create_snapshot(
        self,
        id: str,
        request: InternalRestSnapshotRequest,
        **kwargs
    ) -> InternalRestSnapshotResponse:
        """Create snapshot

        Creates a snapshot of the sandbox

        Args:
            id: Sandbox ID
            request: Snapshot parameters

        Returns:
            InternalRestSnapshotResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.create_snapshot_with_http_info(id=id, request=request, **kwargs)[0]

    def destroy_sandbox(
        self,
        id: str,
        **kwargs
    ) -> None:
        """Destroy sandbox

        Destroys the sandbox and cleans up resources

        Args:
            id: Sandbox ID

        Returns:
            None

        Raises:
            ApiException: If the API call fails
        """
        return self.destroy_sandbox_with_http_info(id=id, **kwargs)[0]

    def diff_snapshots(
        self,
        id: str,
        request: InternalRestDiffRequest,
        **kwargs
    ) -> InternalRestDiffResponse:
        """Diff snapshots

        Computes differences between two snapshots

        Args:
            id: Sandbox ID
            request: Diff parameters

        Returns:
            InternalRestDiffResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.diff_snapshots_with_http_info(id=id, request=request, **kwargs)[0]

    def generate_configuration(
        self,
        id: str,
        tool: str,
        **kwargs
    ) -> None:
        """Generate configuration

        Generates Ansible or Puppet configuration from sandbox changes

        Args:
            id: Sandbox ID
            tool: Tool type (ansible or puppet)

        Returns:
            None

        Raises:
            ApiException: If the API call fails
        """
        return self.generate_configuration_with_http_info(id=id, tool=tool, **kwargs)[0]

    def inject_ssh_key(
        self,
        id: str,
        request: InternalRestInjectSSHKeyRequest,
        **kwargs
    ) -> None:
        """Inject SSH key into sandbox

        Injects a public SSH key for a user in the sandbox

        Args:
            id: Sandbox ID
            request: SSH key injection parameters

        Returns:
            None

        Raises:
            ApiException: If the API call fails
        """
        return self.inject_ssh_key_with_http_info(id=id, request=request, **kwargs)[0]

    def publish_changes(
        self,
        id: str,
        request: InternalRestPublishRequest,
        **kwargs
    ) -> None:
        """Publish changes

        Publishes sandbox changes to GitOps repository

        Args:
            id: Sandbox ID
            request: Publish parameters

        Returns:
            None

        Raises:
            ApiException: If the API call fails
        """
        return self.publish_changes_with_http_info(id=id, request=request, **kwargs)[0]

    def run_sandbox_command(
        self,
        id: str,
        request: InternalRestRunCommandRequest,
        **kwargs
    ) -> InternalRestRunCommandResponse:
        """Run command in sandbox

        Executes a command inside the sandbox via SSH

        Args:
            id: Sandbox ID
            request: Command execution parameters

        Returns:
            InternalRestRunCommandResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.run_sandbox_command_with_http_info(id=id, request=request, **kwargs)[0]

    def start_sandbox(
        self,
        id: str,
        request: Optional[InternalRestStartSandboxRequest] = None,
        **kwargs
    ) -> InternalRestStartSandboxResponse:
        """Start sandbox

        Starts the virtual machine sandbox

        Args:
            id: Sandbox ID
            request: Start parameters

        Returns:
            InternalRestStartSandboxResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.start_sandbox_with_http_info(id=id, request=request, **kwargs)[0]

