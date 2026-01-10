#!/usr/bin/env bash
#
# setup-lima-libvirt.sh
#
# This script sets up a Lima VM with libvirt/KVM support for testing the
# virsh-sandbox API control plane. The API runs on the host (or in a container)
# and connects to libvirt inside Lima via TCP or SSH.
#
# Architecture:
#   ┌─────────────────────────────────────────────────────────────┐
#   │                    Host Machine (macOS/Linux)               │
#   │                                                             │
#   │  ┌─────────────────────┐     ┌───────────────────────────┐  │
#   │  │  virsh-sandbox API  │────►│      Lima VM (Ubuntu)     │  │
#   │  │  (Go REST Server)   │     │                           │  │
#   │  │                     │     │  ┌─────────────────────┐  │  │
#   │  │  LIBVIRT_URI=       │     │  │   libvirt/KVM       │  │  │
#   │  │   qemu+ssh://...    │     │  │                     │  │  │
#   │  │   qemu+tcp://...    │     │  │  ┌───────────────┐  │  │  │
#   │  └─────────────────────┘     │  │  │   test-vm     │  │  │  │
#   │                              │  │  └───────────────┘  │  │  │
#   │                              │  └─────────────────────┘  │  │
#   │                              └───────────────────────────┘  │
#   └─────────────────────────────────────────────────────────────┘
#
# Prerequisites:
#   - macOS: Lima installed (brew install lima)
#   - Linux: Lima installed or native libvirt setup
#   - Sufficient disk space (~20GB recommended)
#
# Usage:
#   ./scripts/setup-lima-libvirt.sh [options]
#
# Options:
#   --name NAME       Lima VM name (default: virsh-sandbox-dev)
#   --cpus N          Number of CPUs (default: 4)
#   --memory N        Memory in GB (default: 8)
#   --disk N          Disk size in GB (default: 50)
#   --create-test-vm  Also create a test VM inside Lima
#   --help            Show this help message
#
# After setup, connect to libvirt from the host:
#   - Via SSH: qemu+ssh://localhost:60022/system?keyfile=~/.lima/virsh-sandbox-dev/ssh/id_ed25519
#   - Via TCP: qemu+tcp://localhost:16509/system (less secure, but simpler)

set -euo pipefail

# =============================================================================
# Configuration Defaults
# =============================================================================

LIMA_VM_NAME="virsh-sandbox-dev"
LIMA_CPUS=4
LIMA_MEMORY=8
LIMA_DISK=50
CREATE_TEST_VM=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${PROJECT_ROOT}/.." && pwd)"
SSH_CA_DIR="${REPO_ROOT}/.ssh-ca"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

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
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    head -50 "$0" | grep -E "^#" | sed 's/^# \?//'
    exit 0
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "Required command '$1' not found. Please install it first."
        exit 1
    fi
}

# =============================================================================
# SSH CA Setup
# =============================================================================

setup_ssh_ca() {
    local ca_dir="$1"
    local ca_key="${ca_dir}/ssh_ca"
    local ca_pub="${ca_dir}/ssh_ca.pub"

    if [[ -f "${ca_key}" ]] && [[ -f "${ca_pub}" ]]; then
        log_info "SSH CA already exists at ${ca_dir}"
        return 0
    fi

    log_info "Generating SSH Certificate Authority..."
    mkdir -p "${ca_dir}"
    chmod 700 "${ca_dir}"

    ssh-keygen -t ed25519 -f "${ca_key}" -N "" -C "virsh-sandbox-ssh-ca" -q

    if [[ ! -f "${ca_key}" ]] || [[ ! -f "${ca_pub}" ]]; then
        log_error "Failed to generate SSH CA"
        return 1
    fi

    chmod 600 "${ca_key}"
    chmod 644 "${ca_pub}"

    log_success "SSH CA generated at ${ca_dir}"
    log_info "  Private key: ${ca_key}"
    log_info "  Public key:  ${ca_pub}"
}

get_ssh_ca_pubkey() {
    local ca_pub="${SSH_CA_DIR}/ssh_ca.pub"
    if [[ -f "${ca_pub}" ]]; then
        cat "${ca_pub}"
    else
        echo ""
    fi
}

# =============================================================================
# Parse Arguments
# =============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        --name)
            LIMA_VM_NAME="$2"
            shift 2
            ;;
        --cpus)
            LIMA_CPUS="$2"
            shift 2
            ;;
        --memory)
            LIMA_MEMORY="$2"
            shift 2
            ;;
        --disk)
            LIMA_DISK="$2"
            shift 2
            ;;
        --create-test-vm)
            CREATE_TEST_VM=true
            shift
            ;;
        --help|-h)
            show_help
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            ;;
    esac
done

# =============================================================================
# Pre-flight Checks
# =============================================================================

