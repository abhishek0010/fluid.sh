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
{import=from typing import Any, Dict}
{import=from typing_extensions import Annotated}
{import=from virsh_sandbox.models.tmux_client_internal_types_approve_request import TmuxClientInternalTypesApproveRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_ask_human_request import TmuxClientInternalTypesAskHumanRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_ask_human_response import TmuxClientInternalTypesAskHumanResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_list_approvals_response import TmuxClientInternalTypesListApprovalsResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_pending_approval import TmuxClientInternalTypesPendingApproval}

class HumanApi:
    """HumanApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def ask_human(
        self,
        request: TmuxClientInternalTypesAskHumanRequest,
        **kwargs
    ) -> TmuxClientInternalTypesAskHumanResponse:
        """Request human approval

        Requests approval from a human for an action

        Args:
            request: Ask human request

        Returns:
            TmuxClientInternalTypesAskHumanResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.ask_human_with_http_info(request=request, **kwargs)[0]

    def ask_human_async(
        self,
        request: TmuxClientInternalTypesAskHumanRequest,
        **kwargs
    ) -> Dict[str, str]:
        """Request human approval asynchronously

        Requests approval from a human asynchronously

        Args:
            request: Ask human async request

        Returns:
            Dict[str, str]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.ask_human_async_with_http_info(request=request, **kwargs)[0]

    def cancel_approval(
        self,
        request_id: str,
        **kwargs
    ) -> Dict[str, object]:
        """Cancel approval

        Cancels a pending approval request

        Args:
            request_id: Request ID

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.cancel_approval_with_http_info(request_id=request_id, **kwargs)[0]

    def get_pending_approval(
        self,
        request_id: str,
        **kwargs
    ) -> TmuxClientInternalTypesPendingApproval:
        """Get pending approval

        Retrieves a specific pending approval request

        Args:
            request_id: Request ID

        Returns:
            TmuxClientInternalTypesPendingApproval: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_pending_approval_with_http_info(request_id=request_id, **kwargs)[0]

    def list_pending_approvals(
        self,
        **kwargs
    ) -> TmuxClientInternalTypesListApprovalsResponse:
        """List pending approvals

        Lists all pending human approval requests

        Args:

        Returns:
            TmuxClientInternalTypesListApprovalsResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.list_pending_approvals_with_http_info(**kwargs)[0]

    def respond_to_approval(
        self,
        request: TmuxClientInternalTypesApproveRequest,
        **kwargs
    ) -> TmuxClientInternalTypesAskHumanResponse:
        """Respond to approval

        Responds to a pending approval request

        Args:
            request: Approve request

        Returns:
            TmuxClientInternalTypesAskHumanResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.respond_to_approval_with_http_info(request=request, **kwargs)[0]

