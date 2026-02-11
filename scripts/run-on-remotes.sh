#!/bin/bash
# run-on-remotes.sh
#
# Copies and executes a specified local script on multiple remote hosts.
# The local script is copied to /tmp/ on the remote machine and executed with sudo.
#
# Usage: ./run-on-remotes.sh <HOSTS_FILE> <SCRIPT_PATH> [SSH_USERS_FILE]
#
# Arguments:
#   HOSTS_FILE       Path to a text file containing one "user@host" per line.
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
if [[ $# -lt 2 ]] || [[ $# -gt 3 ]]; then
    echo "Usage: $0 <HOSTS_FILE> <SCRIPT_PATH> [SSH_USERS_FILE]"
    echo "Example: $0 hosts.txt ./setup-ubuntu.sh ./ssh-users.conf"
    exit 1
fi

HOSTS_FILE="$1"
SCRIPT_PATH="$2"
SSH_USERS_FILE="${3:-}"

# Validate inputs
if [[ ! -f "$HOSTS_FILE" ]]; then
    log_error "Hosts file not found: $HOSTS_FILE"
    exit 1
fi

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

# SSH credential overrides from environment variables
# SSH_PASSWORD: if set, use sshpass for password-based authentication
# SSH_USER: if set, override the user portion of host entries in hosts.txt
SSH_PASS="${SSH_PASSWORD:-}"
SSH_USER_OVERRIDE="${SSH_USER:-}"

# Build SSH/SCP command prefixes
SSH_OPTS="-o ConnectTimeout=5"
if [[ -n "$SSH_PASS" ]]; then
    # Verify sshpass is installed
    if ! command -v sshpass >/dev/null 2>&1; then
        log_error "SSH_PASSWORD is set but 'sshpass' is not installed."
        log_error "Please install sshpass to use password-based authentication:"
        log_error "  - Ubuntu/Debian: apt-get install sshpass"
        log_error "  - macOS (Homebrew): brew install hudochenkov/sshpass/sshpass"
        log_error "  - macOS (MacPorts): port install sshpass"
        log_error "  - RHEL/CentOS: yum install sshpass"
        exit 1
    fi
    export SSHPASS="$SSH_PASS"
    SSH_OPTS="$SSH_OPTS -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
    SCP_CMD="sshpass -p \"$SSH_PASS\" scp $SSH_OPTS"
    SSH_CMD="sshpass -p \"$SSH_PASS\" ssh $SSH_OPTS"
    log_info "Using password-based SSH authentication (via sshpass)"
else
    SCP_CMD="scp $SSH_OPTS"
    SSH_CMD="ssh $SSH_OPTS"
fi

log_info "Deploying $SCRIPT_NAME to hosts listed in $HOSTS_FILE..."

COUNT=1

# Loop through each line in the hosts file
while IFS= read -r HOST <&3 || [[ -n "$HOST" ]]; do
    # Skip empty lines and comments (lines starting with #)
    [[ -z "$HOST" ]] && continue
    [[ "$HOST" =~ ^#.*$ ]] && continue

    # Override user if SSH_USER is set
    if [[ -n "$SSH_USER_OVERRIDE" ]]; then
        # Extract hostname portion (strip user@ prefix if present)
        HOST_ADDR="${HOST#*@}"
        HOST="${SSH_USER_OVERRIDE}@${HOST_ADDR}"
    fi

    echo ""
    echo "----------------------------------------------------------------------------"
    log_info "Processing host: $HOST (Index: $COUNT)"
    echo "----------------------------------------------------------------------------"

    # 1. Copy the script
    log_info "Copying script to $HOST:$REMOTE_DEST..."
    if $SCP_CMD "$SCRIPT_PATH" "${HOST}:${REMOTE_DEST}"; then
        log_success "Script copied successfully."
    else
        log_error "Failed to copy script to $HOST. Skipping..."
        continue
    fi

    # 1b. Copy SSH users file if provided
    REMOTE_USERS_FILE="/tmp/ssh-users.conf"
    EXTRA_ARGS=""
    if [[ -n "$SSH_USERS_FILE" ]]; then
        log_info "Copying SSH users file to $HOST:$REMOTE_USERS_FILE..."
        if $SCP_CMD "$SSH_USERS_FILE" "${HOST}:${REMOTE_USERS_FILE}"; then
            log_success "SSH users file copied."
            EXTRA_ARGS="--ssh-users-file $REMOTE_USERS_FILE"
        else
            log_warn "Failed to copy SSH users file to $HOST. Continuing without it."
        fi
    fi

    # 2. Make executable
    log_info "Setting executable permissions..."
    if $SSH_CMD "$HOST" "chmod +x $REMOTE_DEST"; then
         log_success "Permissions set."
    else
        log_error "Failed to set permissions on $HOST. Skipping..."
        continue
    fi

    # 3. Execute with sudo
    log_info "Executing script (sudo required)..."
    # We use -t to force pseudo-terminal allocation for sudo prompts if needed
    # Pass the COUNT as the first argument to the script
    if $SSH_CMD -t "$HOST" "sudo $REMOTE_DEST $COUNT $EXTRA_ARGS"; then
        log_success "Script execution completed successfully on $HOST."

        # Optional: Cleanup
        # $SSH_CMD "$HOST" "rm $REMOTE_DEST"
    else
        log_error "Script execution failed on $HOST."
    fi

    ((COUNT++))

done 3< "$HOSTS_FILE"

echo ""
echo "============================================================================"
log_info "Batch execution finished."
echo "============================================================================"
