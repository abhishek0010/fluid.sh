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
{import=from typing import Any, Dict, List, Optional}
{import=from typing_extensions import Annotated}
{import=from virsh_sandbox.models.tmux_client_internal_types_create_pane_request import TmuxClientInternalTypesCreatePaneRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_create_pane_response import TmuxClientInternalTypesCreatePaneResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_kill_session_response import TmuxClientInternalTypesKillSessionResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_list_panes_response import TmuxClientInternalTypesListPanesResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_read_pane_request import TmuxClientInternalTypesReadPaneRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_read_pane_response import TmuxClientInternalTypesReadPaneResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_send_keys_request import TmuxClientInternalTypesSendKeysRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_send_keys_response import TmuxClientInternalTypesSendKeysResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_session_info import TmuxClientInternalTypesSessionInfo}
{import=from virsh_sandbox.models.tmux_client_internal_types_switch_pane_request import TmuxClientInternalTypesSwitchPaneRequest}
{import=from virsh_sandbox.models.tmux_client_internal_types_switch_pane_response import TmuxClientInternalTypesSwitchPaneResponse}
{import=from virsh_sandbox.models.tmux_client_internal_types_window_info import TmuxClientInternalTypesWindowInfo}

class TmuxApi:
    """TmuxApi service

    """

    def __init__(self, api_client: Optional[ApiClient] = None):
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def create_tmux_pane(
        self,
        request: TmuxClientInternalTypesCreatePaneRequest,
        **kwargs
    ) -> TmuxClientInternalTypesCreatePaneResponse:
        """Create tmux pane

        Creates a new tmux pane

        Args:
            request: Create pane request

        Returns:
            TmuxClientInternalTypesCreatePaneResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.create_tmux_pane_with_http_info(request=request, **kwargs)[0]

    def create_tmux_session(
        self,
        request: object,
        **kwargs
    ) -> Dict[str, str]:
        """Create tmux session

        Creates a new tmux session

        Args:
            request: Create session request

        Returns:
            Dict[str, str]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.create_tmux_session_with_http_info(request=request, **kwargs)[0]

    def kill_tmux_pane(
        self,
        pane_id: str,
        **kwargs
    ) -> Dict[str, object]:
        """Kill tmux pane

        Kills a tmux pane

        Args:
            pane_id: Pane ID

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.kill_tmux_pane_with_http_info(pane_id=pane_id, **kwargs)[0]

    def kill_tmux_session(
        self,
        session_name: str,
        **kwargs
    ) -> Dict[str, object]:
        """Kill tmux session

        Kills a tmux session

        Args:
            session_name: Session name

        Returns:
            Dict[str, object]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.kill_tmux_session_with_http_info(session_name=session_name, **kwargs)[0]

    def list_tmux_panes(
        self,
        session: Optional[str] = None,
        **kwargs
    ) -> TmuxClientInternalTypesListPanesResponse:
        """List tmux panes

        Get a list of panes in a tmux session

        Args:
            session: Session name

        Returns:
            TmuxClientInternalTypesListPanesResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.list_tmux_panes_with_http_info(session=session, **kwargs)[0]

    def list_tmux_sessions(
        self,
        **kwargs
    ) -> List[TmuxClientInternalTypesSessionInfo]:
        """List tmux sessions

        Get a list of all active tmux sessions

        Args:

        Returns:
            List[TmuxClientInternalTypesSessionInfo]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.list_tmux_sessions_with_http_info(**kwargs)[0]

    def list_tmux_windows(
        self,
        session: Optional[str] = None,
        **kwargs
    ) -> List[TmuxClientInternalTypesWindowInfo]:
        """List tmux windows

        Get a list of windows in a tmux session

        Args:
            session: Session name

        Returns:
            List[TmuxClientInternalTypesWindowInfo]: 

        Raises:
            ApiException: If the API call fails
        """
        return self.list_tmux_windows_with_http_info(session=session, **kwargs)[0]

    def read_tmux_pane(
        self,
        request: TmuxClientInternalTypesReadPaneRequest,
        **kwargs
    ) -> TmuxClientInternalTypesReadPaneResponse:
        """Read tmux pane

        Reads the content of a tmux pane

        Args:
            request: Read pane request

        Returns:
            TmuxClientInternalTypesReadPaneResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.read_tmux_pane_with_http_info(request=request, **kwargs)[0]

    def release_tmux_session(
        self,
        session_id: str,
        **kwargs
    ) -> TmuxClientInternalTypesKillSessionResponse:
        """Release tmux session

        Releases (kills) a tmux session by ID

        Args:
            session_id: Session ID

        Returns:
            TmuxClientInternalTypesKillSessionResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.release_tmux_session_with_http_info(session_id=session_id, **kwargs)[0]

    def send_keys_to_pane(
        self,
        request: TmuxClientInternalTypesSendKeysRequest,
        **kwargs
    ) -> TmuxClientInternalTypesSendKeysResponse:
        """Send keys to tmux pane

        Sends keystrokes to a tmux pane

        Args:
            request: Send keys request

        Returns:
            TmuxClientInternalTypesSendKeysResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.send_keys_to_pane_with_http_info(request=request, **kwargs)[0]

    def switch_tmux_pane(
        self,
        request: TmuxClientInternalTypesSwitchPaneRequest,
        **kwargs
    ) -> TmuxClientInternalTypesSwitchPaneResponse:
        """Switch tmux pane

        Switches to a specific tmux pane

        Args:
            request: Switch pane request

        Returns:
            TmuxClientInternalTypesSwitchPaneResponse: 

        Raises:
            ApiException: If the API call fails
        """
        return self.switch_tmux_pane_with_http_info(request=request, **kwargs)[0]

