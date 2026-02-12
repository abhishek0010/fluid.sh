#!/bin/bash
# run-on-remotes.sh
#
# Copies and executes a specified local script on multiple remote hosts.
# The local script is copied to /tmp/ on the remote machine and executed with sudo.
#
# Reads hosts from the HOSTS env var (newline-separated). Each line can be:
#   user:password@host  - per-host credentials (password auth via sshpass)
#   user@host           - key-based auth
#   host                - key-based auth
#
# Usage: HOSTS="root:pass@1.2.3.4" ./run-on-remotes.sh <SCRIPT_PATH> [SSH_USERS_FILE]
#
# Environment:
#   HOSTS            (Required) Newline-separated host entries.
#   SSH_KEY          (Optional) Path to SSH private key for key-based auth.
#
# Arguments:
#   SCRIPT_PATH      Path to the local script to execute remotely.
#   SSH_USERS_FILE   (Optional) Path to ssh-users.conf to copy and pass to the script.

set -u

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Check arguments
if [[ $# -lt 1 ]] || [[ $# -gt 2 ]]; then
    echo "Usage: HOSTS='user:pass@host' $0 <SCRIPT_PATH> [SSH_USERS_FILE]"
    exit 1
fi

SCRIPT_PATH="$1"
SSH_USERS_FILE="${2:-}"

# HOSTS env var is required
if [[ -z "${HOSTS:-}" ]]; then
    log_error "HOSTS env var is required (newline-separated user:password@host entries)"
    exit 1
fi

HOSTS_FILE=$(mktemp)
trap 'rm -f "$HOSTS_FILE"' EXIT INT TERM
printf '%s\n' "$HOSTS" > "$HOSTS_FILE"
log_info "Loaded $(grep -c '[^[:space:]]' "$HOSTS_FILE") host(s) from HOSTS env var"

if [[ ! -f "$SCRIPT_PATH" ]]; then
    log_error "Script file not found: $SCRIPT_PATH"
    exit 1
fi

if [[ -n "$SSH_USERS_FILE" ]] && [[ ! -f "$SSH_USERS_FILE" ]]; then
    log_error "SSH users file not found: $SSH_USERS_FILE"
    exit 1
fi

SCRIPT_NAME=$(basename "$SCRIPT_PATH")
REMOTE_DEST="/tmp/$SCRIPT_NAME"

log_info "Deploying $SCRIPT_NAME to remote hosts..."

COUNT=1

# Loop through each line in the hosts file
while IFS= read -r HOST <&3 || [[ -n "$HOST" ]]; do
    # Skip empty lines and comments (lines starting with #)
    [[ -z "$HOST" ]] && continue
    [[ "$HOST" =~ ^#.*$ ]] && continue

    # Parse per-host credentials. Supported line formats:
    #   user:password@host  - password auth via sshpass
    #   user@host           - key-based auth
    #   host                - key-based auth
    HOST_USER=""
    HOST_PASS=""
    HOST_ADDR=""

    if [[ "$HOST" == *:*@* ]]; then
        HOST_USER="${HOST%%:*}"
        local_remainder="${HOST#*:}"
        HOST_PASS="${local_remainder%%@*}"
        HOST_ADDR="${local_remainder#*@}"
    elif [[ "$HOST" == *@* ]]; then
        HOST_USER="${HOST%%@*}"
        HOST_ADDR="${HOST#*@}"
    else
        HOST_ADDR="$HOST"
    fi

    # Build SSH target
    if [[ -n "$HOST_USER" ]]; then
        TARGET="${HOST_USER}@${HOST_ADDR}"
    else
        TARGET="$HOST_ADDR"
    fi

    # Build per-host SSH/SCP commands
    SSH_OPTS="-o ConnectTimeout=5 -o StrictHostKeyChecking=accept-new"
    if [[ -n "${SSH_KEY:-}" ]]; then
        SSH_OPTS="$SSH_OPTS -i $SSH_KEY"
    fi
    SCP_CMD="scp"
    SSH_CMD="ssh"

    if [[ -n "$HOST_PASS" ]]; then
        if ! command -v sshpass &> /dev/null; then
            log_error "sshpass is required for password-based SSH but is not installed"
            exit 1
        fi
        export SSHPASS="$HOST_PASS"
        SCP_CMD="sshpass -e scp"
        SSH_CMD="sshpass -e ssh"
    fi

    echo ""
    echo "----------------------------------------------------------------------------"
    log_info "Processing host: $TARGET (Index: $COUNT)"
    echo "----------------------------------------------------------------------------"

    # 1. Copy the script
    log_info "Copying script to $TARGET:$REMOTE_DEST..."
    if $SCP_CMD $SSH_OPTS "$SCRIPT_PATH" "${TARGET}:${REMOTE_DEST}"; then
        log_success "Script copied successfully."
    else
        log_error "Failed to copy script to $TARGET. Skipping..."
        continue
    fi

    # 1b. Copy SSH users file if provided
    REMOTE_USERS_FILE="/tmp/ssh-users.conf"
    EXTRA_ARGS=""
    if [[ -n "$SSH_USERS_FILE" ]]; then
        log_info "Copying SSH users file to $TARGET:$REMOTE_USERS_FILE..."
        if $SCP_CMD $SSH_OPTS "$SSH_USERS_FILE" "${TARGET}:${REMOTE_USERS_FILE}"; then
            log_success "SSH users file copied."
            EXTRA_ARGS="--ssh-users-file $REMOTE_USERS_FILE"
        else
            log_warn "Failed to copy SSH users file to $TARGET. Continuing without it."
        fi
    fi

    # 2. Make executable
    log_info "Setting executable permissions..."
    if $SSH_CMD $SSH_OPTS "$TARGET" "chmod +x $REMOTE_DEST"; then
         log_success "Permissions set."
    else
        log_error "Failed to set permissions on $TARGET. Skipping..."
        continue
    fi

    # 3. Execute with sudo
    log_info "Executing script (sudo required)..."
    if $SSH_CMD -t $SSH_OPTS "$TARGET" "sudo $REMOTE_DEST $COUNT $EXTRA_ARGS"; then
        log_success "Script execution completed successfully on $TARGET."
    else
        log_error "Script execution failed on $TARGET."
    fi

    # Clear per-host password so it doesn't leak to the next iteration
    unset SSHPASS

    ((COUNT++))

done 3< "$HOSTS_FILE"

echo ""
echo "============================================================================"
log_info "Batch execution finished."
echo "============================================================================"
