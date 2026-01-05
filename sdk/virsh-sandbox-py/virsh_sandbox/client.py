# coding: utf-8

"""
Unified VirshSandbox Client

This module provides a unified client wrapper for the virsh-sandbox SDK,
offering a cleaner interface with flattened parameters instead of request objects.

Example:
    from virsh_sandbox import VirshSandbox

    client = VirshSandbox(host="http://localhost:8080")
    # Create a sandbox with simple parameters
    client.sandbox.create_sandbox(source_vm_name="ubuntu-base")
    # Run a command
    client.command.run_command(command="ls", args=["-la"])
"""

from typing import Any, Dict, List, Optional, Tuple, Union

from typing_extensions import TypedDict

from virsh_sandbox.api.access_api import AccessApi
from virsh_sandbox.api.ansible_api import AnsibleApi
from virsh_sandbox.api.audit_api import AuditApi
from virsh_sandbox.api.command_api import CommandApi
from virsh_sandbox.api.file_api import FileApi
from virsh_sandbox.api.health_api import HealthApi
from virsh_sandbox.api.human_api import HumanApi
from virsh_sandbox.api.plan_api import PlanApi
from virsh_sandbox.api.sandbox_api import SandboxApi
from virsh_sandbox.api.tmux_api import TmuxApi
from virsh_sandbox.api.vms_api import VMsApi
from virsh_sandbox.api_client import ApiClient
from virsh_sandbox.configuration import Configuration
from virsh_sandbox.models.internal_ansible_job import InternalAnsibleJob
from virsh_sandbox.models.internal_ansible_job_request import \
    InternalAnsibleJobRequest
from virsh_sandbox.models.internal_ansible_job_response import \
    InternalAnsibleJobResponse
from virsh_sandbox.models.internal_api_create_sandbox_session_request import \
    InternalApiCreateSandboxSessionRequest
from virsh_sandbox.models.internal_api_create_sandbox_session_response import \
    InternalApiCreateSandboxSessionResponse
from virsh_sandbox.models.internal_api_list_sandbox_sessions_response import \
    InternalApiListSandboxSessionsResponse
from virsh_sandbox.models.internal_api_sandbox_session_info import \
    InternalApiSandboxSessionInfo
from virsh_sandbox.models.internal_rest_ca_public_key_response import \
    InternalRestCaPublicKeyResponse
from virsh_sandbox.models.internal_rest_certificate_response import \
    InternalRestCertificateResponse
from virsh_sandbox.models.internal_rest_list_certificates_response import \
    InternalRestListCertificatesResponse
from virsh_sandbox.models.internal_rest_list_sessions_response import \
    InternalRestListSessionsResponse
from virsh_sandbox.models.internal_rest_request_access_request import \
    InternalRestRequestAccessRequest
from virsh_sandbox.models.internal_rest_request_access_response import \
    InternalRestRequestAccessResponse
from virsh_sandbox.models.internal_rest_revoke_certificate_request import \
    InternalRestRevokeCertificateRequest
from virsh_sandbox.models.internal_rest_session_end_request import \
    InternalRestSessionEndRequest
from virsh_sandbox.models.internal_rest_session_start_request import \
    InternalRestSessionStartRequest
from virsh_sandbox.models.internal_rest_session_start_response import \
    InternalRestSessionStartResponse
from virsh_sandbox.models.tmux_client_internal_types_approve_request import \
    TmuxClientInternalTypesApproveRequest
from virsh_sandbox.models.tmux_client_internal_types_ask_human_request import \
    TmuxClientInternalTypesAskHumanRequest
from virsh_sandbox.models.tmux_client_internal_types_ask_human_response import \
    TmuxClientInternalTypesAskHumanResponse
from virsh_sandbox.models.tmux_client_internal_types_audit_query import \
    TmuxClientInternalTypesAuditQuery
from virsh_sandbox.models.tmux_client_internal_types_audit_query_response import \
    TmuxClientInternalTypesAuditQueryResponse
from virsh_sandbox.models.tmux_client_internal_types_copy_file_request import \
    TmuxClientInternalTypesCopyFileRequest
from virsh_sandbox.models.tmux_client_internal_types_copy_file_response import \
    TmuxClientInternalTypesCopyFileResponse
from virsh_sandbox.models.tmux_client_internal_types_create_pane_request import \
    TmuxClientInternalTypesCreatePaneRequest
from virsh_sandbox.models.tmux_client_internal_types_create_pane_response import \
    TmuxClientInternalTypesCreatePaneResponse
from virsh_sandbox.models.tmux_client_internal_types_create_plan_request import \
    TmuxClientInternalTypesCreatePlanRequest
from virsh_sandbox.models.tmux_client_internal_types_create_plan_response import \
    TmuxClientInternalTypesCreatePlanResponse
from virsh_sandbox.models.tmux_client_internal_types_delete_file_request import \
    TmuxClientInternalTypesDeleteFileRequest
from virsh_sandbox.models.tmux_client_internal_types_delete_file_response import \
    TmuxClientInternalTypesDeleteFileResponse
from virsh_sandbox.models.tmux_client_internal_types_edit_file_request import \
    TmuxClientInternalTypesEditFileRequest
from virsh_sandbox.models.tmux_client_internal_types_edit_file_response import \
    TmuxClientInternalTypesEditFileResponse
from virsh_sandbox.models.tmux_client_internal_types_get_plan_response import \
    TmuxClientInternalTypesGetPlanResponse
from virsh_sandbox.models.tmux_client_internal_types_health_response import \
    TmuxClientInternalTypesHealthResponse
from virsh_sandbox.models.tmux_client_internal_types_kill_session_response import \
    TmuxClientInternalTypesKillSessionResponse
from virsh_sandbox.models.tmux_client_internal_types_list_approvals_response import \
    TmuxClientInternalTypesListApprovalsResponse
from virsh_sandbox.models.tmux_client_internal_types_list_dir_request import \
    TmuxClientInternalTypesListDirRequest
from virsh_sandbox.models.tmux_client_internal_types_list_dir_response import \
    TmuxClientInternalTypesListDirResponse
from virsh_sandbox.models.tmux_client_internal_types_list_panes_response import \
    TmuxClientInternalTypesListPanesResponse
from virsh_sandbox.models.tmux_client_internal_types_list_plans_response import \
    TmuxClientInternalTypesListPlansResponse
from virsh_sandbox.models.tmux_client_internal_types_pending_approval import \
    TmuxClientInternalTypesPendingApproval
from virsh_sandbox.models.tmux_client_internal_types_read_file_request import \
    TmuxClientInternalTypesReadFileRequest
from virsh_sandbox.models.tmux_client_internal_types_read_file_response import \
    TmuxClientInternalTypesReadFileResponse
from virsh_sandbox.models.tmux_client_internal_types_read_pane_request import \
    TmuxClientInternalTypesReadPaneRequest
from virsh_sandbox.models.tmux_client_internal_types_read_pane_response import \
    TmuxClientInternalTypesReadPaneResponse
from virsh_sandbox.models.tmux_client_internal_types_run_command_request import \
    TmuxClientInternalTypesRunCommandRequest
from virsh_sandbox.models.tmux_client_internal_types_run_command_response import \
    TmuxClientInternalTypesRunCommandResponse
from virsh_sandbox.models.tmux_client_internal_types_send_keys_request import \
    TmuxClientInternalTypesSendKeysRequest
from virsh_sandbox.models.tmux_client_internal_types_send_keys_response import \
    TmuxClientInternalTypesSendKeysResponse
from virsh_sandbox.models.tmux_client_internal_types_session_info import \
    TmuxClientInternalTypesSessionInfo
from virsh_sandbox.models.tmux_client_internal_types_step_status import \
    TmuxClientInternalTypesStepStatus
from virsh_sandbox.models.tmux_client_internal_types_switch_pane_request import \
    TmuxClientInternalTypesSwitchPaneRequest
from virsh_sandbox.models.tmux_client_internal_types_switch_pane_response import \
    TmuxClientInternalTypesSwitchPaneResponse
from virsh_sandbox.models.tmux_client_internal_types_update_plan_request import \
    TmuxClientInternalTypesUpdatePlanRequest
from virsh_sandbox.models.tmux_client_internal_types_update_plan_response import \
    TmuxClientInternalTypesUpdatePlanResponse
from virsh_sandbox.models.tmux_client_internal_types_window_info import \
    TmuxClientInternalTypesWindowInfo
from virsh_sandbox.models.tmux_client_internal_types_write_file_request import \
    TmuxClientInternalTypesWriteFileRequest
from virsh_sandbox.models.tmux_client_internal_types_write_file_response import \
    TmuxClientInternalTypesWriteFileResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_create_sandbox_request import \
    VirshSandboxInternalRestCreateSandboxRequest
from virsh_sandbox.models.virsh_sandbox_internal_rest_create_sandbox_response import \
    VirshSandboxInternalRestCreateSandboxResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_destroy_sandbox_response import \
    VirshSandboxInternalRestDestroySandboxResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_diff_request import \
    VirshSandboxInternalRestDiffRequest
from virsh_sandbox.models.virsh_sandbox_internal_rest_diff_response import \
    VirshSandboxInternalRestDiffResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_inject_ssh_key_request import \
    VirshSandboxInternalRestInjectSSHKeyRequest
from virsh_sandbox.models.virsh_sandbox_internal_rest_list_sandboxes_response import \
    VirshSandboxInternalRestListSandboxesResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_list_vms_response import \
    VirshSandboxInternalRestListVMsResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_publish_request import \
    VirshSandboxInternalRestPublishRequest
from virsh_sandbox.models.virsh_sandbox_internal_rest_run_command_request import \
    VirshSandboxInternalRestRunCommandRequest
from virsh_sandbox.models.virsh_sandbox_internal_rest_run_command_response import \
    VirshSandboxInternalRestRunCommandResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_snapshot_request import \
    VirshSandboxInternalRestSnapshotRequest
from virsh_sandbox.models.virsh_sandbox_internal_rest_snapshot_response import \
    VirshSandboxInternalRestSnapshotResponse
from virsh_sandbox.models.virsh_sandbox_internal_rest_start_sandbox_request import \
    VirshSandboxInternalRestStartSandboxRequest
from virsh_sandbox.models.virsh_sandbox_internal_rest_start_sandbox_response import \
    VirshSandboxInternalRestStartSandboxResponse


# TypedDict definitions for response types
class InternalAnsibleJobDict(TypedDict, total=False):
    """
    Dictionary representation of InternalAnsibleJob.

    Keys:
        check (bool): check
        id (str): id
        playbook (str): playbook
        status (str): status
        vm_name (str): vm_name
    """

    check: Optional[bool]
    id: Optional[str]
    playbook: Optional[str]
    status: Optional[str]
    vm_name: Optional[str]


class InternalAnsibleJobResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalAnsibleJobResponse.

    Keys:
        job_id (str): job_id
        ws_url (str): ws_url
    """

    job_id: Optional[str]
    ws_url: Optional[str]


class InternalApiCreateSandboxSessionResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalApiCreateSandboxSessionResponse.

    Keys:
        message (str): Message provides additional information
        sandbox_id (str): SandboxID is the sandbox being accessed
        session_id (str): SessionID is the tmux session ID
        session_name (str): SessionName is the tmux session name
        ttl_seconds (int): TTLSeconds is the remaining certificate validity in seconds
        username (str): Username is the SSH username
        valid_until (str): ValidUntil is when the certificate expires (RFC3339)
        vm_ip_address (str): VMIPAddress is the IP of the sandbox VM
    """

    message: Optional[str]
    sandbox_id: Optional[str]
    session_id: Optional[str]
    session_name: Optional[str]
    ttl_seconds: Optional[int]
    username: Optional[str]
    valid_until: Optional[str]
    vm_ip_address: Optional[str]


class InternalApiSandboxSessionInfoDict(TypedDict, total=False):
    """
    Dictionary representation of InternalApiSandboxSessionInfo.

    Keys:
        is_expired (bool): is_expired
        sandbox_id (str): sandbox_id
        session_id (str): session_id
        session_name (str): session_name
        ttl_seconds (int): ttl_seconds
        username (str): username
        valid_until (str): valid_until
        vm_ip_address (str): vm_ip_address
    """

    is_expired: Optional[bool]
    sandbox_id: Optional[str]
    session_id: Optional[str]
    session_name: Optional[str]
    ttl_seconds: Optional[int]
    username: Optional[str]
    valid_until: Optional[str]
    vm_ip_address: Optional[str]


class InternalApiListSandboxSessionsResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalApiListSandboxSessionsResponse.

    Keys:
        sessions (List[InternalApiSandboxSessionInfoDict]): sessions
        total (int): total
    """

    sessions: Optional[List[InternalApiSandboxSessionInfoDict]]
    total: Optional[int]


class InternalRestAccessErrorResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestAccessErrorResponse.

    Keys:
        code (int): code
        details (str): details
        error (str): error
    """

    code: Optional[int]
    details: Optional[str]
    error: Optional[str]


class InternalRestCaPublicKeyResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestCaPublicKeyResponse.

    Keys:
        public_key (str): PublicKey is the CA public key in OpenSSH format.
        usage (str): Usage explains how to use this key.
    """

    public_key: Optional[str]
    usage: Optional[str]


class InternalRestCertificateResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestCertificateResponse.

    Keys:
        id (str): id
        identity (str): identity
        is_expired (bool): is_expired
        issued_at (str): issued_at
        principals (List[str]): principals
        sandbox_id (str): sandbox_id
        serial_number (int): serial_number
        status (str): status
        ttl_seconds (int): ttl_seconds
        user_id (str): user_id
        valid_after (str): valid_after
        valid_before (str): valid_before
        vm_id (str): vm_id
    """

    id: Optional[str]
    identity: Optional[str]
    is_expired: Optional[bool]
    issued_at: Optional[str]
    principals: Optional[List[str]]
    sandbox_id: Optional[str]
    serial_number: Optional[int]
    status: Optional[str]
    ttl_seconds: Optional[int]
    user_id: Optional[str]
    valid_after: Optional[str]
    valid_before: Optional[str]
    vm_id: Optional[str]


class InternalRestErrorResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestErrorResponse.

    Keys:
        code (int): code
        details (str): details
        error (str): error
    """

    code: Optional[int]
    details: Optional[str]
    error: Optional[str]


class InternalRestGenerateResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestGenerateResponse.

    Keys:
        message (str): message
        note (str): note
    """

    message: Optional[str]
    note: Optional[str]


class InternalRestListCertificatesResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestListCertificatesResponse.

    Keys:
        certificates (List[InternalRestCertificateResponseDict]): certificates
        total (int): total
    """

    certificates: Optional[List[InternalRestCertificateResponseDict]]
    total: Optional[int]


class InternalRestPublishResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestPublishResponse.

    Keys:
        message (str): message
        note (str): note
    """

    message: Optional[str]
    note: Optional[str]


class InternalRestRequestAccessResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestRequestAccessResponse.

    Keys:
        certificate (str): Certificate is the SSH certificate content (save as key-cert.pub).
        certificate_id (str): CertificateID is the ID of the issued certificate.
        connect_command (str): ConnectCommand is an example SSH command for connecting.
        instructions (str): Instructions provides usage instructions.
        ssh_port (int): SSHPort is the SSH port (usually 22).
        ttl_seconds (int): TTLSeconds is the remaining validity in seconds.
        username (str): Username is the SSH username to use.
        valid_until (str): ValidUntil is when the certificate expires (RFC3339).
        vm_ip_address (str): VMIPAddress is the IP address of the sandbox VM.
    """

    certificate: Optional[str]
    certificate_id: Optional[str]
    connect_command: Optional[str]
    instructions: Optional[str]
    ssh_port: Optional[int]
    ttl_seconds: Optional[int]
    username: Optional[str]
    valid_until: Optional[str]
    vm_ip_address: Optional[str]


class InternalRestSandboxInfoDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestSandboxInfo.

    Keys:
        agent_id (str): agent_id
        base_image (str): base_image
        created_at (str): created_at
        id (str): id
        ip_address (str): ip_address
        job_id (str): job_id
        network (str): network
        sandbox_name (str): sandbox_name
        state (str): state
        ttl_seconds (int): ttl_seconds
        updated_at (str): updated_at
    """

    agent_id: Optional[str]
    base_image: Optional[str]
    created_at: Optional[str]
    id: Optional[str]
    ip_address: Optional[str]
    job_id: Optional[str]
    network: Optional[str]
    sandbox_name: Optional[str]
    state: Optional[str]
    ttl_seconds: Optional[int]
    updated_at: Optional[str]


class InternalRestListSandboxesResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestListSandboxesResponse.

    Keys:
        sandboxes (List[InternalRestSandboxInfoDict]): sandboxes
        total (int): total
    """

    sandboxes: Optional[List[InternalRestSandboxInfoDict]]
    total: Optional[int]


class InternalRestSessionResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestSessionResponse.

    Keys:
        certificate_id (str): certificate_id
        duration_seconds (int): duration_seconds
        ended_at (str): ended_at
        id (str): id
        sandbox_id (str): sandbox_id
        source_ip (str): source_ip
        started_at (str): started_at
        status (str): status
        user_id (str): user_id
        vm_id (str): vm_id
        vm_ip_address (str): vm_ip_address
    """

    certificate_id: Optional[str]
    duration_seconds: Optional[int]
    ended_at: Optional[str]
    id: Optional[str]
    sandbox_id: Optional[str]
    source_ip: Optional[str]
    started_at: Optional[str]
    status: Optional[str]
    user_id: Optional[str]
    vm_id: Optional[str]
    vm_ip_address: Optional[str]


class InternalRestListSessionsResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestListSessionsResponse.

    Keys:
        sessions (List[InternalRestSessionResponseDict]): sessions
        total (int): total
    """

    sessions: Optional[List[InternalRestSessionResponseDict]]
    total: Optional[int]


class InternalRestSessionStartResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestSessionStartResponse.

    Keys:
        session_id (str): session_id
    """

    session_id: Optional[str]


class InternalRestStartSandboxResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestStartSandboxResponse.

    Keys:
        ip_address (str): ip_address
    """

    ip_address: Optional[str]


class InternalRestVmInfoDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestVmInfo.

    Keys:
        disk_path (str): disk_path
        name (str): name
        persistent (bool): persistent
        state (str): state
        uuid (str): uuid
    """

    disk_path: Optional[str]
    name: Optional[str]
    persistent: Optional[bool]
    state: Optional[str]
    uuid: Optional[str]


class InternalRestListVMsResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestListVMsResponse.

    Keys:
        vms (List[InternalRestVmInfoDict]): vms
    """

    vms: Optional[List[InternalRestVmInfoDict]]


class TimeDurationDict(TypedDict, total=False):
    pass


class TmuxClientInternalApiCreateSandboxSessionResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalApiCreateSandboxSessionResponse.

    Keys:
        message (str): Message provides additional information
        sandbox_id (str): SandboxID is the sandbox being accessed
        session_id (str): SessionID is the tmux session ID
        session_name (str): SessionName is the tmux session name
        ttl_seconds (int): TTLSeconds is the remaining certificate validity in seconds
        username (str): Username is the SSH username
        valid_until (str): ValidUntil is when the certificate expires (RFC3339)
        vm_ip_address (str): VMIPAddress is the IP of the sandbox VM
    """

    message: Optional[str]
    sandbox_id: Optional[str]
    session_id: Optional[str]
    session_name: Optional[str]
    ttl_seconds: Optional[int]
    username: Optional[str]
    valid_until: Optional[str]
    vm_ip_address: Optional[str]


class TmuxClientInternalApiSandboxSessionInfoDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalApiSandboxSessionInfo.

    Keys:
        is_expired (bool): is_expired
        sandbox_id (str): sandbox_id
        session_id (str): session_id
        session_name (str): session_name
        ttl_seconds (int): ttl_seconds
        username (str): username
        valid_until (str): valid_until
        vm_ip_address (str): vm_ip_address
    """

    is_expired: Optional[bool]
    sandbox_id: Optional[str]
    session_id: Optional[str]
    session_name: Optional[str]
    ttl_seconds: Optional[int]
    username: Optional[str]
    valid_until: Optional[str]
    vm_ip_address: Optional[str]


class TmuxClientInternalApiListSandboxSessionsResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalApiListSandboxSessionsResponse.

    Keys:
        sessions (List[TmuxClientInternalApiSandboxSessionInfoDict]): sessions
        total (int): total
    """

    sessions: Optional[List[TmuxClientInternalApiSandboxSessionInfoDict]]
    total: Optional[int]


class TmuxClientInternalTypesAPIErrorDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesAPIError.

    Keys:
        code (str): code
        details (str): details
        message (str): message
    """

    code: Optional[str]
    details: Optional[str]
    message: Optional[str]


class TmuxClientInternalTypesAskHumanResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesAskHumanResponse.

    Keys:
        approved (bool): approved
        approved_at (str): approved_at
        approved_by (str): approved_by
        comment (str): comment
        expires_at (str): expires_at
        request_id (str): request_id
        status (str): status
    """

    approved: Optional[bool]
    approved_at: Optional[str]
    approved_by: Optional[str]
    comment: Optional[str]
    expires_at: Optional[str]
    request_id: Optional[str]
    status: Optional[str]


class TmuxClientInternalTypesAuditEntryDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesAuditEntry.

    Keys:
        action (str): action
        arguments (List[int]): arguments
        client_ip (str): client_ip
        duration_ms (int): duration_ms
        error (TmuxClientInternalTypesAPIErrorDict): error
        request_id (str): request_id
        result (List[int]): result
        timestamp (str): timestamp
        tool (str): tool
        user_agent (str): user_agent
    """

    action: Optional[str]
    arguments: Optional[List[int]]
    client_ip: Optional[str]
    duration_ms: Optional[int]
    error: Optional[TmuxClientInternalTypesAPIErrorDict]
    request_id: Optional[str]
    result: Optional[List[int]]
    timestamp: Optional[str]
    tool: Optional[str]
    user_agent: Optional[str]


class TmuxClientInternalTypesAuditQueryResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesAuditQueryResponse.

    Keys:
        entries (List[TmuxClientInternalTypesAuditEntryDict]): entries
        has_more (bool): has_more
        total_count (int): total_count
    """

    entries: Optional[List[TmuxClientInternalTypesAuditEntryDict]]
    has_more: Optional[bool]
    total_count: Optional[int]


class TmuxClientInternalTypesComponentHealthDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesComponentHealth.

    Keys:
        message (str): message
        name (str): name
        status (str): status
    """

    message: Optional[str]
    name: Optional[str]
    status: Optional[str]


class TmuxClientInternalTypesCopyFileResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesCopyFileResponse.

    Keys:
        bytes_copied (int): bytes_copied
        copied (bool): copied
        destination (str): destination
        source (str): source
    """

    bytes_copied: Optional[int]
    copied: Optional[bool]
    destination: Optional[str]
    source: Optional[str]


class TmuxClientInternalTypesCreatePaneResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesCreatePaneResponse.

    Keys:
        pane_id (str): pane_id
        pane_index (int): pane_index
        session_name (str): session_name
        window_index (int): window_index
    """

    pane_id: Optional[str]
    pane_index: Optional[int]
    session_name: Optional[str]
    window_index: Optional[int]


class TmuxClientInternalTypesDeleteFileResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesDeleteFileResponse.

    Keys:
        deleted (bool): deleted
        path (str): path
        was_dir (bool): was_dir
    """

    deleted: Optional[bool]
    path: Optional[str]
    was_dir: Optional[bool]


class TmuxClientInternalTypesEditFileResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesEditFileResponse.

    Keys:
        content_after (str): For audit trail
        content_before (str): For audit trail
        diff (str): Unified diff format
        edited (bool): edited
        path (str): path
        replacements (int): replacements
    """

    content_after: Optional[str]
    content_before: Optional[str]
    diff: Optional[str]
    edited: Optional[bool]
    path: Optional[str]
    replacements: Optional[int]


class TmuxClientInternalTypesFileInfoDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesFileInfo.

    Keys:
        is_dir (bool): is_dir
        mod_time (str): mod_time
        mode (str): mode
        name (str): name
        path (str): path
        size (int): size
    """

    is_dir: Optional[bool]
    mod_time: Optional[str]
    mode: Optional[str]
    name: Optional[str]
    path: Optional[str]
    size: Optional[int]


class TmuxClientInternalTypesHealthResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesHealthResponse.

    Keys:
        components (List[TmuxClientInternalTypesComponentHealthDict]): components
        status (str): status
        uptime (str): uptime
        version (str): version
    """

    components: Optional[List[TmuxClientInternalTypesComponentHealthDict]]
    status: Optional[str]
    uptime: Optional[str]
    version: Optional[str]


class TmuxClientInternalTypesKillSessionResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesKillSessionResponse.

    Keys:
        session_name (str): session_name
        success (bool): success
    """

    session_name: Optional[str]
    success: Optional[bool]


class TmuxClientInternalTypesListDirResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesListDirResponse.

    Keys:
        files (List[TmuxClientInternalTypesFileInfoDict]): files
        path (str): path
    """

    files: Optional[List[TmuxClientInternalTypesFileInfoDict]]
    path: Optional[str]


class TmuxClientInternalTypesPaneInfoDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesPaneInfo.

    Keys:
        active (bool): active
        current_path (str): current_path
        pane_height (int): pane_height
        pane_id (str): pane_id
        pane_index (int): pane_index
        pane_pid (int): pane_pid
        pane_title (str): pane_title
        pane_width (int): pane_width
        session_name (str): session_name
        window_index (int): window_index
        window_name (str): window_name
    """

    active: Optional[bool]
    current_path: Optional[str]
    pane_height: Optional[int]
    pane_id: Optional[str]
    pane_index: Optional[int]
    pane_pid: Optional[int]
    pane_title: Optional[str]
    pane_width: Optional[int]
    session_name: Optional[str]
    window_index: Optional[int]
    window_name: Optional[str]


class TmuxClientInternalTypesListPanesResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesListPanesResponse.

    Keys:
        panes (List[TmuxClientInternalTypesPaneInfoDict]): panes
    """

    panes: Optional[List[TmuxClientInternalTypesPaneInfoDict]]


class TmuxClientInternalTypesPendingApprovalDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesPendingApproval.

    Keys:
        action_type (str): action_type
        context (str): context
        created_at (str): created_at
        expires_at (str): expires_at
        prompt (str): prompt
        request_id (str): request_id
        status (str): status
        urgency (str): urgency
    """

    action_type: Optional[str]
    context: Optional[str]
    created_at: Optional[str]
    expires_at: Optional[str]
    prompt: Optional[str]
    request_id: Optional[str]
    status: Optional[str]
    urgency: Optional[str]


class TmuxClientInternalTypesListApprovalsResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesListApprovalsResponse.

    Keys:
        pending (List[TmuxClientInternalTypesPendingApprovalDict]): pending
    """

    pending: Optional[List[TmuxClientInternalTypesPendingApprovalDict]]


class TmuxClientInternalTypesPlanStepDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesPlanStep.

    Keys:
        completed_at (str): completed_at
        description (str): description
        error (str): error
        index (int): index
        result (str): result
        started_at (str): started_at
        status (str): status
    """

    completed_at: Optional[str]
    description: Optional[str]
    error: Optional[str]
    index: Optional[int]
    result: Optional[str]
    started_at: Optional[str]
    status: Optional[str]


class TmuxClientInternalTypesPlanDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesPlan.

    Keys:
        completed_at (str): completed_at
        created_at (str): created_at
        current_step (int): -1 if not started
        description (str): description
        id (str): id
        name (str): name
        status (str): status
        steps (List[TmuxClientInternalTypesPlanStepDict]): steps
        updated_at (str): updated_at
    """

    completed_at: Optional[str]
    created_at: Optional[str]
    current_step: Optional[int]
    description: Optional[str]
    id: Optional[str]
    name: Optional[str]
    status: Optional[str]
    steps: Optional[List[TmuxClientInternalTypesPlanStepDict]]
    updated_at: Optional[str]


class TmuxClientInternalTypesCreatePlanResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesCreatePlanResponse.

    Keys:
        plan (TmuxClientInternalTypesPlanDict): plan
        plan_id (str): plan_id
    """

    plan: Optional[TmuxClientInternalTypesPlanDict]
    plan_id: Optional[str]


class TmuxClientInternalTypesGetPlanResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesGetPlanResponse.

    Keys:
        plan (TmuxClientInternalTypesPlanDict): plan
    """

    plan: Optional[TmuxClientInternalTypesPlanDict]


class TmuxClientInternalTypesListPlansResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesListPlansResponse.

    Keys:
        plans (List[TmuxClientInternalTypesPlanDict]): plans
    """

    plans: Optional[List[TmuxClientInternalTypesPlanDict]]


class TmuxClientInternalTypesReadFileResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesReadFileResponse.

    Keys:
        content (str): content
        from_line (int): from_line
        mod_time (str): mod_time
        mode (str): mode
        path (str): path
        size (int): size
        to_line (int): to_line
        total_lines (int): total_lines
        truncated (bool): truncated
    """

    content: Optional[str]
    from_line: Optional[int]
    mod_time: Optional[str]
    mode: Optional[str]
    path: Optional[str]
    size: Optional[int]
    to_line: Optional[int]
    total_lines: Optional[int]
    truncated: Optional[bool]


class TmuxClientInternalTypesReadPaneResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesReadPaneResponse.

    Keys:
        content (str): content
        lines (int): lines
        pane_id (str): pane_id
    """

    content: Optional[str]
    lines: Optional[int]
    pane_id: Optional[str]


class TmuxClientInternalTypesRunCommandResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesRunCommandResponse.

    Keys:
        args (List[str]): args
        command (str): command
        dry_run (bool): dry_run
        duration_ms (int): duration_ms
        exit_code (int): exit_code
        stderr (str): stderr
        stdout (str): stdout
        timed_out (bool): timed_out
    """

    args: Optional[List[str]]
    command: Optional[str]
    dry_run: Optional[bool]
    duration_ms: Optional[int]
    exit_code: Optional[int]
    stderr: Optional[str]
    stdout: Optional[str]
    timed_out: Optional[bool]


class TmuxClientInternalTypesSendKeysResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesSendKeysResponse.

    Keys:
        pane_id (str): pane_id
        sent (bool): sent
    """

    pane_id: Optional[str]
    sent: Optional[bool]


class TmuxClientInternalTypesSessionInfoDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesSessionInfo.

    Keys:
        attached (bool): attached
        created (str): created
        id (str): id
        last_pane_x (int): last_pane_x
        last_pane_y (int): last_pane_y
        name (str): name
        windows (int): windows
    """

    attached: Optional[bool]
    created: Optional[str]
    id: Optional[str]
    last_pane_x: Optional[int]
    last_pane_y: Optional[int]
    name: Optional[str]
    windows: Optional[int]


class TmuxClientInternalTypesSwitchPaneResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesSwitchPaneResponse.

    Keys:
        pane_id (str): pane_id
        switched (bool): switched
    """

    pane_id: Optional[str]
    switched: Optional[bool]


class TmuxClientInternalTypesUpdatePlanResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesUpdatePlanResponse.

    Keys:
        plan (TmuxClientInternalTypesPlanDict): plan
        plan_id (str): plan_id
        updated (bool): updated
    """

    plan: Optional[TmuxClientInternalTypesPlanDict]
    plan_id: Optional[str]
    updated: Optional[bool]


class TmuxClientInternalTypesWindowInfoDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesWindowInfo.

    Keys:
        active (bool): active
        height (int): height
        index (int): index
        name (str): name
        panes (int): panes
        session_name (str): session_name
        width (int): width
    """

    active: Optional[bool]
    height: Optional[int]
    index: Optional[int]
    name: Optional[str]
    panes: Optional[int]
    session_name: Optional[str]
    width: Optional[int]


class TmuxClientInternalTypesWriteFileResponseDict(TypedDict, total=False):
    """
    Dictionary representation of TmuxClientInternalTypesWriteFileResponse.

    Keys:
        bytes_written (int): bytes_written
        created (bool): true if file was created, false if overwritten
        path (str): path
        written (bool): written
    """

    bytes_written: Optional[int]
    created: Optional[bool]
    path: Optional[str]
    written: Optional[bool]


class VirshSandboxInternalAnsibleJobDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalAnsibleJob.

    Keys:
        check (bool): check
        id (str): id
        playbook (str): playbook
        status (str): status
        vm_name (str): vm_name
    """

    check: Optional[bool]
    id: Optional[str]
    playbook: Optional[str]
    status: Optional[str]
    vm_name: Optional[str]


class VirshSandboxInternalAnsibleJobResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalAnsibleJobResponse.

    Keys:
        job_id (str): job_id
        ws_url (str): ws_url
    """

    job_id: Optional[str]
    ws_url: Optional[str]


class VirshSandboxInternalErrorErrorResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalErrorErrorResponse.

    Keys:
        code (int): code
        details (str): details
        error (str): error
    """

    code: Optional[int]
    details: Optional[str]
    error: Optional[str]


class VirshSandboxInternalRestAccessErrorResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestAccessErrorResponse.

    Keys:
        code (int): code
        details (str): details
        error (str): error
    """

    code: Optional[int]
    details: Optional[str]
    error: Optional[str]


class VirshSandboxInternalRestCaPublicKeyResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestCaPublicKeyResponse.

    Keys:
        public_key (str): PublicKey is the CA public key in OpenSSH format.
        usage (str): Usage explains how to use this key.
    """

    public_key: Optional[str]
    usage: Optional[str]


class VirshSandboxInternalRestCertificateResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestCertificateResponse.

    Keys:
        id (str): id
        identity (str): identity
        is_expired (bool): is_expired
        issued_at (str): issued_at
        principals (List[str]): principals
        sandbox_id (str): sandbox_id
        serial_number (int): serial_number
        status (str): status
        ttl_seconds (int): ttl_seconds
        user_id (str): user_id
        valid_after (str): valid_after
        valid_before (str): valid_before
        vm_id (str): vm_id
    """

    id: Optional[str]
    identity: Optional[str]
    is_expired: Optional[bool]
    issued_at: Optional[str]
    principals: Optional[List[str]]
    sandbox_id: Optional[str]
    serial_number: Optional[int]
    status: Optional[str]
    ttl_seconds: Optional[int]
    user_id: Optional[str]
    valid_after: Optional[str]
    valid_before: Optional[str]
    vm_id: Optional[str]


class VirshSandboxInternalRestDestroySandboxResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestDestroySandboxResponse.

    Keys:
        base_image (str): base_image
        sandbox_name (str): sandbox_name
        state (str): state
        ttl_seconds (int): ttl_seconds
    """

    base_image: Optional[str]
    sandbox_name: Optional[str]
    state: Optional[str]
    ttl_seconds: Optional[int]


class VirshSandboxInternalRestErrorResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestErrorResponse.

    Keys:
        code (int): code
        details (str): details
        error (str): error
    """

    code: Optional[int]
    details: Optional[str]
    error: Optional[str]


class VirshSandboxInternalRestGenerateResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestGenerateResponse.

    Keys:
        message (str): message
        note (str): note
    """

    message: Optional[str]
    note: Optional[str]


class VirshSandboxInternalRestListCertificatesResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestListCertificatesResponse.

    Keys:
        certificates (List[VirshSandboxInternalRestCertificateResponseDict]): certificates
        total (int): total
    """

    certificates: Optional[List[VirshSandboxInternalRestCertificateResponseDict]]
    total: Optional[int]


class VirshSandboxInternalRestPublishResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestPublishResponse.

    Keys:
        message (str): message
        note (str): note
    """

    message: Optional[str]
    note: Optional[str]


class VirshSandboxInternalRestRequestAccessResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestRequestAccessResponse.

    Keys:
        certificate (str): Certificate is the SSH certificate content (save as key-cert.pub).
        certificate_id (str): CertificateID is the ID of the issued certificate.
        connect_command (str): ConnectCommand is an example SSH command for connecting.
        instructions (str): Instructions provides usage instructions.
        ssh_port (int): SSHPort is the SSH port (usually 22).
        ttl_seconds (int): TTLSeconds is the remaining validity in seconds.
        username (str): Username is the SSH username to use.
        valid_until (str): ValidUntil is when the certificate expires (RFC3339).
        vm_ip_address (str): VMIPAddress is the IP address of the sandbox VM.
    """

    certificate: Optional[str]
    certificate_id: Optional[str]
    connect_command: Optional[str]
    instructions: Optional[str]
    ssh_port: Optional[int]
    ttl_seconds: Optional[int]
    username: Optional[str]
    valid_until: Optional[str]
    vm_ip_address: Optional[str]


class VirshSandboxInternalRestSandboxInfoDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestSandboxInfo.

    Keys:
        agent_id (str): agent_id
        base_image (str): base_image
        created_at (str): created_at
        id (str): id
        ip_address (str): ip_address
        job_id (str): job_id
        network (str): network
        sandbox_name (str): sandbox_name
        state (str): state
        ttl_seconds (int): ttl_seconds
        updated_at (str): updated_at
    """

    agent_id: Optional[str]
    base_image: Optional[str]
    created_at: Optional[str]
    id: Optional[str]
    ip_address: Optional[str]
    job_id: Optional[str]
    network: Optional[str]
    sandbox_name: Optional[str]
    state: Optional[str]
    ttl_seconds: Optional[int]
    updated_at: Optional[str]


class VirshSandboxInternalRestListSandboxesResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestListSandboxesResponse.

    Keys:
        sandboxes (List[VirshSandboxInternalRestSandboxInfoDict]): sandboxes
        total (int): total
    """

    sandboxes: Optional[List[VirshSandboxInternalRestSandboxInfoDict]]
    total: Optional[int]


class VirshSandboxInternalRestSessionResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestSessionResponse.

    Keys:
        certificate_id (str): certificate_id
        duration_seconds (int): duration_seconds
        ended_at (str): ended_at
        id (str): id
        sandbox_id (str): sandbox_id
        source_ip (str): source_ip
        started_at (str): started_at
        status (str): status
        user_id (str): user_id
        vm_id (str): vm_id
        vm_ip_address (str): vm_ip_address
    """

    certificate_id: Optional[str]
    duration_seconds: Optional[int]
    ended_at: Optional[str]
    id: Optional[str]
    sandbox_id: Optional[str]
    source_ip: Optional[str]
    started_at: Optional[str]
    status: Optional[str]
    user_id: Optional[str]
    vm_id: Optional[str]
    vm_ip_address: Optional[str]


class VirshSandboxInternalRestListSessionsResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestListSessionsResponse.

    Keys:
        sessions (List[VirshSandboxInternalRestSessionResponseDict]): sessions
        total (int): total
    """

    sessions: Optional[List[VirshSandboxInternalRestSessionResponseDict]]
    total: Optional[int]


class VirshSandboxInternalRestSessionStartResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestSessionStartResponse.

    Keys:
        session_id (str): session_id
    """

    session_id: Optional[str]


class VirshSandboxInternalRestStartSandboxResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestStartSandboxResponse.

    Keys:
        ip_address (str): ip_address
    """

    ip_address: Optional[str]


class VirshSandboxInternalRestVmInfoDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestVmInfo.

    Keys:
        disk_path (str): disk_path
        name (str): name
        persistent (bool): persistent
        state (str): state
        uuid (str): uuid
    """

    disk_path: Optional[str]
    name: Optional[str]
    persistent: Optional[bool]
    state: Optional[str]
    uuid: Optional[str]


class VirshSandboxInternalRestListVMsResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestListVMsResponse.

    Keys:
        vms (List[VirshSandboxInternalRestVmInfoDict]): vms
    """

    vms: Optional[List[VirshSandboxInternalRestVmInfoDict]]


class VirshSandboxInternalStoreCommandExecRecordDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreCommandExecRecord.

    Keys:
        redacted (Dict[str, str]): placeholders for secrets redaction
        timeout (TimeDurationDict): timeout
        user (str): user
        work_dir (str): work_dir
    """

    redacted: Optional[Dict[str, str]]
    timeout: Optional[TimeDurationDict]
    user: Optional[str]
    work_dir: Optional[str]


class VirshSandboxInternalStoreCommandDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreCommand.

    Keys:
        command (str): command
        ended_at (str): ended_at
        env_json (str): JSON-encoded env map
        exit_code (int): exit_code
        id (str): id
        metadata (VirshSandboxInternalStoreCommandExecRecordDict): metadata
        sandbox_id (str): sandbox_id
        started_at (str): started_at
        stderr (str): stderr
        stdout (str): stdout
    """

    command: Optional[str]
    ended_at: Optional[str]
    env_json: Optional[str]
    exit_code: Optional[int]
    id: Optional[str]
    metadata: Optional[VirshSandboxInternalStoreCommandExecRecordDict]
    sandbox_id: Optional[str]
    started_at: Optional[str]
    stderr: Optional[str]
    stdout: Optional[str]


class InternalRestRunCommandResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestRunCommandResponse.

    Keys:
        command (VirshSandboxInternalStoreCommandDict): command
    """

    command: Optional[VirshSandboxInternalStoreCommandDict]


class VirshSandboxInternalRestRunCommandResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestRunCommandResponse.

    Keys:
        command (VirshSandboxInternalStoreCommandDict): command
    """

    command: Optional[VirshSandboxInternalStoreCommandDict]


class VirshSandboxInternalStoreCommandSummaryDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreCommandSummary.

    Keys:
        at (str): at
        cmd (str): cmd
        exit_code (int): exit_code
    """

    at: Optional[str]
    cmd: Optional[str]
    exit_code: Optional[int]


class VirshSandboxInternalStorePackageInfoDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStorePackageInfo.

    Keys:
        name (str): name
        version (str): version
    """

    name: Optional[str]
    version: Optional[str]


class VirshSandboxInternalStoreSandboxDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreSandbox.

    Keys:
        agent_id (str): requesting agent identity
        base_image (str): base qcow2 filename
        created_at (str): Metadata
        deleted_at (str): deleted_at
        id (str): e.g., \
        ip_address (str): discovered IP (if any)
        job_id (str): correlation id for the end-to-end change set
        network (str): libvirt network name
        sandbox_name (str): libvirt domain name
        state (str): state
        ttl_seconds (int): optional TTL for auto GC
        updated_at (str): updated_at
    """

    agent_id: Optional[str]
    base_image: Optional[str]
    created_at: Optional[str]
    deleted_at: Optional[str]
    id: Optional[str]
    ip_address: Optional[str]
    job_id: Optional[str]
    network: Optional[str]
    sandbox_name: Optional[str]
    state: Optional[str]
    ttl_seconds: Optional[int]
    updated_at: Optional[str]


class InternalRestCreateSandboxResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestCreateSandboxResponse.

    Keys:
        sandbox (VirshSandboxInternalStoreSandboxDict): sandbox
    """

    sandbox: Optional[VirshSandboxInternalStoreSandboxDict]


class VirshSandboxInternalRestCreateSandboxResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestCreateSandboxResponse.

    Keys:
        sandbox (VirshSandboxInternalStoreSandboxDict): sandbox
        ip_address (str): populated when auto_start and wait_for_ip are true
    """

    sandbox: Optional[VirshSandboxInternalStoreSandboxDict]
    ip_address: Optional[str]


class VirshSandboxInternalStoreServiceChangeDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreServiceChange.

    Keys:
        enabled (bool): enabled
        name (str): name
        state (str): started|stopped|restarted|reloaded
    """

    enabled: Optional[bool]
    name: Optional[str]
    state: Optional[str]


class VirshSandboxInternalStoreChangeDiffDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreChangeDiff.

    Keys:
        commands_run (List[VirshSandboxInternalStoreCommandSummaryDict]): commands_run
        files_added (List[str]): files_added
        files_modified (List[str]): files_modified
        files_removed (List[str]): files_removed
        packages_added (List[VirshSandboxInternalStorePackageInfoDict]): packages_added
        packages_removed (List[VirshSandboxInternalStorePackageInfoDict]): packages_removed
        services_changed (List[VirshSandboxInternalStoreServiceChangeDict]): services_changed
    """

    commands_run: Optional[List[VirshSandboxInternalStoreCommandSummaryDict]]
    files_added: Optional[List[str]]
    files_modified: Optional[List[str]]
    files_removed: Optional[List[str]]
    packages_added: Optional[List[VirshSandboxInternalStorePackageInfoDict]]
    packages_removed: Optional[List[VirshSandboxInternalStorePackageInfoDict]]
    services_changed: Optional[List[VirshSandboxInternalStoreServiceChangeDict]]


class VirshSandboxInternalStoreDiffDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreDiff.

    Keys:
        created_at (str): created_at
        diff_json (VirshSandboxInternalStoreChangeDiffDict): JSON-encoded change diff
        from_snapshot (str): from_snapshot
        id (str): id
        sandbox_id (str): sandbox_id
        to_snapshot (str): to_snapshot
    """

    created_at: Optional[str]
    diff_json: Optional[VirshSandboxInternalStoreChangeDiffDict]
    from_snapshot: Optional[str]
    id: Optional[str]
    sandbox_id: Optional[str]
    to_snapshot: Optional[str]


class InternalRestDiffResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestDiffResponse.

    Keys:
        diff (VirshSandboxInternalStoreDiffDict): diff
    """

    diff: Optional[VirshSandboxInternalStoreDiffDict]


class VirshSandboxInternalRestDiffResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestDiffResponse.

    Keys:
        diff (VirshSandboxInternalStoreDiffDict): diff
    """

    diff: Optional[VirshSandboxInternalStoreDiffDict]


class VirshSandboxInternalStoreSnapshotDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalStoreSnapshot.

    Keys:
        created_at (str): created_at
        id (str): id
        kind (str): kind
        meta_json (str): optional JSON metadata
        name (str): logical name (unique per sandbox)
        ref (str): Ref is a backend-specific reference: for internal snapshots this could be a UUID or name, for external snapshots it could be a file path to the overlay qcow2.
        sandbox_id (str): sandbox_id
    """

    created_at: Optional[str]
    id: Optional[str]
    kind: Optional[str]
    meta_json: Optional[str]
    name: Optional[str]
    ref: Optional[str]
    sandbox_id: Optional[str]


class InternalRestSnapshotResponseDict(TypedDict, total=False):
    """
    Dictionary representation of InternalRestSnapshotResponse.

    Keys:
        snapshot (VirshSandboxInternalStoreSnapshotDict): snapshot
    """

    snapshot: Optional[VirshSandboxInternalStoreSnapshotDict]


class VirshSandboxInternalRestSnapshotResponseDict(TypedDict, total=False):
    """
    Dictionary representation of VirshSandboxInternalRestSnapshotResponse.

    Keys:
        snapshot (VirshSandboxInternalStoreSnapshotDict): snapshot
    """

    snapshot: Optional[VirshSandboxInternalStoreSnapshotDict]


def _to_dict(obj: Any) -> Any:
    """Convert a response object to a dictionary.

    This helper function handles the conversion of API response objects
    to dictionaries for easier consumption.

    Args:
        obj: The object to convert. Can be None, a dict, a list,
             a response object with to_dict() method, or a primitive.

    Returns:
        The converted dictionary, list of dictionaries, or the original
        value if it's already a dict/primitive.
    """
    if obj is None:
        return None
    if isinstance(obj, dict):
        return obj
    if isinstance(obj, list):
        return [_to_dict(item) for item in obj]
    if hasattr(obj, "to_dict"):
        return obj.to_dict()
    return obj


class AccessOperations:
    """Wrapper for AccessApi with simplified method signatures."""

    def __init__(self, api: AccessApi):
        self._api = api

    def v1_access_ca_pubkey_get(self) -> InternalRestCaPublicKeyResponseDict:
        """Get the SSH CA public key

        Returns:
            Dict with keys:
                - public_key (str): PublicKey is the CA public key in OpenSSH format.
                - usage (str): Usage explains how to use this key.
        """
        return _to_dict(self._api.v1_access_ca_pubkey_get())

    def v1_access_certificate_cert_id_delete(
        self,
        cert_id: str,
        reason: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Revoke a certificate

        Args:
            cert_id: str
            reason: reason

        Returns:
            Dict with keys:
        """
        request = InternalRestRevokeCertificateRequest(
            reason=reason,
        )
        return _to_dict(
            self._api.v1_access_certificate_cert_id_delete(
                cert_id=cert_id, request=request
            )
        )

    def v1_access_certificate_cert_id_get(
        self,
        cert_id: str,
    ) -> InternalRestCertificateResponseDict:
        """Get certificate details

        Args:
            cert_id: str

        Returns:
            Dict with keys:
                - id (str)
                - identity (str)
                - is_expired (bool)
                - issued_at (str)
                - principals (List[str])
                - sandbox_id (str)
                - serial_number (int)
                - status (str)
                - ttl_seconds (int)
                - user_id (str)
                - valid_after (str)
                - valid_before (str)
                - vm_id (str)
        """
        return _to_dict(self._api.v1_access_certificate_cert_id_get(cert_id=cert_id))

    def v1_access_certificates_get(
        self,
        sandbox_id: Optional[str] = None,
        user_id: Optional[str] = None,
        status: Optional[str] = None,
        active_only: Optional[bool] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> InternalRestListCertificatesResponseDict:
        """List certificates

        Args:
            sandbox_id: Optional[str]
            user_id: Optional[str]
            status: Optional[str]
            active_only: Optional[bool]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            Dict with keys:
                - certificates (List[InternalRestCertificateResponse])
                - total (int)
        """
        return _to_dict(
            self._api.v1_access_certificates_get(
                sandbox_id=sandbox_id,
                user_id=user_id,
                status=status,
                active_only=active_only,
                limit=limit,
                offset=offset,
            )
        )

    def v1_access_request_post(
        self,
        public_key: Optional[str] = None,
        sandbox_id: Optional[str] = None,
        ttl_minutes: Optional[int] = None,
        user_id: Optional[str] = None,
    ) -> InternalRestRequestAccessResponseDict:
        """Request SSH access to a sandbox

        Args:
            public_key: PublicKey is the user
            sandbox_id: SandboxID is the target sandbox.
            ttl_minutes: TTLMinutes is the requested access duration (1-10 minutes).
            user_id: UserID identifies the requesting user.

        Returns:
            Dict with keys:
                - certificate (str): Certificate is the SSH certificate content (save as key-cert.pub).
                - certificate_id (str): CertificateID is the ID of the issued certificate.
                - connect_command (str): ConnectCommand is an example SSH command for connecting.
                - instructions (str): Instructions provides usage instructions.
                - ssh_port (int): SSHPort is the SSH port (usually 22).
                - ttl_seconds (int): TTLSeconds is the remaining validity in seconds.
                - username (str): Username is the SSH username to use.
                - valid_until (str): ValidUntil is when the certificate expires (RFC3339).
                - vm_ip_address (str): VMIPAddress is the IP address of the sandbox VM.
        """
        request = InternalRestRequestAccessRequest(
            public_key=public_key,
            sandbox_id=sandbox_id,
            ttl_minutes=ttl_minutes,
            user_id=user_id,
        )
        return _to_dict(self._api.v1_access_request_post(request=request))

    def v1_access_session_end_post(
        self,
        reason: Optional[str] = None,
        session_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Record session end

        Args:
            reason: reason
            session_id: session_id

        Returns:
            Dict with keys:
        """
        request = InternalRestSessionEndRequest(
            reason=reason,
            session_id=session_id,
        )
        return _to_dict(self._api.v1_access_session_end_post(request=request))

    def v1_access_session_start_post(
        self,
        certificate_id: Optional[str] = None,
        source_ip: Optional[str] = None,
    ) -> InternalRestSessionStartResponseDict:
        """Record session start

        Args:
            certificate_id: certificate_id
            source_ip: source_ip

        Returns:
            Dict with keys:
                - session_id (str)
        """
        request = InternalRestSessionStartRequest(
            certificate_id=certificate_id,
            source_ip=source_ip,
        )
        return _to_dict(self._api.v1_access_session_start_post(request=request))

    def v1_access_sessions_get(
        self,
        sandbox_id: Optional[str] = None,
        certificate_id: Optional[str] = None,
        user_id: Optional[str] = None,
        active_only: Optional[bool] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> InternalRestListSessionsResponseDict:
        """List sessions

        Args:
            sandbox_id: Optional[str]
            certificate_id: Optional[str]
            user_id: Optional[str]
            active_only: Optional[bool]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            Dict with keys:
                - sessions (List[InternalRestSessionResponse])
                - total (int)
        """
        return _to_dict(
            self._api.v1_access_sessions_get(
                sandbox_id=sandbox_id,
                certificate_id=certificate_id,
                user_id=user_id,
                active_only=active_only,
                limit=limit,
                offset=offset,
            )
        )


class AnsibleOperations:
    """Wrapper for AnsibleApi with simplified method signatures."""

    def __init__(self, api: AnsibleApi):
        self._api = api

    def create_ansible_job(
        self,
        check: Optional[bool] = None,
        playbook: Optional[str] = None,
        vm_name: Optional[str] = None,
    ) -> InternalAnsibleJobResponseDict:
        """Create Ansible job

        Args:
            check: check
            playbook: playbook
            vm_name: vm_name

        Returns:
            Dict with keys:
                - job_id (str)
                - ws_url (str)
        """
        request = InternalAnsibleJobRequest(
            check=check,
            playbook=playbook,
            vm_name=vm_name,
        )
        return _to_dict(self._api.create_ansible_job(request=request))

    def get_ansible_job(
        self,
        job_id: str,
    ) -> InternalAnsibleJobDict:
        """Get Ansible job

        Args:
            job_id: str

        Returns:
            Dict with keys:
                - check (bool)
                - id (str)
                - playbook (str)
                - status (InternalAnsibleJobStatus)
                - vm_name (str)
        """
        return _to_dict(self._api.get_ansible_job(job_id=job_id))

    def stream_ansible_job_output(
        self,
        job_id: str,
    ) -> None:
        """Stream Ansible job output

        Args:
            job_id: str
        """
        return self._api.stream_ansible_job_output(job_id=job_id)


class AuditOperations:
    """Wrapper for AuditApi with simplified method signatures."""

    def __init__(self, api: AuditApi):
        self._api = api

    def get_audit_stats(self) -> Dict[str, Any]:
        """Get audit stats

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.get_audit_stats())

    def query_audit_log(
        self,
        action: Optional[str] = None,
        limit: Optional[int] = None,
        request_id: Optional[str] = None,
        since: Optional[str] = None,
        tool: Optional[str] = None,
        until: Optional[str] = None,
    ) -> TmuxClientInternalTypesAuditQueryResponseDict:
        """Query audit log

        Args:
            action: action
            limit: limit
            request_id: request_id
            since: since
            tool: tool
            until: until

        Returns:
            Dict with keys:
                - entries (List[TmuxClientInternalTypesAuditEntry])
                - has_more (bool)
                - total_count (int)
        """
        request = TmuxClientInternalTypesAuditQuery(
            action=action,
            limit=limit,
            request_id=request_id,
            since=since,
            tool=tool,
            until=until,
        )
        return _to_dict(self._api.query_audit_log(request=request))


class CommandOperations:
    """Wrapper for CommandApi with simplified method signatures."""

    def __init__(self, api: CommandApi):
        self._api = api

    def get_allowed_commands(self) -> Dict[str, Any]:
        """Get allowed commands

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.get_allowed_commands())

    def run_command(
        self,
        args: Optional[List[str]] = None,
        command: Optional[str] = None,
        dry_run: Optional[bool] = None,
        env: Optional[List[str]] = None,
        timeout: Optional[int] = None,
        work_dir: Optional[str] = None,
    ) -> TmuxClientInternalTypesRunCommandResponseDict:
        """Run command

        Args:
            args: Arguments as separate items
            command: Executable name only
            dry_run: If true, don
            env: Additional env vars (KEY=VALUE)
            timeout: Seconds, 0 = default (30s)
            work_dir: Working directory

        Returns:
            Dict with keys:
                - args (List[str])
                - command (str)
                - dry_run (bool)
                - duration_ms (int)
                - exit_code (int)
                - stderr (str)
                - stdout (str)
                - timed_out (bool)
        """
        request = TmuxClientInternalTypesRunCommandRequest(
            args=args,
            command=command,
            dry_run=dry_run,
            env=env,
            timeout=timeout,
            work_dir=work_dir,
        )
        return _to_dict(self._api.run_command(request=request))


class FileOperations:
    """Wrapper for FileApi with simplified method signatures."""

    def __init__(self, api: FileApi):
        self._api = api

    def check_file_exists(self) -> Dict[str, Any]:
        """Check if file exists

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.check_file_exists(request={}))

    def copy_file(
        self,
        destination: Optional[str] = None,
        overwrite: Optional[bool] = None,
        source: Optional[str] = None,
    ) -> TmuxClientInternalTypesCopyFileResponseDict:
        """Copy file

        Args:
            destination: destination
            overwrite: overwrite
            source: source

        Returns:
            Dict with keys:
                - bytes_copied (int)
                - copied (bool)
                - destination (str)
                - source (str)
        """
        request = TmuxClientInternalTypesCopyFileRequest(
            destination=destination,
            overwrite=overwrite,
            source=source,
        )
        return _to_dict(self._api.copy_file(request=request))

    def delete_file(
        self,
        path: Optional[str] = None,
        recursive: Optional[bool] = None,
    ) -> TmuxClientInternalTypesDeleteFileResponseDict:
        """Delete file

        Args:
            path: path
            recursive: For directories

        Returns:
            Dict with keys:
                - deleted (bool)
                - path (str)
                - was_dir (bool)
        """
        request = TmuxClientInternalTypesDeleteFileRequest(
            path=path,
            recursive=recursive,
        )
        return _to_dict(self._api.delete_file(request=request))

    def edit_file(
        self,
        all: Optional[bool] = None,
        new_text: Optional[str] = None,
        old_text: Optional[str] = None,
        path: Optional[str] = None,
    ) -> TmuxClientInternalTypesEditFileResponseDict:
        """Edit file

        Args:
            all: Replace all occurrences (default: first only)
            new_text: Replacement text
            old_text: Text to find and replace
            path: path

        Returns:
            Dict with keys:
                - content_after (str): For audit trail
                - content_before (str): For audit trail
                - diff (str): Unified diff format
                - edited (bool)
                - path (str)
                - replacements (int)
        """
        request = TmuxClientInternalTypesEditFileRequest(
            all=all,
            new_text=new_text,
            old_text=old_text,
            path=path,
        )
        return _to_dict(self._api.edit_file(request=request))

    def get_file_hash(self) -> Dict[str, Any]:
        """Get file hash

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.get_file_hash(request={}))

    def list_directory(
        self,
        max_depth: Optional[int] = None,
        path: Optional[str] = None,
        recursive: Optional[bool] = None,
    ) -> TmuxClientInternalTypesListDirResponseDict:
        """List directory contents

        Args:
            max_depth: max_depth
            path: path
            recursive: recursive

        Returns:
            Dict with keys:
                - files (List[TmuxClientInternalTypesFileInfo])
                - path (str)
        """
        request = TmuxClientInternalTypesListDirRequest(
            max_depth=max_depth,
            path=path,
            recursive=recursive,
        )
        return _to_dict(self._api.list_directory(request=request))

    def read_file(
        self,
        from_line: Optional[int] = None,
        max_lines: Optional[int] = None,
        path: Optional[str] = None,
        to_line: Optional[int] = None,
    ) -> TmuxClientInternalTypesReadFileResponseDict:
        """Read file

        Args:
            from_line: 1-indexed, 0 = start
            max_lines: 0 = no limit
            path: path
            to_line: 1-indexed, 0 = end

        Returns:
            Dict with keys:
                - content (str)
                - from_line (int)
                - mod_time (str)
                - mode (str)
                - path (str)
                - size (int)
                - to_line (int)
                - total_lines (int)
                - truncated (bool)
        """
        request = TmuxClientInternalTypesReadFileRequest(
            from_line=from_line,
            max_lines=max_lines,
            path=path,
            to_line=to_line,
        )
        return _to_dict(self._api.read_file(request=request))

    def write_file(
        self,
        content: Optional[str] = None,
        create_dir: Optional[bool] = None,
        mode: Optional[str] = None,
        overwrite: Optional[bool] = None,
        path: Optional[str] = None,
    ) -> TmuxClientInternalTypesWriteFileResponseDict:
        """Write file

        Args:
            content: content
            create_dir: Create parent directories if needed
            mode: e.g., \
            overwrite: Must be true to overwrite existing
            path: path

        Returns:
            Dict with keys:
                - bytes_written (int)
                - created (bool): true if file was created, false if overwritten
                - path (str)
                - written (bool)
        """
        request = TmuxClientInternalTypesWriteFileRequest(
            content=content,
            create_dir=create_dir,
            mode=mode,
            overwrite=overwrite,
            path=path,
        )
        return _to_dict(self._api.write_file(request=request))


class HealthOperations:
    """Wrapper for HealthApi with simplified method signatures."""

    def __init__(self, api: HealthApi):
        self._api = api

    def get_health(self) -> TmuxClientInternalTypesHealthResponseDict:
        """Get health status

        Returns:
            Dict with keys:
                - components (List[TmuxClientInternalTypesComponentHealth])
                - status (TmuxClientInternalTypesHealthStatus)
                - uptime (str)
                - version (str)
        """
        return _to_dict(self._api.get_health())


class HumanOperations:
    """Wrapper for HumanApi with simplified method signatures."""

    def __init__(self, api: HumanApi):
        self._api = api

    def ask_human(
        self,
        action_type: Optional[str] = None,
        alternatives: Optional[List[str]] = None,
        context: Optional[str] = None,
        prompt: Optional[str] = None,
        timeout_secs: Optional[int] = None,
        urgency: Optional[str] = None,
    ) -> TmuxClientInternalTypesAskHumanResponseDict:
        """Request human approval

        Args:
            action_type: Category: \
            alternatives: Suggested alternative actions
            context: Additional context
            prompt: Human-readable description
            timeout_secs: Auto-reject after timeout, 0 = no timeout
            urgency: \

        Returns:
            Dict with keys:
                - approved (bool)
                - approved_at (str)
                - approved_by (str)
                - comment (str)
                - expires_at (str)
                - request_id (str)
                - status (TmuxClientInternalTypesApprovalStatus)
        """
        request = TmuxClientInternalTypesAskHumanRequest(
            action_type=action_type,
            alternatives=alternatives,
            context=context,
            prompt=prompt,
            timeout_secs=timeout_secs,
            urgency=urgency,
        )
        return _to_dict(self._api.ask_human(request=request))

    def ask_human_async(
        self,
        action_type: Optional[str] = None,
        alternatives: Optional[List[str]] = None,
        context: Optional[str] = None,
        prompt: Optional[str] = None,
        timeout_secs: Optional[int] = None,
        urgency: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Request human approval asynchronously

        Args:
            action_type: Category: \
            alternatives: Suggested alternative actions
            context: Additional context
            prompt: Human-readable description
            timeout_secs: Auto-reject after timeout, 0 = no timeout
            urgency: \

        Returns:
            Dict with keys:
        """
        request = TmuxClientInternalTypesAskHumanRequest(
            action_type=action_type,
            alternatives=alternatives,
            context=context,
            prompt=prompt,
            timeout_secs=timeout_secs,
            urgency=urgency,
        )
        return _to_dict(self._api.ask_human_async(request=request))

    def cancel_approval(
        self,
        request_id: str,
    ) -> Dict[str, Any]:
        """Cancel approval

        Args:
            request_id: str

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.cancel_approval(request_id=request_id))

    def get_pending_approval(
        self,
        request_id: str,
    ) -> TmuxClientInternalTypesPendingApprovalDict:
        """Get pending approval

        Args:
            request_id: str

        Returns:
            Dict with keys:
                - action_type (str)
                - context (str)
                - created_at (str)
                - expires_at (str)
                - prompt (str)
                - request_id (str)
                - status (TmuxClientInternalTypesApprovalStatus)
                - urgency (str)
        """
        return _to_dict(self._api.get_pending_approval(request_id=request_id))

    def list_pending_approvals(
        self,
    ) -> TmuxClientInternalTypesListApprovalsResponseDict:
        """List pending approvals

        Returns:
            Dict with keys:
                - pending (List[TmuxClientInternalTypesPendingApproval])
        """
        return _to_dict(self._api.list_pending_approvals())

    def respond_to_approval(
        self,
        approved: Optional[bool] = None,
        approved_by: Optional[str] = None,
        comment: Optional[str] = None,
        request_id: Optional[str] = None,
    ) -> TmuxClientInternalTypesAskHumanResponseDict:
        """Respond to approval

        Args:
            approved: approved
            approved_by: approved_by
            comment: comment
            request_id: request_id

        Returns:
            Dict with keys:
                - approved (bool)
                - approved_at (str)
                - approved_by (str)
                - comment (str)
                - expires_at (str)
                - request_id (str)
                - status (TmuxClientInternalTypesApprovalStatus)
        """
        request = TmuxClientInternalTypesApproveRequest(
            approved=approved,
            approved_by=approved_by,
            comment=comment,
            request_id=request_id,
        )
        return _to_dict(self._api.respond_to_approval(request=request))


class PlanOperations:
    """Wrapper for PlanApi with simplified method signatures."""

    def __init__(self, api: PlanApi):
        self._api = api

    def abort_plan(
        self,
        plan_id: str,
        request: Optional[object] = None,
    ) -> Dict[str, Any]:
        """Abort plan

        Args:
            plan_id: str
            request: Optional[object]

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.abort_plan(plan_id=plan_id, request=request))

    def advance_plan_step(
        self,
        plan_id: str,
        request: Optional[object] = None,
    ) -> Dict[str, Any]:
        """Advance plan step

        Args:
            plan_id: str
            request: Optional[object]

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.advance_plan_step(plan_id=plan_id, request=request))

    def create_plan(
        self,
        description: Optional[str] = None,
        name: Optional[str] = None,
        steps: Optional[List[str]] = None,
    ) -> TmuxClientInternalTypesCreatePlanResponseDict:
        """Create plan

        Args:
            description: description
            name: name
            steps: Step descriptions

        Returns:
            Dict with keys:
                - plan (TmuxClientInternalTypesPlan)
                - plan_id (str)
        """
        request = TmuxClientInternalTypesCreatePlanRequest(
            description=description,
            name=name,
            steps=steps,
        )
        return _to_dict(self._api.create_plan(request=request))

    def delete_plan(
        self,
        plan_id: str,
    ) -> Dict[str, Any]:
        """Delete plan

        Args:
            plan_id: str

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.delete_plan(plan_id=plan_id))

    def get_plan(
        self,
        plan_id: str,
    ) -> TmuxClientInternalTypesGetPlanResponseDict:
        """Get plan

        Args:
            plan_id: str

        Returns:
            Dict with keys:
                - plan (TmuxClientInternalTypesPlan)
        """
        return _to_dict(self._api.get_plan(plan_id=plan_id))

    def list_plans(self) -> TmuxClientInternalTypesListPlansResponseDict:
        """List plans

        Returns:
            Dict with keys:
                - plans (List[TmuxClientInternalTypesPlan])
        """
        return _to_dict(self._api.list_plans())

    def update_plan(
        self,
        error: Optional[str] = None,
        plan_id: Optional[str] = None,
        result: Optional[str] = None,
        status: Optional[TmuxClientInternalTypesStepStatus] = None,
        step_index: Optional[int] = None,
    ) -> TmuxClientInternalTypesUpdatePlanResponseDict:
        """Update plan

        Args:
            error: error
            plan_id: plan_id
            result: result
            status: status
            step_index: step_index

        Returns:
            Dict with keys:
                - plan (TmuxClientInternalTypesPlan)
                - plan_id (str)
                - updated (bool)
        """
        request = TmuxClientInternalTypesUpdatePlanRequest(
            error=error,
            plan_id=plan_id,
            result=result,
            status=status,
            step_index=step_index,
        )
        return _to_dict(self._api.update_plan(request=request))


