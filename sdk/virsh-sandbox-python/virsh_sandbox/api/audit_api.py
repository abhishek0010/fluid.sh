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
{import=from typing import Any, Dict, Optional}
{import=from typing_extensions import Annotated}
{import=from virsh_sandbox.models.tmux_client_internal_types_audit_query import TmuxClientInternalTypesAuditQuery}
{import=from virsh_sandbox.models.tmux_client_internal_types_audit_query_response import TmuxClientInternalTypesAuditQueryResponse}

class AuditApi:
    """AuditApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def get_audit_stats(
        self,
        **kwargs
    ) -> Dict[str, object]:
        """Get audit stats

        Retrieves audit log statistics

        Args:

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_audit_stats_with_http_info(**kwargs)[0]

    def query_audit_log(
        self,
        request: Optional[TmuxClientInternalTypesAuditQuery] = None,
        **kwargs
    ) -> TmuxClientInternalTypesAuditQueryResponse:
        """Query audit log

        Queries the audit log for entries

        Args:
            request: Audit query

        Returns:
            TmuxClientInternalTypesAuditQueryResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.query_audit_log_with_http_info(request=request, **kwargs)[0]