log_info "Running pre-flight checks..."

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Darwin)
        log_info "Detected macOS"
        check_command "limactl"
        check_command "brew"
        PLATFORM="macos"
        ;;
    Linux)
        log_info "Detected Linux"
        # On Linux, we can either use Lima or native libvirt
        if command -v limactl &> /dev/null; then
            PLATFORM="linux-lima"
        else
            PLATFORM="linux-native"
            log_info "Lima not found, will use native libvirt setup"
        fi
        ;;
    *)
        log_error "Unsupported operating system: ${OS}"
        exit 1
        ;;
esac

# =============================================================================
# Lima Configuration Template
# =============================================================================

generate_lima_config() {
    local config_file="$1"

    cat > "${config_file}" << 'LIMA_CONFIG_EOF'
# Lima configuration for virsh-sandbox development
# This VM provides a libvirt/KVM environment accessible from the host

# VM Images - Using Ubuntu 24.04 LTS for stability
images:
  - location: "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img"
    arch: "x86_64"
  - location: "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-arm64.img"
    arch: "aarch64"

# Resource allocation
cpus: __CPUS__
memory: "__MEMORY__GiB"
disk: "__DISK__GiB"

# Enable nested virtualization (critical for KVM inside Lima)
vmType: "qemu"
firmware:
  legacyBIOS: false

# Mount directories to share between host and Lima
mounts:
  # Mount home directory at same macOS path (allows cd to work from host)
  - location: "__HOME__"
    mountPoint: "__HOME__"
    writable: true
  # Libvirt images directory - mount to /mnt to avoid conflicts with libvirt package install
  # The package tries to chmod /var/lib/libvirt/images which fails on 9p mounts
  # We symlink the subdirectories in the provision script after libvirt is installed
  - location: "/var/lib/libvirt/images"
    mountPoint: "/mnt/host-libvirt-images"
    writable: true

# Containerd is not needed for our use case
containerd:
  system: false
  user: false

# SSH configuration - we'll use this for qemu+ssh:// connections
ssh:
  forwardAgent: true

# Port forwarding for libvirt access from host
portForwards:
  # Forward libvirt TCP port (for qemu+tcp:// connections)
  - guestPort: 16509
    hostPort: 16509
    proto: tcp
  # Forward SSH for qemu+ssh:// connections (Lima default is 60022)
  # Lima handles this automatically, but explicit for clarity

# Provision script to install libvirt and configure for remote access
provision:
  - mode: system
    script: |
      #!/bin/bash
      set -eux

      # Update package lists
      apt-get update

      # Install libvirt, QEMU, and related tools
      DEBIAN_FRONTEND=noninteractive apt-get install -y \
        qemu-kvm \
        qemu-utils \
        libvirt-daemon-system \
        libvirt-clients \
        virtinst \
        bridge-utils \
        ovmf \
        cpu-checker \
        cloud-image-utils \
        genisoimage \
        libguestfs-tools \
        qemu-block-extra \
        podman \
        buildah \
        skopeo \
        curl \
        wget \
        jq \
        htop \
        vim

      # Enable and start libvirtd
      systemctl enable libvirtd
      systemctl start libvirtd

      # Configure libvirt for remote TCP access (qemu+tcp://)
      # WARNING: TCP without TLS is not secure - use only for local development
      printf '%s\n' \
        'listen_tls = 0' \
        'listen_tcp = 1' \
        'tcp_port = "16509"' \
        'auth_tcp = "none"' \
        'unix_sock_group = "libvirt"' \
        'unix_sock_rw_perms = "0770"' \
        'listen_addr = "0.0.0.0"' \
        > /etc/libvirt/libvirtd.conf

      # Configure QEMU to disable security driver (AppArmor) and run as root
      # Required for accessing disk images on 9p mounts from host
      printf '%s\n' \
        'security_driver = "none"' \
        'user = "root"' \
        'group = "root"' \
        >> /etc/libvirt/qemu.conf

      # Disable socket activation so we can use -l flag for TCP listening
      # The -l flag and socket activation are mutually exclusive
      systemctl stop libvirtd.socket libvirtd-ro.socket libvirtd-admin.socket || true
      systemctl disable libvirtd.socket libvirtd-ro.socket libvirtd-admin.socket || true
      systemctl mask libvirtd.socket libvirtd-ro.socket libvirtd-admin.socket || true

      # Configure libvirtd service to run with -l (listen) flag
      mkdir -p /etc/systemd/system/libvirtd.service.d
      printf '%s\n' \
        '[Service]' \
        'ExecStart=' \
        'ExecStart=/usr/sbin/libvirtd -l' \
        > /etc/systemd/system/libvirtd.service.d/override.conf

      systemctl daemon-reload
      systemctl enable libvirtd
      systemctl restart libvirtd

      # Enable default network
      virsh net-autostart default || true
      virsh net-start default || true

      # Verify KVM is available
      if kvm-ok; then
        echo "KVM acceleration is available"
      else
        echo "WARNING: KVM acceleration may not be available (nested virt)"
        echo "VMs will run in emulation mode (slower)"
      fi

      # Symlink libvirt image directories to host-mounted location
      # The host directory is mounted at /mnt/host-libvirt-images to avoid
      # conflicts with libvirt package installation (it tries to chmod which fails on 9p)
      mkdir -p /mnt/host-libvirt-images/base
      mkdir -p /mnt/host-libvirt-images/jobs
      # Remove libvirt's default directories and replace with symlinks
      rm -rf /var/lib/libvirt/images/base
      rm -rf /var/lib/libvirt/images/jobs
      ln -sf /mnt/host-libvirt-images/base /var/lib/libvirt/images/base
      ln -sf /mnt/host-libvirt-images/jobs /var/lib/libvirt/images/jobs

  - mode: user
    script: |
      #!/bin/bash
      set -eux

      # Add user to libvirt and kvm groups
      sudo usermod -aG libvirt,kvm $(whoami)

      # Create a test script to verify libvirt is working
      cat > ~/test-libvirt.sh << 'EOF'
      #!/bin/bash
      echo "Testing local libvirt connection..."
      virsh -c qemu:///system version
      echo ""
      echo "Listing networks..."
      virsh -c qemu:///system net-list
      echo ""
      echo "Listing VMs..."
      virsh -c qemu:///system list --all
      EOF
      chmod +x ~/test-libvirt.sh

# Message displayed after VM creation
message: |

  ╔═══════════════════════════════════════════════════════════════════════════╗
  ║                    virsh-sandbox Libvirt Environment                       ║
  ╠═══════════════════════════════════════════════════════════════════════════╣
  ║                                                                           ║
  ║  Your Lima VM with libvirt/KVM is ready!                                  ║
  ║                                                                           ║
  ║  Connect to libvirt from your host using one of these URIs:               ║
  ║                                                                           ║
  ║  Option 1 - TCP (simpler, less secure - dev only):                        ║
  ║    LIBVIRT_URI="qemu+tcp://localhost:16509/system"                        ║
  ║                                                                           ║
  ║  Option 2 - SSH (more secure, recommended):                               ║
  ║    LIBVIRT_URI="qemu+ssh://__USER__@localhost:__SSH_PORT__/system?keyfile=__SSH_KEY__"
  ║                                                                           ║
  ║  Test the connection:                                                     ║
  ║    virsh -c "$LIBVIRT_URI" list --all                                     ║
  ║                                                                           ║
  ║  SSH into Lima VM:                                                        ║
  ║    limactl shell __VM_NAME__                                              ║
  ║                                                                           ║
  ╚═══════════════════════════════════════════════════════════════════════════╝

LIMA_CONFIG_EOF

    # Replace placeholders
    sed -i.bak "s|__CPUS__|${LIMA_CPUS}|g" "${config_file}"
    sed -i.bak "s|__MEMORY__|${LIMA_MEMORY}|g" "${config_file}"
    sed -i.bak "s|__DISK__|${LIMA_DISK}|g" "${config_file}"
    sed -i.bak "s|__USER__|${USER}|g" "${config_file}"
    sed -i.bak "s|__VM_NAME__|${LIMA_VM_NAME}|g" "${config_file}"
    sed -i.bak "s|__HOME__|${HOME}|g" "${config_file}"
    rm -f "${config_file}.bak"
}