class SandboxOperations:
    """Wrapper for SandboxApi with simplified method signatures."""

    def __init__(self, api: SandboxApi):
        self._api = api

    def create_sandbox(
        self,
        agent_id: Optional[str] = None,
        cpu: Optional[int] = None,
        memory_mb: Optional[int] = None,
        source_vm_name: Optional[str] = None,
        vm_name: Optional[str] = None,
        ttl_seconds: Optional[int] = None,
        auto_start: Optional[bool] = None,
        wait_for_ip: Optional[bool] = None,
    ) -> VirshSandboxInternalRestCreateSandboxResponseDict:
        """Create a new sandbox

        Args:
            agent_id: required
            cpu: optional; default from service config if <=0
            memory_mb: optional; default from service config if <=0
            source_vm_name: required; name of existing VM in libvirt to clone from
            vm_name: optional; generated if empty
            ttl_seconds: optional; TTL for auto garbage collection
            auto_start: optional; if true, start the VM immediately after creation
            wait_for_ip: optional; if true and auto_start, wait for IP discovery

        Returns:
            Dict with keys:
                - sandbox (VirshSandboxInternalStoreSandbox)
                - ip_address (str): populated when auto_start and wait_for_ip are true
        """
        request = VirshSandboxInternalRestCreateSandboxRequest(
            agent_id=agent_id,
            cpu=cpu,
            memory_mb=memory_mb,
            source_vm_name=source_vm_name,
            vm_name=vm_name,
            ttl_seconds=ttl_seconds,
            auto_start=auto_start,
            wait_for_ip=wait_for_ip,
        )
        return _to_dict(self._api.create_sandbox(request=request))

    def create_sandbox_session(
        self,
        sandbox_id: Optional[str] = None,
        session_name: Optional[str] = None,
        ttl_minutes: Optional[int] = None,
    ) -> InternalApiCreateSandboxSessionResponseDict:
        """Create sandbox session

        Args:
            sandbox_id: SandboxID is the ID of the sandbox to connect to
            session_name: SessionName is the optional tmux session name (auto-generated if empty)
            ttl_minutes: TTLMinutes is the certificate TTL in minutes (1-10, default 5)

        Returns:
            Dict with keys:
                - message (str): Message provides additional information
                - sandbox_id (str): SandboxID is the sandbox being accessed
                - session_id (str): SessionID is the tmux session ID
                - session_name (str): SessionName is the tmux session name
                - ttl_seconds (int): TTLSeconds is the remaining certificate validity in seconds
                - username (str): Username is the SSH username
                - valid_until (str): ValidUntil is when the certificate expires (RFC3339)
                - vm_ip_address (str): VMIPAddress is the IP of the sandbox VM
        """
        request = InternalApiCreateSandboxSessionRequest(
            sandbox_id=sandbox_id,
            session_name=session_name,
            ttl_minutes=ttl_minutes,
        )
        return _to_dict(self._api.create_sandbox_session(request=request))

    def create_snapshot(
        self,
        id: str,
        external: Optional[bool] = None,
        name: Optional[str] = None,
    ) -> VirshSandboxInternalRestSnapshotResponseDict:
        """Create snapshot

        Args:
            id: str
            external: optional; default false (internal snapshot)
            name: required

        Returns:
            Dict with keys:
                - snapshot (VirshSandboxInternalStoreSnapshot)
        """
        request = VirshSandboxInternalRestSnapshotRequest(
            external=external,
            name=name,
        )
        return _to_dict(self._api.create_snapshot(id=id, request=request))

    def destroy_sandbox(
        self,
        id: str,
    ) -> VirshSandboxInternalRestDestroySandboxResponseDict:
        """Destroy sandbox

        Args:
            id: str

        Returns:
            Dict with keys:
                - base_image (str)
                - sandbox_name (str)
                - state (VirshSandboxInternalStoreSandboxState)
                - ttl_seconds (int)
        """
        return _to_dict(self._api.destroy_sandbox(id=id))

    def diff_snapshots(
        self,
        id: str,
        from_snapshot: Optional[str] = None,
        to_snapshot: Optional[str] = None,
    ) -> VirshSandboxInternalRestDiffResponseDict:
        """Diff snapshots

        Args:
            id: str
            from_snapshot: required
            to_snapshot: required

        Returns:
            Dict with keys:
                - diff (VirshSandboxInternalStoreDiff)
        """
        request = VirshSandboxInternalRestDiffRequest(
            from_snapshot=from_snapshot,
            to_snapshot=to_snapshot,
        )
        return _to_dict(self._api.diff_snapshots(id=id, request=request))

    def generate_configuration(
        self,
        id: str,
        tool: str,
    ) -> None:
        """Generate configuration

        Args:
            id: str
            tool: str
        """
        return self._api.generate_configuration(id=id, tool=tool)

    def get_sandbox_session(
        self,
        session_name: str,
    ) -> InternalApiSandboxSessionInfoDict:
        """Get sandbox session

        Args:
            session_name: str

        Returns:
            Dict with keys:
                - is_expired (bool)
                - sandbox_id (str)
                - session_id (str)
                - session_name (str)
                - ttl_seconds (int)
                - username (str)
                - valid_until (str)
                - vm_ip_address (str)
        """
        return _to_dict(self._api.get_sandbox_session(session_name=session_name))

    def inject_ssh_key(
        self,
        id: str,
        public_key: Optional[str] = None,
        username: Optional[str] = None,
    ) -> None:
        """Inject SSH key into sandbox

        Args:
            id: str
            public_key: required
            username: required (explicit); typical: \
        """
        request = VirshSandboxInternalRestInjectSSHKeyRequest(
            public_key=public_key,
            username=username,
        )
        return self._api.inject_ssh_key(id=id, request=request)

    def kill_sandbox_session(
        self,
        session_name: str,
    ) -> Dict[str, Any]:
        """Kill sandbox session

        Args:
            session_name: str

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.kill_sandbox_session(session_name=session_name))

    def list_sandbox_sessions(self) -> InternalApiListSandboxSessionsResponseDict:
        """List sandbox sessions

        Returns:
            Dict with keys:
                - sessions (List[InternalApiSandboxSessionInfo])
                - total (int)
        """
        return _to_dict(self._api.list_sandbox_sessions())

    def list_sandboxes(
        self,
        agent_id: Optional[str] = None,
        job_id: Optional[str] = None,
        base_image: Optional[str] = None,
        state: Optional[str] = None,
        vm_name: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> VirshSandboxInternalRestListSandboxesResponseDict:
        """List sandboxes

        Args:
            agent_id: Optional[str]
            job_id: Optional[str]
            base_image: Optional[str]
            state: Optional[str]
            vm_name: Optional[str]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            Dict with keys:
                - sandboxes (List[VirshSandboxInternalRestSandboxInfo])
                - total (int)
        """
        return _to_dict(
            self._api.list_sandboxes(
                agent_id=agent_id,
                job_id=job_id,
                base_image=base_image,
                state=state,
                vm_name=vm_name,
                limit=limit,
                offset=offset,
            )
        )

    def publish_changes(
        self,
        id: str,
        job_id: Optional[str] = None,
        message: Optional[str] = None,
        reviewers: Optional[List[str]] = None,
    ) -> None:
        """Publish changes

        Args:
            id: str
            job_id: required
            message: optional commit/PR message
            reviewers: optional
        """
        request = VirshSandboxInternalRestPublishRequest(
            job_id=job_id,
            message=message,
            reviewers=reviewers,
        )
        return self._api.publish_changes(id=id, request=request)

    def run_sandbox_command(
        self,
        id: str,
        command: Optional[str] = None,
        env: Optional[Dict[str, str]] = None,
        private_key_path: Optional[str] = None,
        timeout_sec: Optional[int] = None,
        username: Optional[str] = None,
    ) -> VirshSandboxInternalRestRunCommandResponseDict:
        """Run command in sandbox

        Args:
            id: str
            command: required
            env: optional
            private_key_path: required; path on API host
            timeout_sec: optional; default from service config
            username: required

        Returns:
            Dict with keys:
                - command (VirshSandboxInternalStoreCommand)
        """
        request = VirshSandboxInternalRestRunCommandRequest(
            command=command,
            env=env,
            private_key_path=private_key_path,
            timeout_sec=timeout_sec,
            username=username,
        )
        return _to_dict(self._api.run_sandbox_command(id=id, request=request))

    def sandbox_api_health(self) -> Dict[str, Any]:
        """Check sandbox API health

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.sandbox_api_health())

    def start_sandbox(
        self,
        id: str,
        wait_for_ip: Optional[bool] = None,
    ) -> VirshSandboxInternalRestStartSandboxResponseDict:
        """Start sandbox

        Args:
            id: str
            wait_for_ip: optional; default false

        Returns:
            Dict with keys:
                - ip_address (str)
        """
        request = VirshSandboxInternalRestStartSandboxRequest(
            wait_for_ip=wait_for_ip,
        )
        return _to_dict(self._api.start_sandbox(id=id, request=request))


