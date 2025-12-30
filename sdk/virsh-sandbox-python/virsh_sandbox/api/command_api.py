# coding: utf-8

"""
    virsh-sandbox API
    API for managing virtual machine sandboxes using libvirt
"""

from typing import Optional, Dict, Any, List
import asyncio

from virsh_sandbox.api_client import ApiClient
from virsh_sandbox.exceptions import ApiException
{import=from pydantic import Field}
{import=from typing import Any, Dict}
{import=from typing_extensions import Annotated}
{import=from virsh_sandbox.models.tmux_client_internal_types_run_command_request import TmuxClientInternalTypesRunCommandRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_run_command_response import TmuxClientInternalTypesRunCommandResponse}

class CommandApi:
    """CommandApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def get_allowed_commands(
        self,
        **kwargs
    ) -> Dict[str, object]:
        """Get allowed commands

        Retrieves the list of allowed and denied commands

        Args:

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_allowed_commands_with_http_info(**kwargs)[0]

    def run_command(
        self,
        request: TmuxClientInternalTypesRunCommandRequest,
        **kwargs
    ) -> TmuxClientInternalTypesRunCommandResponse:
        """Run command

        Executes a shell command

        Args:
            request: Run command request

        Returns:
            TmuxClientInternalTypesRunCommandResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.run_command_with_http_info(request=request, **kwargs)[0]