# =============================================================================
# Create Test VM Script (runs inside Lima)
# =============================================================================

generate_test_vm_script() {
    local script_file="$1"

    cat > "${script_file}" << 'TEST_VM_EOF'
#!/bin/bash
#
# create-test-vm.sh
#
# Creates a lightweight test VM inside the Lima libvirt environment
# for testing the virsh-sandbox API control plane.
#
# Usage: ./create-test-vm.sh [vm-name]

set -euo pipefail

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

VM_NAME="${1:-test-vm}"
VM_MEMORY=2048
VM_VCPUS=2
VM_DISK_SIZE="10G"
BASE_IMAGE_DIR="/var/lib/libvirt/images/base"

# Detect architecture and select appropriate cloud image and virt-install options
ARCH=$(uname -m)
case "${ARCH}" in
    aarch64|arm64)
        CLOUD_IMAGE_ARCH="arm64"
        CLOUD_IMAGE_URL="https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-arm64.img"
        # ARM64 requires specific virt-install options
        VIRT_INSTALL_ARCH_OPTS="--arch aarch64 --machine virt --boot uefi"
        ;;
    x86_64|amd64)
        CLOUD_IMAGE_ARCH="amd64"
        CLOUD_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/ubuntu-22.04-minimal-cloudimg-amd64.img"
        # x86_64 uses default options
        VIRT_INSTALL_ARCH_OPTS=""
        ;;
    *)
        echo -e "${RED}[ERROR]${NC} Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

echo -e "${BLUE}[INFO]${NC} Detected architecture: ${ARCH} -> using ${CLOUD_IMAGE_ARCH} image"
echo -e "${BLUE}[INFO]${NC} Creating test VM: ${VM_NAME}"

