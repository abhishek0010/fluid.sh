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
{import=from virsh_sandbox.models.tmux_client_internal_types_copy_file_request import TmuxClientInternalTypesCopyFileRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_copy_file_response import TmuxClientInternalTypesCopyFileResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_delete_file_request import TmuxClientInternalTypesDeleteFileRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_delete_file_response import TmuxClientInternalTypesDeleteFileResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_edit_file_request import TmuxClientInternalTypesEditFileRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_edit_file_response import TmuxClientInternalTypesEditFileResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_list_dir_request import TmuxClientInternalTypesListDirRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_list_dir_response import TmuxClientInternalTypesListDirResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_read_file_request import TmuxClientInternalTypesReadFileRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_read_file_response import TmuxClientInternalTypesReadFileResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_write_file_request import TmuxClientInternalTypesWriteFileRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_write_file_response import TmuxClientInternalTypesWriteFileResponse}

class FileApi:
    """FileApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def check_file_exists(
        self,
        request: object,
        **kwargs
    ) -> Dict[str, object]:
        """Check if file exists

        Checks if a file or directory exists

        Args:
            request: File exists request

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.check_file_exists_with_http_info(request=request, **kwargs)[0]

    def copy_file(
        self,
        request: TmuxClientInternalTypesCopyFileRequest,
        **kwargs
    ) -> TmuxClientInternalTypesCopyFileResponse:
        """Copy file

        Copies a file from source to destination

        Args:
            request: Copy file request

        Returns:
            TmuxClientInternalTypesCopyFileResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.copy_file_with_http_info(request=request, **kwargs)[0]

    def delete_file(
        self,
        request: TmuxClientInternalTypesDeleteFileRequest,
        **kwargs
    ) -> TmuxClientInternalTypesDeleteFileResponse:
        """Delete file

        Deletes a file or directory

        Args:
            request: Delete file request

        Returns:
            TmuxClientInternalTypesDeleteFileResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.delete_file_with_http_info(request=request, **kwargs)[0]

    def edit_file(
        self,
        request: TmuxClientInternalTypesEditFileRequest,
        **kwargs
    ) -> TmuxClientInternalTypesEditFileResponse:
        """Edit file

        Edits the content of a file

        Args:
            request: Edit file request

        Returns:
            TmuxClientInternalTypesEditFileResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.edit_file_with_http_info(request=request, **kwargs)[0]

    def get_file_hash(
        self,
        request: object,
        **kwargs
    ) -> Dict[str, str]:
        """Get file hash

        Computes the SHA256 hash of a file

        Args:
            request: File hash request

        Returns:
            Dict[str, str]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.get_file_hash_with_http_info(request=request, **kwargs)[0]

    def list_directory(
        self,
        request: TmuxClientInternalTypesListDirRequest,
        **kwargs
    ) -> TmuxClientInternalTypesListDirResponse:
        """List directory contents

        Lists the contents of a directory

        Args:
            request: List directory request

        Returns:
            TmuxClientInternalTypesListDirResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.list_directory_with_http_info(request=request, **kwargs)[0]

    def read_file(
        self,
        request: TmuxClientInternalTypesReadFileRequest,
        **kwargs
    ) -> TmuxClientInternalTypesReadFileResponse:
        """Read file

        Reads the content of a file

        Args:
            request: Read file request

        Returns:
            TmuxClientInternalTypesReadFileResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.read_file_with_http_info(request=request, **kwargs)[0]

    def write_file(
        self,
        request: TmuxClientInternalTypesWriteFileRequest,
        **kwargs
    ) -> TmuxClientInternalTypesWriteFileResponse:
        """Write file

        Writes content to a file

        Args:
            request: Write file request

        Returns:
            TmuxClientInternalTypesWriteFileResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.write_file_with_http_info(request=request, **kwargs)[0]