class TmuxOperations:
    """Wrapper for TmuxApi with simplified method signatures."""

    def __init__(self, api: TmuxApi):
        self._api = api

    def create_tmux_pane(
        self,
        command: Optional[str] = None,
        horizontal: Optional[bool] = None,
        new_window: Optional[bool] = None,
        session_name: Optional[str] = None,
        window_name: Optional[str] = None,
    ) -> TmuxClientInternalTypesCreatePaneResponseDict:
        """Create tmux pane

        Args:
            command: command
            horizontal: false = vertical split
            new_window: true = create new window instead of split
            session_name: session_name
            window_name: window_name

        Returns:
            Dict with keys:
                - pane_id (str)
                - pane_index (int)
                - session_name (str)
                - window_index (int)
        """
        request = TmuxClientInternalTypesCreatePaneRequest(
            command=command,
            horizontal=horizontal,
            new_window=new_window,
            session_name=session_name,
            window_name=window_name,
        )
        return _to_dict(self._api.create_tmux_pane(request=request))

    def create_tmux_session(self) -> Dict[str, Any]:
        """Create tmux session

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.create_tmux_session(request={}))

    def kill_tmux_pane(
        self,
        pane_id: str,
    ) -> Dict[str, Any]:
        """Kill tmux pane

        Args:
            pane_id: str

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.kill_tmux_pane(pane_id=pane_id))

    def kill_tmux_session(
        self,
        session_name: str,
    ) -> Dict[str, Any]:
        """Kill tmux session

        Args:
            session_name: str

        Returns:
            Dict with keys:
        """
        return _to_dict(self._api.kill_tmux_session(session_name=session_name))

    def list_tmux_panes(
        self,
        session: Optional[str] = None,
    ) -> TmuxClientInternalTypesListPanesResponseDict:
        """List tmux panes

        Args:
            session: Optional[str]

        Returns:
            Dict with keys:
                - panes (List[TmuxClientInternalTypesPaneInfo])
        """
        return _to_dict(self._api.list_tmux_panes(session=session))

    def list_tmux_sessions(self) -> List[TmuxClientInternalTypesSessionInfoDict]:
        """List tmux sessions

        Returns:
            List of dicts with keys:
                - attached (bool)
                - created (str)
                - id (str)
                - last_pane_x (int)
                - last_pane_y (int)
                - name (str)
                - windows (int)
        """
        return _to_dict(self._api.list_tmux_sessions())

    def list_tmux_windows(
        self,
        session: Optional[str] = None,
    ) -> List[TmuxClientInternalTypesWindowInfoDict]:
        """List tmux windows

        Args:
            session: Optional[str]

        Returns:
            List of dicts with keys:
                - active (bool)
                - height (int)
                - index (int)
                - name (str)
                - panes (int)
                - session_name (str)
                - width (int)
        """
        return _to_dict(self._api.list_tmux_windows(session=session))

    def read_tmux_pane(
        self,
        last_n_lines: Optional[int] = None,
        pane_id: Optional[str] = None,
    ) -> TmuxClientInternalTypesReadPaneResponseDict:
        """Read tmux pane

        Args:
            last_n_lines: 0 means all visible content
            pane_id: pane_id

        Returns:
            Dict with keys:
                - content (str)
                - lines (int)
                - pane_id (str)
        """
        request = TmuxClientInternalTypesReadPaneRequest(
            last_n_lines=last_n_lines,
            pane_id=pane_id,
        )
        return _to_dict(self._api.read_tmux_pane(request=request))

    def release_tmux_session(
        self,
        session_id: str,
    ) -> TmuxClientInternalTypesKillSessionResponseDict:
        """Release tmux session

        Args:
            session_id: str

        Returns:
            Dict with keys:
                - session_name (str)
                - success (bool)
        """
        return _to_dict(self._api.release_tmux_session(session_id=session_id))

    def send_keys_to_pane(
        self,
        key: Optional[str] = None,
        pane_id: Optional[str] = None,
    ) -> TmuxClientInternalTypesSendKeysResponseDict:
        """Send keys to tmux pane

        Args:
            key: Must be from approved list: \
            pane_id: pane_id

        Returns:
            Dict with keys:
                - pane_id (str)
                - sent (bool)
        """
        request = TmuxClientInternalTypesSendKeysRequest(
            key=key,
            pane_id=pane_id,
        )
        return _to_dict(self._api.send_keys_to_pane(request=request))

    def switch_tmux_pane(
        self,
        pane_id: Optional[str] = None,
    ) -> TmuxClientInternalTypesSwitchPaneResponseDict:
        """Switch tmux pane

        Args:
            pane_id: pane_id

        Returns:
            Dict with keys:
                - pane_id (str)
                - switched (bool)
        """
        request = TmuxClientInternalTypesSwitchPaneRequest(
            pane_id=pane_id,
        )
        return _to_dict(self._api.switch_tmux_pane(request=request))


class VMsOperations:
    """Wrapper for VMsApi with simplified method signatures."""

    def __init__(self, api: VMsApi):
        self._api = api

    def list_virtual_machines(self) -> VirshSandboxInternalRestListVMsResponseDict:
        """List all VMs

        Returns:
            Dict with keys:
                - vms (List[VirshSandboxInternalRestVmInfo])
        """
        return _to_dict(self._api.list_virtual_machines())


class VirshSandbox:
    """Unified client for the virsh-sandbox API.

    This class provides a single entry point for all virsh-sandbox API operations,
    with support for separate hosts for the main API and tmux API.
    All methods use flattened parameters instead of request objects.

    Args:
        host: Base URL for the main virsh-sandbox API
        tmux_host: Base URL for the tmux API (defaults to host)
        api_key: Optional API key for authentication
        verify_ssl: Whether to verify SSL certificates

    Example:
        >>> from virsh_sandbox import VirshSandbox
        >>> client = VirshSandbox()
        >>> client.sandbox.create_sandbox(source_vm_name="base-vm")
    """

    def __init__(
        self,
        host: str = "http://localhost:8080",
        tmux_host: Optional[str] = None,
        api_key: Optional[str] = None,
        access_token: Optional[str] = None,
        username: Optional[str] = None,
        password: Optional[str] = None,
        verify_ssl: bool = True,
        ssl_ca_cert: Optional[str] = None,
        retries: Optional[int] = None,
    ) -> None:
        """Initialize the VirshSandbox client."""
        self._main_config = Configuration(
            host=host,
            api_key={"Authorization": api_key} if api_key else None,
            access_token=access_token,
            username=username,
            password=password,
            ssl_ca_cert=ssl_ca_cert,
            retries=retries,
        )
        self._main_config.verify_ssl = verify_ssl
        self._main_api_client = ApiClient(configuration=self._main_config)

        tmux_host = tmux_host or host
        if tmux_host != host:
            self._tmux_config = Configuration(
                host=tmux_host,
                api_key={"Authorization": api_key} if api_key else None,
                access_token=access_token,
                username=username,
                password=password,
                ssl_ca_cert=ssl_ca_cert,
                retries=retries,
            )
            self._tmux_config.verify_ssl = verify_ssl
            self._tmux_api_client = ApiClient(configuration=self._tmux_config)
        else:
            self._tmux_config = self._main_config
            self._tmux_api_client = self._main_api_client

        self._access: Optional[AccessOperations] = None
        self._ansible: Optional[AnsibleOperations] = None
        self._audit: Optional[AuditOperations] = None
        self._command: Optional[CommandOperations] = None
        self._file: Optional[FileOperations] = None
        self._health: Optional[HealthOperations] = None
        self._human: Optional[HumanOperations] = None
        self._plan: Optional[PlanOperations] = None
        self._sandbox: Optional[SandboxOperations] = None
        self._tmux: Optional[TmuxOperations] = None
        self._vms: Optional[VMsOperations] = None

    @property
    def access(self) -> AccessOperations:
        """Access AccessApi operations."""
        if self._access is None:
            api = AccessApi(api_client=self._main_api_client)
            self._access = AccessOperations(api)
        return self._access

    @property
    def ansible(self) -> AnsibleOperations:
        """Access AnsibleApi operations."""
        if self._ansible is None:
            api = AnsibleApi(api_client=self._main_api_client)
            self._ansible = AnsibleOperations(api)
        return self._ansible

    @property
    def audit(self) -> AuditOperations:
        """Access AuditApi operations."""
        if self._audit is None:
            api = AuditApi(api_client=self._tmux_api_client)
            self._audit = AuditOperations(api)
        return self._audit

    @property
    def command(self) -> CommandOperations:
        """Access CommandApi operations."""
        if self._command is None:
            api = CommandApi(api_client=self._tmux_api_client)
            self._command = CommandOperations(api)
        return self._command

    @property
    def file(self) -> FileOperations:
        """Access FileApi operations."""
        if self._file is None:
            api = FileApi(api_client=self._tmux_api_client)
            self._file = FileOperations(api)
        return self._file

    @property
    def health(self) -> HealthOperations:
        """Access HealthApi operations."""
        if self._health is None:
            api = HealthApi(api_client=self._tmux_api_client)
            self._health = HealthOperations(api)
        return self._health

    @property
    def human(self) -> HumanOperations:
        """Access HumanApi operations."""
        if self._human is None:
            api = HumanApi(api_client=self._tmux_api_client)
            self._human = HumanOperations(api)
        return self._human

    @property
    def plan(self) -> PlanOperations:
        """Access PlanApi operations."""
        if self._plan is None:
            api = PlanApi(api_client=self._tmux_api_client)
            self._plan = PlanOperations(api)
        return self._plan

    @property
    def sandbox(self) -> SandboxOperations:
        """Access SandboxApi operations."""
        if self._sandbox is None:
            api = SandboxApi(api_client=self._main_api_client)
            self._sandbox = SandboxOperations(api)
        return self._sandbox

    @property
    def tmux(self) -> TmuxOperations:
        """Access TmuxApi operations."""
        if self._tmux is None:
            api = TmuxApi(api_client=self._tmux_api_client)
            self._tmux = TmuxOperations(api)
        return self._tmux

    @property
    def vms(self) -> VMsOperations:
        """Access VMsApi operations."""
        if self._vms is None:
            api = VMsApi(api_client=self._main_api_client)
            self._vms = VMsOperations(api)
        return self._vms

    @property
    def configuration(self) -> Configuration:
        """Get the main API configuration."""
        return self._main_config

    @property
    def tmux_configuration(self) -> Configuration:
        """Get the tmux API configuration."""
        return self._tmux_config

    def set_debug(self, debug: bool) -> None:
        """Enable or disable debug mode."""
        self._main_config.debug = debug
        if self._tmux_config is not self._main_config:
            self._tmux_config.debug = debug

    def close(self) -> None:
        """Close the API client connections."""
        if hasattr(self._main_api_client.rest_client, "close"):
            self._main_api_client.rest_client.close()
        if self._tmux_api_client is not self._main_api_client:
            if hasattr(self._tmux_api_client.rest_client, "close"):
                self._tmux_api_client.rest_client.close()

    def __enter__(self) -> "VirshSandbox":
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        self.close()