# Download cloud image if not present
CLOUD_IMAGE="${BASE_IMAGE_DIR}/ubuntu-22.04-cloudimg-${CLOUD_IMAGE_ARCH}.img"
if [ ! -f "${CLOUD_IMAGE}" ]; then
    echo -e "${BLUE}[INFO]${NC} Downloading Ubuntu cloud image..."
    sudo mkdir -p "${BASE_IMAGE_DIR}"
    sudo wget -q --show-progress -O "${CLOUD_IMAGE}" "${CLOUD_IMAGE_URL}"
    sudo chmod 644 "${CLOUD_IMAGE}"
fi

# Create a copy for this VM
VM_DISK="${BASE_IMAGE_DIR}/${VM_NAME}.qcow2"
if [ -f "${VM_DISK}" ]; then
    echo -e "${BLUE}[INFO]${NC} VM disk already exists, skipping creation"
else
    echo -e "${BLUE}[INFO]${NC} Creating VM disk from cloud image..."
    sudo qemu-img create -f qcow2 -b "${CLOUD_IMAGE}" -F qcow2 "${VM_DISK}" "${VM_DISK_SIZE}"
    sudo chmod 644 "${VM_DISK}"
fi

# Create cloud-init configuration
CLOUD_INIT_DIR="/tmp/cloud-init-${VM_NAME}"
mkdir -p "${CLOUD_INIT_DIR}"

# User data - configure the VM
cat > "${CLOUD_INIT_DIR}/user-data" << 'USERDATA'
#cloud-config
hostname: test-vm
manage_etc_hosts: true

users:
  - name: testuser
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    lock_passwd: false

chpasswd:
  list: |
    testuser:testpassword
    root:rootpassword
  expire: false

ssh_pwauth: true

packages:
  - curl
  - wget
  - vim
  - htop

runcmd:
  - echo "Test VM is ready for virsh-sandbox testing" > /etc/motd
  - systemctl enable ssh
  - systemctl start ssh

final_message: "Test VM boot completed in $UPTIME seconds"
USERDATA

# Meta data
cat > "${CLOUD_INIT_DIR}/meta-data" << METADATA
instance-id: ${VM_NAME}
local-hostname: ${VM_NAME}
METADATA

# Create cloud-init ISO
CLOUD_INIT_ISO="${BASE_IMAGE_DIR}/${VM_NAME}-cloud-init.iso"
echo -e "${BLUE}[INFO]${NC} Creating cloud-init ISO..."
sudo genisoimage -output "${CLOUD_INIT_ISO}" -volid cidata -joliet -rock \
    "${CLOUD_INIT_DIR}/user-data" \
    "${CLOUD_INIT_DIR}/meta-data" 2>/dev/null
sudo chmod 644 "${CLOUD_INIT_ISO}"

# Clean up temp directory
rm -rf "${CLOUD_INIT_DIR}"

# Check if VM already exists
if virsh dominfo "${VM_NAME}" &>/dev/null; then
    echo -e "${BLUE}[INFO]${NC} VM ${VM_NAME} already exists"
    virsh dominfo "${VM_NAME}"
    exit 0
fi

# Create the VM using virt-install
echo -e "${BLUE}[INFO]${NC} Creating VM with virt-install..."
echo -e "${BLUE}[INFO]${NC} Architecture options: ${VIRT_INSTALL_ARCH_OPTS:-default}"

# Build virt-install command with architecture-specific options
# shellcheck disable=SC2086
sudo virt-install \
    --name "${VM_NAME}" \
    --memory "${VM_MEMORY}" \
    --vcpus "${VM_VCPUS}" \
    --disk "path=${VM_DISK},format=qcow2,bus=virtio" \
    --disk "path=${CLOUD_INIT_ISO},device=cdrom,bus=scsi" \
    --controller scsi,model=virtio-scsi \
    --os-variant ubuntu22.04 \
    --network network=default,model=virtio \
    --graphics vnc \
    --console pty,target_type=serial \
    --import \
    --noautoconsole \
    --wait 0 \
    ${VIRT_INSTALL_ARCH_OPTS}

echo -e "${GREEN}[SUCCESS]${NC} Test VM '${VM_NAME}' created successfully!"
echo ""
echo "VM Details:"
virsh dominfo "${VM_NAME}"
echo ""
echo "To connect to the VM console:"
echo "  virsh console ${VM_NAME}"
echo ""
echo "To get the VM's IP address (after it boots):"
echo "  virsh domifaddr ${VM_NAME}"
echo ""
echo "Default credentials:"
echo "  Username: testuser"
echo "  Password: testpassword"
echo ""
echo "  Username: root"
echo "  Password: rootpassword"

TEST_VM_EOF

    chmod +x "${script_file}"
}

# =============================================================================
# Create Test VM in Lima (using virsh define instead of virt-install)
# =============================================================================

