# coding: utf-8

"""
    virsh-sandbox API
    API for managing virtual machine sandboxes using libvirt
"""

from typing import Optional, Dict, Any, List
import asyncio

from virsh_sandbox.api_client import ApiClient
from virsh_sandbox.exceptions import ApiException
{import=from typing import Any, Dict}
{import=from virsh_sandbox.models.tmux_client_internal_types_health_response import TmuxClientInternalTypesHealthResponse}

class HealthApi:
    """HealthApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def get_health(
        self,
        **kwargs
    ) -> Dict[str, object]:
        """Health check

        Returns service health status

        Args:

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_health_with_http_info(**kwargs)[0]

    def get_health1(
        self,
        **kwargs
    ) -> TmuxClientInternalTypesHealthResponse:
        """Get health status

        Retrieves the health status of the API server and its components

        Args:

        Returns:
            TmuxClientInternalTypesHealthResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_health1_with_http_info(**kwargs)[0]

