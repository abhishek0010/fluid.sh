#!/bin/bash
# setup-ubuntu.sh
#
# Sets up libvirt and KVM on Ubuntu (x86 architecture).
# This script installs necessary packages, configures groups, and validates the installation.
#
# Usage: sudo ./setup-ubuntu.sh [VM_INDEX]

VM_INDEX=${1:-1}
VM_NAME="test-vm-${VM_INDEX}"

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

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

log_info "Starting libvirt setup for Ubuntu..."

# ============================================================================
# STEP 1: Check for Virtualization Support
# ============================================================================
log_info "Checking for hardware virtualization support..."

if egrep -c '(vmx|svm)' /proc/cpuinfo > /dev/null; then
    log_success "Hardware virtualization support detected."
else
    log_warn "Hardware virtualization (VT-x/AMD-V) NOT detected in /proc/cpuinfo."
    log_warn "If you are running this inside a VM, ensure nested virtualization is enabled."
    log_warn "Proceeding, but KVM might not work..."
    # We don't exit here because sometimes it might be emulated or the check is flaky in some containers
fi

if command -v kvm-ok &>/dev/null; then
    if kvm-ok; then
        log_success "KVM acceleration can be used."
    else
        log_warn "KVM acceleration CANNOT be used."
    fi
else
    log_info "'kvm-ok' command not found. Installing cpu-checker to check KVM status..."
    apt-get update -qq
    apt-get install -y -qq cpu-checker
    if kvm-ok; then
        log_success "KVM acceleration can be used."
    else
        log_warn "KVM acceleration CANNOT be used."
    fi
fi

# ============================================================================
# STEP 2: Install Libvirt and QEMU Packages
# ============================================================================
log_info "Installing libvirt, qemu-kvm, and utilities..."

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
    wget \
    cloud-image-utils \
    qemu-kvm \
    libvirt-daemon-system \
    libvirt-clients \
    bridge-utils \
    virtinst \
    virt-manager \
    genisoimage

log_success "Packages installed successfully."

# ============================================================================
# STEP 3: Enable and Start libvirtd Service
# ============================================================================
log_info "Enabling and starting libvirtd service..."

systemctl enable --now libvirtd
systemctl start libvirtd

if systemctl is-active --quiet libvirtd; then
    log_success "libvirtd is running."
else
    log_error "libvirtd failed to start. Check 'systemctl status libvirtd'."
    exit 1
fi

# ============================================================================
# STEP 4: Ensure default network is active
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

# Verify DHCP is configured
if ! virsh net-dumpxml default | grep -q "<dhcp>"; then
    log_warn "Default network does not have DHCP configured!"
    log_warn "VMs may not get IP addresses automatically."
fi

# ============================================================================
# STEP 5: Configure User Groups
# ============================================================================
log_info "Configuring user groups..."

# Get the user who invoked sudo (if applicable), otherwise assume root or current user
REAL_USER=${SUDO_USER:-$USER}

if [[ -n "$REAL_USER" ]]; then
    log_info "Adding user '$REAL_USER' to 'libvirt' and 'kvm' groups..."
    usermod -aG libvirt "$REAL_USER"
    usermod -aG kvm "$REAL_USER"
    log_success "User '$REAL_USER' added to groups."
else
    log_warn "Could not determine real user. Skipping group addition."
fi

# ============================================================================
# STEP 6: Verify Installation
# ============================================================================
log_info "Verifying installation..."

if virsh list --all > /dev/null; then
    log_success "virsh command working correctly."
else
    log_error "virsh command failed. Check permissions or libvirtd status."
    exit 1
fi

# ============================================================================
# STEP 7: Create Test VM (Ubuntu 22.04 Cloud Image)
# ============================================================================
log_info "Creating real Ubuntu test VM '${VM_NAME}'..."

IMAGE_DIR="/var/lib/libvirt/images"
CLOUD_INIT_DIR="${IMAGE_DIR}/cloud-init"
BASE_IMAGE="ubuntu-22.04-minimal-cloudimg-amd64.img"
BASE_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/${BASE_IMAGE}"
BASE_IMAGE_PATH="${IMAGE_DIR}/${BASE_IMAGE}"

# Ensure directories exist
mkdir -p "$IMAGE_DIR"
mkdir -p "$CLOUD_INIT_DIR"

# 1. Download Base Image if missing
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

# 2. Create Disk for this VM (Copy-on-Write)
VM_DISK="${IMAGE_DIR}/${VM_NAME}.qcow2"
log_info "Creating VM disk: $VM_DISK"
if [[ -f "$VM_DISK" ]]; then
    log_warn "Disk $VM_DISK already exists, overwriting..."
    rm -f "$VM_DISK"
fi
qemu-img create -f qcow2 -F qcow2 -b "$BASE_IMAGE_PATH" "$VM_DISK" 10G

# 3. Create Cloud-Init Config with proper network configuration
# Store in persistent location so VM can access it on reboot
SEED_DIR="${CLOUD_INIT_DIR}/${VM_NAME}"
mkdir -p "$SEED_DIR"

USER_DATA="${SEED_DIR}/user-data"
META_DATA="${SEED_DIR}/meta-data"
NETWORK_CONFIG="${SEED_DIR}/network-config"

# Generate a unique instance-id for this VM
INSTANCE_ID="${VM_NAME}-$(date +%s)"

log_info "Creating cloud-init configuration with network settings..."

# User-data: password, SSH, guest agent
# NOTE: Network config is in separate network-config file, not here
# Having it in both places can cause conflicts
cat > "$USER_DATA" <<EOF
#cloud-config
password: ubuntu
chpasswd: { expire: False }
ssh_pwauth: True