create_test_vm_in_lima() {
    local lima_vm="$1"
    local vm_name="test-vm"

    # Check architecture
    local arch
    arch=$(limactl shell "${lima_vm}" -- uname -m)

    if [[ "${arch}" == "aarch64" ]]; then
        local cloud_image_url="https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-arm64.img"
        local cloud_image="ubuntu-22.04-server-cloudimg-arm64.img"
    else
        local cloud_image_url="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/ubuntu-22.04-minimal-cloudimg-amd64.img"
        local cloud_image="ubuntu-22.04-minimal-cloudimg-amd64.img"
    fi

    log_info "Detected architecture: ${arch}"

    # Get SSH CA public key for VM trust
    local ssh_ca_pubkey
    ssh_ca_pubkey=$(get_ssh_ca_pubkey)
    if [[ -z "${ssh_ca_pubkey}" ]]; then
        log_warn "SSH CA not found - VMs will not trust certificate-based auth"
        log_warn "Run: ./setup-ssh-ca.sh --dir ${SSH_CA_DIR}"
    else
        log_info "SSH CA public key will be injected into VM"
    fi

    # Run setup inside Lima
    limactl shell "${lima_vm}" -- sudo bash << SETUP_VM
set -e

BASE_DIR="/var/lib/libvirt/images/base"
VM_NAME="${vm_name}"
CLOUD_IMAGE="${cloud_image}"
CLOUD_IMAGE_URL="${cloud_image_url}"
SSH_CA_PUBKEY="${ssh_ca_pubkey}"

# Download cloud image if needed
if [ ! -f "\${BASE_DIR}/\${CLOUD_IMAGE}" ]; then
    echo "[INFO] Downloading cloud image..."
    mkdir -p "\${BASE_DIR}"
    wget -q --show-progress -O "\${BASE_DIR}/\${CLOUD_IMAGE}" "\${CLOUD_IMAGE_URL}"
fi

# Create VM disk if needed
if [ ! -f "\${BASE_DIR}/\${VM_NAME}.qcow2" ]; then
    echo "[INFO] Creating VM disk..."
    qemu-img create -f qcow2 -F qcow2 -b "\${BASE_DIR}/\${CLOUD_IMAGE}" "\${BASE_DIR}/\${VM_NAME}.qcow2" 10G
fi

# Create cloud-init ISO
echo "[INFO] Creating cloud-init ISO..."
mkdir -p /tmp/cloud-init-\${VM_NAME}

# Build cloud-init user-data with SSH CA trust
cat > /tmp/cloud-init-\${VM_NAME}/user-data << USERDATA
#cloud-config
hostname: test-vm

users:
  - name: testuser
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    lock_passwd: false
  - name: sandbox
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    lock_passwd: true

chpasswd:
  list: |
    testuser:testpassword
  expire: false

ssh_pwauth: true

network:
  version: 2
  ethernets:
    id0:
      match:
        driver: virtio*
      dhcp4: true

write_files:
  - path: /etc/ssh/ssh_ca.pub
    content: "\${SSH_CA_PUBKEY}"
    permissions: '0644'
    owner: root:root

runcmd:
  - |
    # Configure SSH to trust the CA for user authentication
    if [ -s /etc/ssh/ssh_ca.pub ]; then
      echo "TrustedUserCAKeys /etc/ssh/ssh_ca.pub" >> /etc/ssh/sshd_config
      systemctl restart sshd || systemctl restart ssh
      echo "[INFO] SSH CA trust configured"
    fi
USERDATA

cat > /tmp/cloud-init-\${VM_NAME}/meta-data << METADATA
instance-id: \${VM_NAME}
local-hostname: \${VM_NAME}
METADATA

cloud-localds "\${BASE_DIR}/\${VM_NAME}-cloud-init.iso" /tmp/cloud-init-\${VM_NAME}/user-data /tmp/cloud-init-\${VM_NAME}/meta-data
rm -rf /tmp/cloud-init-\${VM_NAME}

# Check if VM already defined
if virsh dominfo "\${VM_NAME}" &>/dev/null; then
    echo "[INFO] VM \${VM_NAME} already exists"
    virsh list --all
    exit 0
fi

# Create VM XML based on architecture
if [ "${arch}" = "aarch64" ]; then
    cat > /tmp/\${VM_NAME}.xml << 'VMXML'
<domain type='qemu'>
  <name>test-vm</name>
  <memory unit='MiB'>2048</memory>
  <vcpu>2</vcpu>
  <os>
    <type arch='aarch64' machine='virt'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/AAVMF/AAVMF_CODE.fd</loader>
    <nvram template='/usr/share/AAVMF/AAVMF_VARS.fd'/>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <gic version='3'/>
  </features>
  <cpu mode='custom'>
    <model>cortex-a57</model>
  </cpu>
  <devices>
    <emulator>/usr/bin/qemu-system-aarch64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='writeback'/>
      <source file='/var/lib/libvirt/images/base/test-vm.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='/var/lib/libvirt/images/base/test-vm-cloud-init.iso'/>
      <target dev='sda' bus='scsi'/>
      <readonly/>
    </disk>
    <controller type='scsi' model='virtio-scsi'/>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
    <serial type='pty'>
      <target port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <graphics type='vnc' port='-1' autoport='yes'/>
  </devices>
</domain>
VMXML
else
    cat > /tmp/\${VM_NAME}.xml << 'VMXML'
<domain type='kvm'>
  <name>test-vm</name>
  <memory unit='MiB'>2048</memory>
  <vcpu>2</vcpu>
  <os>
    <type arch='x86_64' machine='q35'>hvm</type>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
  </features>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='writeback'/>
      <source file='/var/lib/libvirt/images/base/test-vm.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='/var/lib/libvirt/images/base/test-vm-cloud-init.iso'/>
      <target dev='sda' bus='scsi'/>
      <readonly/>
    </disk>
    <controller type='scsi' model='virtio-scsi'/>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
    <serial type='pty'>
      <target port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <graphics type='vnc' port='-1' autoport='yes'/>
  </devices>
</domain>
VMXML
fi

echo "[INFO] Defining VM..."
virsh define /tmp/\${VM_NAME}.xml
rm -f /tmp/\${VM_NAME}.xml

echo "[SUCCESS] Test VM created!"
virsh list --all
echo ""
echo "Default credentials:"
echo "  Username: testuser"
echo "  Password: testpassword"
echo ""
echo "  Username: root"
echo "  Password: rootpassword"
SETUP_VM

    if [ $? -eq 0 ]; then
        log_success "Test VM 'test-vm' created successfully"
    else
        log_error "Failed to create test VM"
        return 1
    fi
}

