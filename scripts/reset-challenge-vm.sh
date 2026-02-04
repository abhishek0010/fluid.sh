#!/bin/bash
# reset-challenge-vm.sh
#
# Resets a single challenge VM by its ID.
# Destroys the specified VM and creates a fresh one with a new random ID.
#
# Usage: sudo ./reset-challenge-vm.sh <vm-id>
#   vm-id: The VM ID to reset (e.g., vm-8x92j)
#
# Options:
#   --keep-id    Keep the same VM ID instead of generating a new one
#   --help       Show this help message

set -euo pipefail

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

show_help() {
    echo "Usage: sudo ./reset-challenge-vm.sh <vm-id> [OPTIONS]"
    echo ""
    echo "Reset a single challenge VM by destroying it and creating a fresh one."
    echo ""
    echo "Arguments:"
    echo "  vm-id         The VM ID to reset (e.g., vm-8x92j)"
    echo ""
    echo "Options:"
    echo "  --keep-id     Keep the same VM ID instead of generating a new one"
    echo "  --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  sudo ./reset-challenge-vm.sh vm-8x92j           # Reset with new random ID"
    echo "  sudo ./reset-challenge-vm.sh vm-8x92j --keep-id # Reset keeping same ID"
    echo ""
}

# Generate a random VM ID like vm-8x92j
generate_vm_id() {
    local chars="abcdefghijklmnopqrstuvwxyz0123456789"
    local id="vm-"
    for i in {1..5}; do
        id+="${chars:RANDOM%${#chars}:1}"
    done
    echo "$id"
}

# Parse arguments
VM_ID=""
KEEP_ID=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --keep-id)
            KEEP_ID=true
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
        *)
            if [[ -z "$VM_ID" ]]; then
                VM_ID="$1"
            else
                log_error "Unexpected argument: $1"
                show_help
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate VM ID provided
if [[ -z "$VM_ID" ]]; then
    log_error "VM ID is required"
    show_help
    exit 1
fi

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

# Check for required commands
if ! command -v virsh &> /dev/null || ! command -v virt-install &> /dev/null; then
    log_error "Required commands (virsh, virt-install) not found."
    log_error "Please run setup-challenge-ubuntu.sh first."
    exit 1
fi

# Check if VM exists
if ! virsh dominfo "$VM_ID" &>/dev/null; then
    log_error "VM '$VM_ID' does not exist."
    log_info "Available VMs:"
    virsh list --all --name | grep -v "^$" | sed 's/^/  /'
    exit 1
fi

log_info "Resetting VM: $VM_ID"
if [[ "$KEEP_ID" == "true" ]]; then
    log_info "Will keep the same VM ID"
else
    log_info "Will generate a new random VM ID"
fi

# ============================================================================
# STEP 1: Ensure default network is active
# ============================================================================
log_info "Ensuring default network is active..."

if ! virsh net-list | grep -q "default.*active"; then
    log_info "Starting default network..."
    virsh net-start default || true
fi

# ============================================================================
# STEP 2: Destroy and cleanup the specified VM
# ============================================================================
log_info "Destroying VM '$VM_ID'..."

# Get VM info before destroying (for reference)
OLD_MAC=$(virsh domiflist "$VM_ID" 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1 || true)
OLD_IP=$(virsh domifaddr "$VM_ID" --source lease 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)

if [[ -n "$OLD_IP" ]]; then
    log_info "Old IP was: $OLD_IP"
fi
if [[ -n "$OLD_MAC" ]]; then
    log_info "Old MAC was: $OLD_MAC"
fi

# Destroy (stop) if running
virsh destroy "$VM_ID" > /dev/null 2>&1 || true

# Undefine
virsh undefine "$VM_ID" --nvram > /dev/null 2>&1 || virsh undefine "$VM_ID" > /dev/null 2>&1 || true

# Clean up disk and cloud-init
IMAGE_DIR="/var/lib/libvirt/images"
CLOUD_INIT_DIR="${IMAGE_DIR}/cloud-init"

rm -f "${IMAGE_DIR}/${VM_ID}.qcow2" 2>/dev/null || true
rm -rf "${CLOUD_INIT_DIR}/${VM_ID}" 2>/dev/null || true

log_success "VM '$VM_ID' destroyed and cleaned up."

# ============================================================================
# STEP 3: Determine new VM ID
# ============================================================================
if [[ "$KEEP_ID" == "true" ]]; then
    NEW_VM_ID="$VM_ID"
else
    NEW_VM_ID=$(generate_vm_id)
    # Ensure unique
    while virsh dominfo "$NEW_VM_ID" &>/dev/null || [[ -f "${IMAGE_DIR}/${NEW_VM_ID}.qcow2" ]]; do
        NEW_VM_ID=$(generate_vm_id)
    done
    log_info "New VM ID: $NEW_VM_ID"
