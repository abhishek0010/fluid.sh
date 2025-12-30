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
{import=from typing_extensions import Annotated}
{import=from virsh_sandbox.models.internal_ansible_job import InternalAnsibleJob}
{import=from virsh_sandbox.models.internal_ansible_job_request import InternalAnsibleJobRequest}
{import=from virsh_sandbox.models.internal_ansible_job_response import InternalAnsibleJobResponse}

class AnsibleApi:
    """AnsibleApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def create_ansible_job(
        self,
        request: InternalAnsibleJobRequest,
        **kwargs
    ) -> InternalAnsibleJobResponse:
        """Create Ansible job

        Creates a new Ansible playbook execution job

        Args:
            request: Job creation parameters

        Returns:
            InternalAnsibleJobResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.create_ansible_job_with_http_info(request=request, **kwargs)[0]

    def get_ansible_job(
        self,
        job_id: str,
        **kwargs
    ) -> InternalAnsibleJob:
        """Get Ansible job

        Gets the status of an Ansible job

        Args:
            job_id: Job ID

        Returns:
            InternalAnsibleJob: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_ansible_job_with_http_info(job_id=job_id, **kwargs)[0]

    def stream_ansible_job_output(
        self,
        job_id: str,
        **kwargs
    ) -> None:
        """Stream Ansible job output

        Connects via WebSocket to run an Ansible job and stream output

        Args:
            job_id: Job ID

        Returns:
            None

        Raises:
            ApiException: If the API call fails
        """
        return self.stream_ansible_job_output_with_http_info(job_id=job_id, **kwargs)[0]