# =============================================================================
# Native Linux Setup (without Lima)
# =============================================================================

setup_native_linux() {
    log_info "Setting up native libvirt on Linux..."

    # Check if running as root or with sudo
    if [ "$EUID" -ne 0 ]; then
        log_warn "Some operations may require sudo access"
    fi

    # Install required packages
    log_info "Installing required packages..."
    if command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
            qemu-kvm \
            qemu-utils \
            libvirt-daemon-system \
            libvirt-clients \
            virtinst \
            bridge-utils \
            ovmf \
            cpu-checker \
            cloud-image-utils \
            genisoimage \
            libguestfs-tools \
            podman \
            buildah \
            skopeo
    elif command -v dnf &> /dev/null; then
        sudo dnf install -y \
            qemu-kvm \
            qemu-img \
            libvirt \
            libvirt-client \
            virt-install \
            bridge-utils \
            edk2-ovmf \
            cloud-utils \
            genisoimage \
            libguestfs-tools \
            podman \
            buildah \
            skopeo
    else
        log_error "Unsupported package manager. Please install libvirt manually."
        exit 1
    fi

    # Enable and start libvirtd
    sudo systemctl enable libvirtd
    sudo systemctl start libvirtd

    # Add current user to libvirt group
    sudo usermod -aG libvirt,kvm "$(whoami)"

    # Enable default network
    sudo virsh net-autostart default || true
    sudo virsh net-start default || true

    # Create directories
    sudo mkdir -p /var/lib/libvirt/images/base
    sudo mkdir -p /var/lib/libvirt/images/jobs

    log_success "Native libvirt setup complete!"
    log_info "You may need to log out and back in for group changes to take effect"
}

# =============================================================================
# Generate Environment File
# =============================================================================

generate_env_file() {
    local env_file="$1"
    local ssh_port="$2"
    local ssh_key="$3"

    cat > "${env_file}" << ENV_EOF
# virsh-sandbox development environment configuration
# Source this file or copy values to your .env

# Option 1: TCP connection (simpler, less secure - dev only)
LIBVIRT_URI_TCP="qemu+tcp://localhost:16509/system"

# Option 2: SSH connection (more secure, recommended)
LIBVIRT_URI_SSH="qemu+ssh://${USER}@localhost:${ssh_port}/system?keyfile=${ssh_key}"

# Default to SSH connection
LIBVIRT_URI="\${LIBVIRT_URI_SSH}"

# Lima VM details
LIMA_VM_NAME="${LIMA_VM_NAME}"
LIMA_SSH_PORT="${ssh_port}"
LIMA_SSH_KEY="${ssh_key}"

# Libvirt image directories (inside the Lima VM)
BASE_IMAGE_DIR="/var/lib/libvirt/images/base"
SANDBOX_WORKDIR="/var/lib/libvirt/images/jobs"

# API configuration
API_HTTP_ADDR=":8080"
ENV_EOF

    log_success "Environment file created: ${env_file}"
}

# =============================================================================
# Update Root .env File for Docker Compose
# =============================================================================