fi

# ============================================================================
# STEP 4: Check base image exists
# ============================================================================
BASE_IMAGE="ubuntu-22.04-minimal-cloudimg-amd64.img"
BASE_IMAGE_PATH="${IMAGE_DIR}/${BASE_IMAGE}"

if [[ ! -f "$BASE_IMAGE_PATH" ]]; then
    log_info "Base image not found. Downloading..."
    BASE_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/${BASE_IMAGE}"
    if wget -q --show-progress -O "$BASE_IMAGE_PATH" "$BASE_IMAGE_URL"; then
        log_success "Image downloaded."
    else
        log_error "Failed to download base image."
        exit 1
    fi
fi

# ============================================================================
# STEP 5: Create the new VM
# ============================================================================
log_info "Creating new challenge VM '$NEW_VM_ID'..."
log_info "  - RAM: 512 MB"
log_info "  - vCPUs: 1"
log_info "  - Disk: 10 GB"

# Create VM disk
VM_DISK="${IMAGE_DIR}/${NEW_VM_ID}.qcow2"
qemu-img create -f qcow2 -F qcow2 -b "$BASE_IMAGE_PATH" "$VM_DISK" 10G

# Create Cloud-Init Config
SEED_DIR="${CLOUD_INIT_DIR}/${NEW_VM_ID}"
mkdir -p "$SEED_DIR"

USER_DATA="${SEED_DIR}/user-data"
META_DATA="${SEED_DIR}/meta-data"
NETWORK_CONFIG="${SEED_DIR}/network-config"

INSTANCE_ID="${NEW_VM_ID}-$(date +%s)-${RANDOM}"

# User-data with Apache installation
cat > "$USER_DATA" <<EOF
#cloud-config
# Enable root login with password
disable_root: false
ssh_pwauth: True

# Set root password
chpasswd:
  expire: False
  users:
    - name: root
      password: root
      type: text

# Install packages
packages:
  - qemu-guest-agent
  - apache2

# Enable root SSH login and services
runcmd:
  # Enable root SSH login
  - sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
  - sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config
  - systemctl restart sshd || systemctl restart ssh
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
  - systemctl enable apache2
  - systemctl start apache2
  - |
    cat > /var/www/html/index.html <<'HTMLEOF'