# Install and enable guest agent for better VM management
packages:
  - qemu-guest-agent

# Enable guest agent on boot
runcmd:
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
EOF

# Meta-data: unique instance-id is CRITICAL for cloud-init to run on clones
cat > "$META_DATA" <<EOF
instance-id: ${INSTANCE_ID}
local-hostname: ${VM_NAME}
EOF

# Network-config (NoCloud v2 format) - explicit network configuration
# Use 'name: en*' to match interface names (ens3, enp1s0, etc.) - more reliable than driver matching
cat > "$NETWORK_CONFIG" <<EOF
version: 2
ethernets:
  id0:
    match:
      name: en*
    dhcp4: true
EOF

log_success "Cloud-init config files created."

# 4. Install/Boot VM using virt-install's native --cloud-init option
# This uses SMBIOS to hint the datasource, which is more reliable than manual ISO
log_info "Booting VM with virt-install --cloud-init..."

# Generate a deterministic MAC address based on VM index to avoid conflicts
# Using the QEMU/KVM prefix 52:54:00
MAC_SUFFIX=$(printf '%02x:%02x:%02x' $((VM_INDEX / 256 / 256 % 256)) $((VM_INDEX / 256 % 256)) $((VM_INDEX % 256)))
MAC_ADDRESS="52:54:00:${MAC_SUFFIX}"

log_info "Using MAC address: ${MAC_ADDRESS}"

virt-install \
    --name "${VM_NAME}" \
    --memory 2048 \
    --vcpus 2 \
    --disk "${VM_DISK},device=disk,bus=virtio" \
    --cloud-init user-data="${USER_DATA}",meta-data="${META_DATA}",network-config="${NETWORK_CONFIG}" \
    --os-variant ubuntu22.04 \
    --import \
    --noautoconsole \
    --graphics none \
    --console pty,target_type=serial \
    --network network=default,model=virtio,mac="${MAC_ADDRESS}"

log_success "VM '${VM_NAME}' started!"

# ============================================================================
# STEP 8: Wait for VM to get IP address
# ============================================================================
log_info "Waiting for VM to obtain IP address (this may take 30-60 seconds)..."

MAX_WAIT=120
WAIT_INTERVAL=5
ELAPSED=0
VM_IP=""

while [[ $ELAPSED -lt $MAX_WAIT ]]; do
    # Try to get IP from DHCP leases
    VM_IP=$(virsh domifaddr "${VM_NAME}" --source lease 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)

    if [[ -n "$VM_IP" ]]; then
        log_success "VM '${VM_NAME}' obtained IP address: ${VM_IP}"
        break
    fi

    log_info "Waiting for IP... (${ELAPSED}s / ${MAX_WAIT}s)"
    sleep $WAIT_INTERVAL
    ELAPSED=$((ELAPSED + WAIT_INTERVAL))
done

if [[ -z "$VM_IP" ]]; then
    log_warn "VM did not obtain IP address within ${MAX_WAIT} seconds."
    log_warn "This may indicate a network configuration issue."
    log_warn "Check: virsh domifaddr ${VM_NAME} --source lease"
    log_warn "Check: virsh console ${VM_NAME} (login: ubuntu/ubuntu)"
fi

# ============================================================================
# STEP 9: Verify VM network interface
# ============================================================================
log_info "Verifying VM network configuration..."

# Check MAC address
VM_MAC=$(virsh domiflist "${VM_NAME}" 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1 || true)
if [[ -n "$VM_MAC" ]]; then
    log_success "VM MAC address: ${VM_MAC}"
else
    log_warn "Could not determine VM MAC address"
fi

# Check interface stats
IFACE=$(virsh domiflist "${VM_NAME}" 2>/dev/null | awk 'NR>2 && $1 != "" {print $1}' | head -1 || true)
if [[ -n "$IFACE" ]]; then
    log_info "Network interface: ${IFACE}"
    virsh domifstat "${VM_NAME}" "${IFACE}" 2>/dev/null || true
fi

# ============================================================================
# STEP 10: Final Summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "Libvirt setup complete!"
echo "============================================================================"
echo ""
echo "Setup Summary:"
echo "  - Installed: qemu-kvm, libvirt-daemon-system, libvirt-clients, bridge-utils, virtinst"
echo "  - Service: libvirtd enabled and started"
echo "  - Network: default network active with DHCP"
echo "  - Test VM: '${VM_NAME}' has been created and started"
echo "  - VM Disk: ${VM_DISK}"
echo "  - Cloud-Init: virt-install --cloud-init (native injection)"
if [[ -n "$VM_MAC" ]]; then
    echo "  - MAC Address: ${VM_MAC}"
fi
if [[ -n "$VM_IP" ]]; then
    echo "  - IP Address: ${VM_IP}"
else
    echo "  - IP Address: (pending - check with 'virsh domifaddr ${VM_NAME} --source lease')"
fi
echo "  - Login: ubuntu / ubuntu"
if [[ -n "$REAL_USER" ]]; then
    echo "  - User: '$REAL_USER' added to 'libvirt' and 'kvm' groups"
    echo "    NOTE: You may need to log out and log back in for group changes to take effect."
fi
echo ""
echo "Useful commands:"
echo "  virsh list --all                          # List all VMs"
echo "  virsh domifaddr ${VM_NAME} --source lease # Get VM IP"
echo "  virsh console ${VM_NAME}                  # Access VM console"
echo "  ssh ubuntu@${VM_IP:-<IP>}                 # SSH to VM (password: ubuntu)"
echo ""