update_root_env_file() {
    local ssh_port="$1"
    local env_file="${REPO_ROOT}/.env"
    local libvirt_uri="qemu+ssh://${USER}@host.docker.internal:${ssh_port}/system"

    if [ -f "${env_file}" ]; then
        # File exists - update LIBVIRT_URI if present, otherwise append
        if grep -q "^LIBVIRT_URI=" "${env_file}"; then
            # Update existing LIBVIRT_URI line
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' "s|^LIBVIRT_URI=.*|LIBVIRT_URI=${libvirt_uri}|" "${env_file}"
            else
                sed -i "s|^LIBVIRT_URI=.*|LIBVIRT_URI=${libvirt_uri}|" "${env_file}"
            fi
            log_info "Updated LIBVIRT_URI in ${env_file}"
        else
            # Append LIBVIRT_URI
            echo "" >> "${env_file}"
            echo "# Lima VM libvirt connection" >> "${env_file}"
            echo "LIBVIRT_URI=${libvirt_uri}" >> "${env_file}"
            log_info "Added LIBVIRT_URI to ${env_file}"
        fi

        # Update or add LIMA_SSH_PORT
        if grep -q "^LIMA_SSH_PORT=" "${env_file}"; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' "s|^LIMA_SSH_PORT=.*|LIMA_SSH_PORT=${ssh_port}|" "${env_file}"
            else
                sed -i "s|^LIMA_SSH_PORT=.*|LIMA_SSH_PORT=${ssh_port}|" "${env_file}"
            fi
        else
            echo "LIMA_SSH_PORT=${ssh_port}" >> "${env_file}"
        fi
    else
        # Create new .env file with essential settings
        log_info "Creating ${env_file}..."
        cat > "${env_file}" << ENV_EOF
# Lima VM SSH port - update this if Lima restarts with a different port
# Check current port with: limactl list ${LIMA_VM_NAME}
LIBVIRT_URI=${libvirt_uri}
LIMA_SSH_PORT=${ssh_port}

# Libvirt network
LIBVIRT_NETWORK=default

# Image directories (shared between Mac, Docker, and Lima via mount)
BASE_IMAGES_DIR=/var/lib/libvirt/images/base
JOBS_DIR=/var/lib/libvirt/images/jobs

# API settings
API_HTTP_ADDR=:8080

# VM defaults
DEFAULT_VCPUS=2
DEFAULT_MEMORY_MB=2048

# Timeouts
COMMAND_TIMEOUT_SEC=600
IP_DISCOVERY_TIMEOUT_SEC=120

# Logging
LOG_FORMAT=text
LOG_LEVEL=info
ENV_EOF
    fi

    log_success "Root .env file updated: ${env_file}"
    log_info "  LIBVIRT_URI=${libvirt_uri}"
    log_info "  LIMA_SSH_PORT=${ssh_port}"
}

# =============================================================================
# Main Setup Logic
# =============================================================================