<!DOCTYPE html>
<html>
<head>
    <title>Challenge VM: ${NEW_VM_ID}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); max-width: 600px; margin: 0 auto; }
        h1 { color: #333; }
        .info { background: #e8f5e9; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .vm-id { font-family: monospace; font-size: 1.2em; color: #1976d2; }
        code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Challenge VM</h1>
        <div class="info">
            <p><strong>VM ID:</strong> <span class="vm-id">${NEW_VM_ID}</span></p>
            <p><strong>Status:</strong> Running</p>
        </div>
        <p>This is a challenge VM running Apache HTTP Server.</p>
        <p>Edit <code>/var/www/html/index.html</code> to customize this page.</p>
    </div>
</body>
</html>
HTMLEOF
  - chown -R www-data:www-data /var/www/html
  - chmod 644 /var/www/html/index.html

EOF

cat > "$META_DATA" <<EOF
instance-id: ${INSTANCE_ID}
local-hostname: ${NEW_VM_ID}
EOF

# Network-config - use 'name: en*' for reliable interface matching
cat > "$NETWORK_CONFIG" <<EOF
version: 2
ethernets:
  id0:
    match:
      name: en*
    dhcp4: true
EOF

log_success "Cloud-init config files created."

# Generate random MAC address
MAC_SUFFIX=$(printf '%02x:%02x:%02x' $((RANDOM%256)) $((RANDOM%256)) $((RANDOM%256)))
MAC_ADDRESS="52:54:00:${MAC_SUFFIX}"

log_info "Using MAC address: ${MAC_ADDRESS}"

# Create VM using virt-install's native --cloud-init
log_info "Booting VM with virt-install --cloud-init..."

virt-install \
    --name "${NEW_VM_ID}" \
    --memory 512 \
    --vcpus 1 \
    --disk "${VM_DISK},device=disk,bus=virtio" \
    --cloud-init user-data="${USER_DATA}",meta-data="${META_DATA}",network-config="${NETWORK_CONFIG}" \
    --os-variant ubuntu22.04 \
    --import \
    --noautoconsole \
    --graphics none \
    --console pty,target_type=serial \
    --network network=default,model=virtio,mac="${MAC_ADDRESS}"

log_success "VM '${NEW_VM_ID}' created!"

# ============================================================================
# STEP 6: Wait for IP address
# ============================================================================
log_info "Waiting for VM to obtain IP address..."

MAX_WAIT=180
WAIT_INTERVAL=5
ELAPSED=0
VM_IP=""

while [[ $ELAPSED -lt $MAX_WAIT ]]; do
    VM_IP=$(virsh domifaddr "${NEW_VM_ID}" --source lease 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)

    if [[ -n "$VM_IP" ]]; then
        log_success "VM '${NEW_VM_ID}' obtained IP: ${VM_IP}"
        break
    fi

    sleep $WAIT_INTERVAL
    ELAPSED=$((ELAPSED + WAIT_INTERVAL))

    if (( ELAPSED % 30 == 0 )); then
        log_info "Still waiting... (${ELAPSED}s / ${MAX_WAIT}s)"
    fi
done

if [[ -z "$VM_IP" ]]; then
    log_warn "VM did not obtain IP within ${MAX_WAIT} seconds."
fi

# Get MAC address
VM_MAC=$(virsh domiflist "${NEW_VM_ID}" 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1 || true)

# ============================================================================
# STEP 7: Verify HTTP accessibility
# ============================================================================
HTTP_STATUS="UNKNOWN"

if [[ -n "$VM_IP" ]]; then
    log_info "Waiting for Apache to start..."
    sleep 10

    log_info "Checking HTTP accessibility..."

    for attempt in {1..6}; do
        HTTP_RESPONSE=$(curl -s --connect-timeout 5 --max-time 10 "http://${VM_IP}/" 2>/dev/null || true)

        if [[ -n "$HTTP_RESPONSE" ]] && echo "$HTTP_RESPONSE" | grep -q "Challenge VM"; then
            log_success "HTTP server is accessible!"
            HTTP_STATUS="OK"
            break
        fi

        log_info "HTTP not ready yet, retrying... (attempt $attempt/6)"
        sleep 10
    done

    if [[ "$HTTP_STATUS" != "OK" ]]; then
        log_warn "HTTP server is not accessible yet"
        HTTP_STATUS="FAILED"
    fi
else
    HTTP_STATUS="NO_IP"
fi

# ============================================================================
# STEP 8: Update summary file
# ============================================================================
SUMMARY_FILE="${IMAGE_DIR}/challenge-vms.txt"

if [[ -f "$SUMMARY_FILE" ]]; then
    # Remove old entry for the original VM ID
    grep -v "^${VM_ID} " "$SUMMARY_FILE" > "${SUMMARY_FILE}.tmp" 2>/dev/null || true
    mv "${SUMMARY_FILE}.tmp" "$SUMMARY_FILE"
fi

# Add new entry
echo "${NEW_VM_ID} ${VM_IP:-N/A} ${VM_MAC:-N/A} ${HTTP_STATUS}" >> "$SUMMARY_FILE"

# ============================================================================
# STEP 9: Final Summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "VM Reset Complete!"
echo "============================================================================"
echo ""

if [[ "$KEEP_ID" == "true" ]]; then
    echo "Reset Summary:"
    echo "  - VM ID: ${NEW_VM_ID} (kept same ID)"
else
    echo "Reset Summary:"
    echo "  - Old VM ID: ${VM_ID}"
    echo "  - New VM ID: ${NEW_VM_ID}"
fi

echo ""
echo "New VM Details:"
echo "  - Disk: ${VM_DISK}"
echo "  - MAC Address: ${VM_MAC:-N/A}"
echo "  - IP Address: ${VM_IP:-N/A}"
echo "  - HTTP Status: ${HTTP_STATUS}"

if [[ "$HTTP_STATUS" == "OK" ]] && [[ -n "$VM_IP" ]]; then
    echo "  - URL: http://${VM_IP}/"
fi

echo "  - Login: root / root"
echo ""
echo "VM Specifications:"
echo "  - RAM: 512 MB"
echo "  - vCPUs: 1"
echo "  - Disk: 10 GB"
echo "  - Web Server: Apache (httpd)"
echo ""
echo "Useful commands:"
echo "  virsh console ${NEW_VM_ID}                     # Access VM console"
echo "  virsh domifaddr ${NEW_VM_ID} --source lease    # Get VM IP"
echo "  curl http://${VM_IP:-<IP>}/                    # Test HTTP server"
echo "  ssh root@${VM_IP:-<IP>}                         # SSH to VM"
echo ""

if [[ "$HTTP_STATUS" == "OK" ]]; then
    log_success "VM '${NEW_VM_ID}' is ready!"
else
    log_warn "VM may need more time to fully initialize."
    log_warn "Check: virsh console ${NEW_VM_ID}"
fi
