#!/bin/bash
# run-on-remotes.sh
#
# Copies and executes a specified local script on multiple remote hosts.
# The local script is copied to /tmp/ on the remote machine and executed with sudo.
#
# Usage: ./run-on-remotes.sh <HOSTS_FILE> <SCRIPT_PATH>
#
# Arguments:
#   HOSTS_FILE   Path to a text file containing one "user@host" per line.
#   SCRIPT_PATH  Path to the local script to execute remotely.

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
if [[ $# -ne 2 ]]; then
    echo "Usage: $0 <HOSTS_FILE> <SCRIPT_PATH>"
    echo "Example: $0 hosts.txt ./setup-ubuntu.sh"
    exit 1
fi

HOSTS_FILE="$1"
SCRIPT_PATH="$2"

# Validate inputs
if [[ ! -f "$HOSTS_FILE" ]]; then
    log_error "Hosts file not found: $HOSTS_FILE"
    exit 1
fi

if [[ ! -f "$SCRIPT_PATH" ]]; then
    log_error "Script file not found: $SCRIPT_PATH"
    exit 1
fi

SCRIPT_NAME=$(basename "$SCRIPT_PATH")
REMOTE_DEST="/tmp/$SCRIPT_NAME"

log_info "Deploying $SCRIPT_NAME to hosts listed in $HOSTS_FILE..."

COUNT=1

# Loop through each line in the hosts file
while IFS= read -r HOST <&3 || [[ -n "$HOST" ]]; do
    # Skip empty lines and comments (lines starting with #)
    [[ -z "$HOST" ]] && continue
    [[ "$HOST" =~ ^#.*$ ]] && continue

    echo ""
    echo "----------------------------------------------------------------------------"
    log_info "Processing host: $HOST (Index: $COUNT)"
    echo "----------------------------------------------------------------------------"

    # 1. Copy the script
    log_info "Copying script to $HOST:$REMOTE_DEST..."
    if scp -o ConnectTimeout=5 "$SCRIPT_PATH" "${HOST}:${REMOTE_DEST}"; then
        log_success "Script copied successfully."
    else
        log_error "Failed to copy script to $HOST. Skipping..."
        continue
    fi

    # 2. Make executable
    log_info "Setting executable permissions..."
    if ssh -o ConnectTimeout=5 "$HOST" "chmod +x $REMOTE_DEST"; then
         log_success "Permissions set."
    else
        log_error "Failed to set permissions on $HOST. Skipping..."
        continue
    fi

    # 3. Execute with sudo
    log_info "Executing script (sudo required)..."
    # We use -t to force pseudo-terminal allocation for sudo prompts if needed
    # Pass the COUNT as the first argument to the script
    if ssh -t -o ConnectTimeout=5 "$HOST" "sudo $REMOTE_DEST $COUNT"; then
        log_success "Script execution completed successfully on $HOST."
        
        # Optional: Cleanup
        # ssh "$HOST" "rm $REMOTE_DEST"
    else
        log_error "Script execution failed on $HOST."
    fi

    ((COUNT++))

done 3< "$HOSTS_FILE"

echo ""
echo "============================================================================"
log_info "Batch execution finished."
echo "============================================================================"
