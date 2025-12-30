# coding: utf-8

"""
    virsh-sandbox API
    API for managing virtual machine sandboxes using libvirt
"""

from typing import Optional, Dict, Any, List
import asyncio

from virsh_sandbox.api_client import ApiClient
from virsh_sandbox.exceptions import ApiException
{import=from virsh_sandbox.models.internal_rest_list_vms_response import InternalRestListVMsResponse}

class VMsApi:
    """VMsApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def list_virtual_machines(
        self,
        **kwargs
    ) -> InternalRestListVMsResponse:
        """List all VMs

        Returns a list of all virtual machines from the libvirt instance

        Args:

        Returns:
            InternalRestListVMsResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.list_virtual_machines_with_http_info(**kwargs)[0]

