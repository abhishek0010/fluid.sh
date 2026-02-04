#!/bin/bash
# reset-challenge-ubuntu.sh
#
# Resets the Ubuntu host by destroying ALL VMs and creating fresh challenge VMs.
# Challenge VMs are minimal (512MB RAM, 1 vCPU) with httpd serving a test page.
#
# Usage: sudo ./reset-challenge-ubuntu.sh [COUNT]
#   COUNT: Number of challenge VMs to create (default: 1)

VM_COUNT=${1:-1}

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

# Generate a random VM ID like vm-8x92j
generate_vm_id() {
    local chars="abcdefghijklmnopqrstuvwxyz0123456789"
    local id="vm-"
    for i in {1..5}; do
        id+="${chars:RANDOM%${#chars}:1}"
    done
    echo "$id"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

# Check for required commands
if ! command -v virsh &> /dev/null || ! command -v virt-install &> /dev/null; then
    log_error "Required commands (virsh, virt-install) not found."
    log_error "Please run setup-challenge-ubuntu.sh first to install dependencies."
    exit 1
fi

log_warn "This script will DESTROY ALL VMs on this host and create ${VM_COUNT} fresh challenge VM(s)."

# ============================================================================
# STEP 1: Ensure default network is active
# ============================================================================
log_info "Ensuring default network is active..."

if ! virsh net-info default &>/dev/null; then
    log_info "Default network not found, creating it..."
    virsh net-define /usr/share/libvirt/networks/default.xml || true
fi

if ! virsh net-list | grep -q "default.*active"; then
    log_info "Starting default network..."
    virsh net-start default || true
    virsh net-autostart default || true
fi

log_success "Default network is active."

if ! virsh net-dumpxml default | grep -q "<dhcp>"; then
    log_warn "Default network does not have DHCP configured!"
fi

# ============================================================================
# STEP 2: Destroy and Undefine ALL VMs
# ============================================================================
log_info "Cleaning up ALL existing VMs..."

VMS=$(virsh list --all --name 2>/dev/null || true)

for VM in $VMS; do
    if [[ -n "$VM" ]]; then
        log_info "Removing VM: $VM"
        virsh destroy "$VM" > /dev/null 2>&1 || true
        virsh undefine "$VM" --nvram > /dev/null 2>&1 || virsh undefine "$VM" > /dev/null 2>&1 || true
    fi
done

# Clean up old cloud-init directories
log_info "Cleaning up old cloud-init data..."
rm -rf /var/lib/libvirt/images/cloud-init/* 2>/dev/null || true

# Clean up old VM disks (keep base images)
log_info "Cleaning up old VM disks..."
rm -f /var/lib/libvirt/images/vm-*.qcow2 2>/dev/null || true
rm -f /var/lib/libvirt/images/test-vm-*.qcow2 2>/dev/null || true
rm -f /var/lib/libvirt/images/sbx-*.qcow2 2>/dev/null || true

# Clean up sandbox work directories
rm -rf /var/lib/libvirt/images/sandboxes/* 2>/dev/null || true

# Clean up old summary file
rm -f /var/lib/libvirt/images/challenge-vms.txt 2>/dev/null || true

log_success "Cleanup complete."

# ============================================================================
# STEP 3: Ensure Caddy is installed and running
# ============================================================================
log_info "Checking Caddy installation on host..."

if ! command -v caddy &>/dev/null; then
    log_info "Installing Caddy on host..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg 2>/dev/null || true
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list > /dev/null
    apt-get update -qq
    apt-get install -y -qq caddy
    log_success "Caddy installed."
else
    log_info "Caddy already installed."
fi

systemctl enable caddy || true
systemctl restart caddy || true
log_success "Caddy is running."

# ============================================================================
# STEP 4: Download Base Image if needed
# ============================================================================
IMAGE_DIR="/var/lib/libvirt/images"
CLOUD_INIT_DIR="${IMAGE_DIR}/cloud-init"
BASE_IMAGE="ubuntu-22.04-minimal-cloudimg-amd64.img"
BASE_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/${BASE_IMAGE}"
BASE_IMAGE_PATH="${IMAGE_DIR}/${BASE_IMAGE}"

mkdir -p "$IMAGE_DIR"
mkdir -p "$CLOUD_INIT_DIR"

if [[ ! -f "$BASE_IMAGE_PATH" ]]; then
    log_info "Downloading Ubuntu Minimal Cloud Image (approx 300MB)..."
    if wget -q --show-progress -O "$BASE_IMAGE_PATH" "$BASE_IMAGE_URL"; then
        log_success "Image downloaded."
    else
        log_error "Failed to download image from $BASE_IMAGE_URL"
        exit 1
    fi
else
    log_info "Base image already exists at $BASE_IMAGE_PATH"
fi

# ============================================================================
# STEP 5: Create Challenge VMs
# ============================================================================

declare -a CREATED_VMS=()

create_challenge_vm() {
    local VM_ID=$(generate_vm_id)
    local VM_NAME="$VM_ID"

    # Ensure unique VM name
    while virsh dominfo "$VM_NAME" &>/dev/null || [[ -f "${IMAGE_DIR}/${VM_NAME}.qcow2" ]]; do
        VM_ID=$(generate_vm_id)
        VM_NAME="$VM_ID"
    done

    log_info "Creating challenge VM '${VM_NAME}'..."
    log_info "  - RAM: 512 MB"
    log_info "  - vCPUs: 1"
    log_info "  - Disk: 10 GB"

    # Create VM disk
    VM_DISK="${IMAGE_DIR}/${VM_NAME}.qcow2"
    qemu-img create -f qcow2 -F qcow2 -b "$BASE_IMAGE_PATH" "$VM_DISK" 10G

    # Create Cloud-Init Config
    SEED_DIR="${CLOUD_INIT_DIR}/${VM_NAME}"
    mkdir -p "$SEED_DIR"

    USER_DATA="${SEED_DIR}/user-data"
    META_DATA="${SEED_DIR}/meta-data"
    NETWORK_CONFIG="${SEED_DIR}/network-config"

    INSTANCE_ID="${VM_NAME}-$(date +%s)-${RANDOM}"

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
    <title>Challenge VM: ${VM_NAME}</title>
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
            <p><strong>VM ID:</strong> <span class="vm-id">${VM_NAME}</span></p>
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
local-hostname: ${VM_NAME}
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
        --name "${VM_NAME}" \
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

    log_success "VM '${VM_NAME}' created!"

    CREATED_VMS+=("$VM_NAME")

    echo "$VM_NAME"
}

log_info "Creating ${VM_COUNT} challenge VM(s)..."

for ((i=1; i<=VM_COUNT; i++)); do
    log_info "Creating VM $i of $VM_COUNT..."
    create_challenge_vm
    echo ""
done

# ============================================================================
# STEP 6: Wait for VMs to get IP addresses
# ============================================================================
log_info "Waiting for VMs to boot and obtain IP addresses..."

MAX_WAIT=180
WAIT_INTERVAL=5

declare -A VM_IPS
declare -A VM_MACS

for VM_NAME in "${CREATED_VMS[@]}"; do
    log_info "Waiting for '${VM_NAME}' to get IP..."

    ELAPSED=0
    VM_IP=""

    while [[ $ELAPSED -lt $MAX_WAIT ]]; do
        VM_IP=$(virsh domifaddr "${VM_NAME}" --source lease 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)

        if [[ -n "$VM_IP" ]]; then
            log_success "VM '${VM_NAME}' obtained IP: ${VM_IP}"
            VM_IPS["$VM_NAME"]="$VM_IP"
            break
        fi

        sleep $WAIT_INTERVAL
        ELAPSED=$((ELAPSED + WAIT_INTERVAL))

        if (( ELAPSED % 30 == 0 )); then
            log_info "Still waiting for ${VM_NAME}... (${ELAPSED}s / ${MAX_WAIT}s)"
        fi
    done

    if [[ -z "$VM_IP" ]]; then
        log_warn "VM '${VM_NAME}' did not obtain IP within ${MAX_WAIT} seconds."
        VM_IPS["$VM_NAME"]=""
    fi

    VM_MAC=$(virsh domiflist "${VM_NAME}" 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1 || true)
    if [[ -n "$VM_MAC" ]]; then
        VM_MACS["$VM_NAME"]="$VM_MAC"
    fi
done

# ============================================================================
# STEP 7: Verify HTTP accessibility
# ============================================================================
log_info "Verifying HTTP accessibility for each VM..."

sleep 10

declare -A VM_HTTP_STATUS

for VM_NAME in "${CREATED_VMS[@]}"; do
    VM_IP="${VM_IPS[$VM_NAME]:-}"

    if [[ -z "$VM_IP" ]]; then
        log_warn "Skipping HTTP check for '${VM_NAME}' - no IP address"
        VM_HTTP_STATUS["$VM_NAME"]="NO_IP"
        continue
    fi

    log_info "Checking HTTP on ${VM_NAME} (${VM_IP})..."

    HTTP_RESPONSE=""
    for attempt in {1..6}; do
        HTTP_RESPONSE=$(curl -s --connect-timeout 5 --max-time 10 "http://${VM_IP}/" 2>/dev/null || true)

        if [[ -n "$HTTP_RESPONSE" ]] && echo "$HTTP_RESPONSE" | grep -q "Challenge VM"; then
            log_success "HTTP server on '${VM_NAME}' is accessible!"
            VM_HTTP_STATUS["$VM_NAME"]="OK"
            break
        fi

        log_info "HTTP not ready yet, retrying... (attempt $attempt/6)"
        sleep 10
    done

    if [[ "${VM_HTTP_STATUS[$VM_NAME]:-}" != "OK" ]]; then
        log_warn "HTTP server on '${VM_NAME}' is not accessible"
        VM_HTTP_STATUS["$VM_NAME"]="FAILED"
    fi
done

# ============================================================================
# STEP 8: Final Summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "Host reset complete!"
echo "============================================================================"
echo ""
echo "Reset Summary:"
echo "  - All previous VMs destroyed"
echo "  - Cloud-init data cleaned up"
echo "  - Caddy verified running on host"
echo ""
echo "Challenge VMs Created:"
echo ""

for VM_NAME in "${CREATED_VMS[@]}"; do
    VM_IP="${VM_IPS[$VM_NAME]:-N/A}"
    VM_MAC="${VM_MACS[$VM_NAME]:-N/A}"
    HTTP_STATUS="${VM_HTTP_STATUS[$VM_NAME]:-UNKNOWN}"

    echo "  VM: ${VM_NAME}"
    echo "    - Disk: ${IMAGE_DIR}/${VM_NAME}.qcow2"
    echo "    - MAC Address: ${VM_MAC}"
    echo "    - IP Address: ${VM_IP}"
    echo "    - HTTP Status: ${HTTP_STATUS}"
    if [[ "$HTTP_STATUS" == "OK" ]] && [[ "$VM_IP" != "N/A" ]]; then
        echo "    - URL: http://${VM_IP}/"
    fi
    echo "    - Login: root / root"
    echo ""
done

echo "VM Specifications:"
echo "  - RAM: 512 MB"
echo "  - vCPUs: 1"
echo "  - Disk: 10 GB"
echo "  - Web Server: Apache (httpd)"
echo ""
echo "Useful commands:"
echo "  virsh list --all                          # List all VMs"
echo "  virsh domifaddr <vm-id> --source lease    # Get VM IP"
echo "  virsh console <vm-id>                     # Access VM console"
echo "  curl http://<vm-ip>/                      # Test HTTP server"
echo ""
echo "To customize the web page:"
echo "  ssh root@<vm-ip>"
echo "  nano /var/www/html/index.html"
echo ""

# Write summary file
SUMMARY_FILE="${IMAGE_DIR}/challenge-vms.txt"
log_info "Writing VM summary to ${SUMMARY_FILE}..."

cat > "$SUMMARY_FILE" <<EOF
# Challenge VMs - Created $(date)
# Format: VM_ID IP_ADDRESS MAC_ADDRESS HTTP_STATUS

EOF

for VM_NAME in "${CREATED_VMS[@]}"; do
    VM_IP="${VM_IPS[$VM_NAME]:-N/A}"
    VM_MAC="${VM_MACS[$VM_NAME]:-N/A}"
    HTTP_STATUS="${VM_HTTP_STATUS[$VM_NAME]:-UNKNOWN}"
    echo "${VM_NAME} ${VM_IP} ${VM_MAC} ${HTTP_STATUS}" >> "$SUMMARY_FILE"
done

log_success "Summary written to ${SUMMARY_FILE}"

# Final status
FAILED_COUNT=0
for VM_NAME in "${CREATED_VMS[@]}"; do
    if [[ "${VM_HTTP_STATUS[$VM_NAME]:-}" != "OK" ]]; then
        ((FAILED_COUNT++))
    fi
done

if [[ $FAILED_COUNT -eq 0 ]]; then
    log_success "All ${#CREATED_VMS[@]} challenge VM(s) are running with accessible HTTP servers!"
    log_success "You can now create sandboxes from these VMs with: fluid create --source-vm=<vm-id>"
else
    log_warn "${FAILED_COUNT} of ${#CREATED_VMS[@]} VM(s) may have issues. Check the summary above."
    log_warn "Troubleshooting steps:"
    log_warn "  1. Check VM is running: virsh list --all"
    log_warn "  2. Check network: virsh domiflist <vm-id>"
    log_warn "  3. Check DHCP: virsh net-dhcp-leases default"
    log_warn "  4. Access console: virsh console <vm-id> (login: ubuntu/ubuntu)"
    log_warn "  5. Inside VM: systemctl status apache2; ip addr show"
fi