main() {
    log_info "Starting virsh-sandbox libvirt environment setup"
    log_info "Configuration:"
    log_info "  VM Name: ${LIMA_VM_NAME}"
    log_info "  CPUs: ${LIMA_CPUS}"
    log_info "  Memory: ${LIMA_MEMORY}GB"
    log_info "  Disk: ${LIMA_DISK}GB"
    log_info "  Create Test VM: ${CREATE_TEST_VM}"
    log_info "  SSH CA Dir: ${SSH_CA_DIR}"
    echo ""

    # Setup SSH CA (generates if not exists)
    setup_ssh_ca "${SSH_CA_DIR}"
    echo ""

    case "${PLATFORM}" in
        macos|linux-lima)
            # Check if Lima VM already exists
            if limactl list -q | grep -q "^${LIMA_VM_NAME}$"; then
                log_warn "Lima VM '${LIMA_VM_NAME}' already exists"
                read -p "Do you want to delete and recreate it? [y/N] " -n 1 -r
                echo ""
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    log_info "Stopping and deleting existing VM..."
                    limactl stop "${LIMA_VM_NAME}" 2>/dev/null || true
                    limactl delete "${LIMA_VM_NAME}" --force
                else
                    log_info "Keeping existing VM"
                    if [ "${CREATE_TEST_VM}" = true ]; then
                        log_info "Creating test VM inside existing Lima instance..."
                        create_test_vm_in_lima "${LIMA_VM_NAME}"
                    fi
                    exit 0
                fi
            fi

            # Generate Lima configuration
            LIMA_CONFIG="/tmp/${LIMA_VM_NAME}.yaml"
            log_info "Generating Lima configuration..."
            generate_lima_config "${LIMA_CONFIG}"

            # Create the Lima VM
            log_info "Creating Lima VM (this may take several minutes)..."
            limactl create --name="${LIMA_VM_NAME}" "${LIMA_CONFIG}"

            # Start the Lima VM
            log_info "Starting Lima VM..."
            limactl start "${LIMA_VM_NAME}"

            # Wait for VM to be ready
            log_info "Waiting for VM to be fully ready..."
            sleep 10

            # Get SSH port and key path
            SSH_PORT=$(limactl list "${LIMA_VM_NAME}" --format '{{.SSHLocalPort}}' 2>/dev/null || echo "60022")
            SSH_KEY="${HOME}/.lima/_config/user"

            # Wait for libvirtd to be fully ready
            log_info "Waiting for libvirtd to be ready..."
            for i in {1..30}; do
                if limactl shell "${LIMA_VM_NAME}" -- sudo virsh version &>/dev/null; then
                    log_success "Libvirt is working correctly inside Lima"
                    break
                fi
                sleep 2
                if [ "$i" -eq 30 ]; then
                    log_warn "Libvirt may not be fully configured yet"
                fi
            done

            # Test TCP connection from host
            log_info "Testing TCP connection from host..."
            for i in {1..10}; do
                if virsh -c "qemu+tcp://localhost:16509/system" version &>/dev/null; then
                    log_success "TCP connection to libvirt is working!"
                    break
                fi
                sleep 2
                if [ "$i" -eq 10 ]; then
                    log_warn "TCP connection not yet available. It may take a moment."
                    log_info "Try: virsh -c 'qemu+tcp://localhost:16509/system' version"
                fi
            done

            # Create test VM if requested
            if [ "${CREATE_TEST_VM}" = true ]; then
                log_info "Creating test VM inside Lima..."
                create_test_vm_in_lima "${LIMA_VM_NAME}"
            fi

            # Update root .env file for docker-compose
            update_root_env_file "${SSH_PORT}"

            # Generate environment file
            ENV_FILE="${PROJECT_ROOT}/.env.lima"
            generate_env_file "${ENV_FILE}" "${SSH_PORT}" "${SSH_KEY}"

            # Also save the test VM script to the project
            TEST_VM_SCRIPT_LOCAL="${SCRIPT_DIR}/create-test-vm.sh"
            generate_test_vm_script "${TEST_VM_SCRIPT_LOCAL}"

            # Clean up
            rm -f "${LIMA_CONFIG}"

            log_success "Lima VM '${LIMA_VM_NAME}' is ready!"
            echo ""
            echo "═══════════════════════════════════════════════════════════════════════════"
            echo "                         Connection Information                             "
            echo "═══════════════════════════════════════════════════════════════════════════"
            echo ""
            echo "  Connect to libvirt from your host:"
            echo ""
            echo "  Option 1 - TCP (simpler, development only):"
            echo "    export LIBVIRT_URI='qemu+tcp://localhost:16509/system'"
            echo "    virsh list --all"
            echo ""
            echo "  Option 2 - SSH (more secure):"
            echo "    export LIBVIRT_URI='qemu+ssh://${USER}@localhost:${SSH_PORT}/system?keyfile=${SSH_KEY}'"
            echo "    virsh list --all"
            echo ""
            echo "  Environment files:"
            echo "    ${REPO_ROOT}/.env (for docker-compose)"
            echo "    ${ENV_FILE} (for local development)"
            echo ""
            echo "  SSH CA (for managed credentials):"
            echo "    ${SSH_CA_DIR}/ssh_ca (private key)"
            echo "    ${SSH_CA_DIR}/ssh_ca.pub (public key - injected into VMs)"
            echo ""
            echo "  Docker Compose (uses .env automatically):"
            echo "    docker-compose up --build"
            echo ""
            echo "  Run the API locally with:"
            echo "    export LIBVIRT_URI='qemu+tcp://localhost:16509/system'"
            echo "    go run ./cmd/api"
            echo ""
            echo "  SSH into Lima VM:"
            echo "    limactl shell ${LIMA_VM_NAME}"
            echo ""
            echo "  Create additional test VMs:"
            echo "    limactl shell ${LIMA_VM_NAME} -- bash /tmp/create-test-vm.sh test-vm-2"
            echo ""
            echo "  Stop/Start Lima VM:"
            echo "    limactl stop ${LIMA_VM_NAME}"
            echo "    limactl start ${LIMA_VM_NAME}"
            echo ""
            echo "═══════════════════════════════════════════════════════════════════════════"
            ;;

        linux-native)
            setup_native_linux

            if [ "${CREATE_TEST_VM}" = true ]; then
                log_info "Creating test VM..."
                TEST_VM_SCRIPT="${SCRIPT_DIR}/create-test-vm.sh"
                generate_test_vm_script "${TEST_VM_SCRIPT}"
                bash "${TEST_VM_SCRIPT}"
            fi

            # Generate simple env file for native Linux
            ENV_FILE="${PROJECT_ROOT}/.env.libvirt"
            cat > "${ENV_FILE}" << ENV_EOF
# virsh-sandbox native libvirt configuration
LIBVIRT_URI="qemu:///system"
BASE_IMAGE_DIR="/var/lib/libvirt/images/base"
SANDBOX_WORKDIR="/var/lib/libvirt/images/jobs"
API_HTTP_ADDR=":8080"
ENV_EOF
            log_success "Environment file created: ${ENV_FILE}"

            log_success "Native Linux setup complete!"
            echo ""
            echo "  Run the API with:"
            echo "    export LIBVIRT_URI='qemu:///system'"
            echo "    go run ./cmd/api"
            ;;
    esac
}

# =============================================================================
# Run Main
# =============================================================================

main "$@"
