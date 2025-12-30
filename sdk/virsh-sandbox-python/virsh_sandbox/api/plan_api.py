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
{import=from typing import Any, Dict, Optional}
{import=from typing_extensions import Annotated}
{import=from virsh_sandbox.models.tmux_client_internal_types_create_plan_request import TmuxClientInternalTypesCreatePlanRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_create_plan_response import TmuxClientInternalTypesCreatePlanResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_get_plan_response import TmuxClientInternalTypesGetPlanResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_list_plans_response import TmuxClientInternalTypesListPlansResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_update_plan_request import TmuxClientInternalTypesUpdatePlanRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_update_plan_response import TmuxClientInternalTypesUpdatePlanResponse}

class PlanApi:
    """PlanApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def abort_plan(
        self,
        plan_id: str,
        request: Optional[object] = None,
        **kwargs
    ) -> Dict[str, object]:
        """Abort plan

        Aborts an execution plan

        Args:
            plan_id: Plan ID
            request: Abort plan request

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.abort_plan_with_http_info(plan_id=plan_id, request=request, **kwargs)[0]

    def advance_plan_step(
        self,
        plan_id: str,
        request: Optional[object] = None,
        **kwargs
    ) -> Dict[str, object]:
        """Advance plan step

        Advances to the next step in a plan

        Args:
            plan_id: Plan ID
            request: Advance step request

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.advance_plan_step_with_http_info(plan_id=plan_id, request=request, **kwargs)[0]

    def create_plan(
        self,
        request: TmuxClientInternalTypesCreatePlanRequest,
        **kwargs
    ) -> TmuxClientInternalTypesCreatePlanResponse:
        """Create plan

        Creates a new execution plan

        Args:
            request: Create plan request

        Returns:
            TmuxClientInternalTypesCreatePlanResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.create_plan_with_http_info(request=request, **kwargs)[0]

    def delete_plan(
        self,
        plan_id: str,
        **kwargs
    ) -> Dict[str, object]:
        """Delete plan

        Deletes an execution plan

        Args:
            plan_id: Plan ID

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.delete_plan_with_http_info(plan_id=plan_id, **kwargs)[0]

    def get_plan(
        self,
        plan_id: str,
        **kwargs
    ) -> TmuxClientInternalTypesGetPlanResponse:
        """Get plan

        Retrieves a specific execution plan

        Args:
            plan_id: Plan ID

        Returns:
            TmuxClientInternalTypesGetPlanResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_plan_with_http_info(plan_id=plan_id, **kwargs)[0]

    def list_plans(
        self,
        **kwargs
    ) -> TmuxClientInternalTypesListPlansResponse:
        """List plans

        Lists all execution plans

        Args:

        Returns:
            TmuxClientInternalTypesListPlansResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.list_plans_with_http_info(**kwargs)[0]

    def update_plan(
        self,
        request: TmuxClientInternalTypesUpdatePlanRequest,
        **kwargs
    ) -> TmuxClientInternalTypesUpdatePlanResponse:
        """Update plan

        Updates an execution plan

        Args:
            request: Update plan request

        Returns:
            TmuxClientInternalTypesUpdatePlanResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.update_plan_with_http_info(request=request, **kwargs)[0]

