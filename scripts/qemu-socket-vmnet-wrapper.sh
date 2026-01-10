#!/bin/bash
# Wrapper script to run QEMU with socket_vmnet networking
# This enables proper VM networking on macOS via socket_vmnet

SOCKET_VMNET_CLIENT="/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet_client"
SOCKET_PATH="/opt/homebrew/var/run/socket_vmnet"
QEMU_BINARY="/opt/homebrew/bin/qemu-system-aarch64"

# Check if socket_vmnet_client exists
if [[ ! -x "$SOCKET_VMNET_CLIENT" ]]; then
    echo "ERROR: socket_vmnet_client not found at $SOCKET_VMNET_CLIENT" >&2
    exit 1
fi

# Check if socket exists
if [[ ! -S "$SOCKET_PATH" ]]; then
    echo "ERROR: socket_vmnet socket not found at $SOCKET_PATH" >&2
    echo "Run: sudo brew services start socket_vmnet" >&2
    exit 1
fi

# Run QEMU through socket_vmnet_client
# socket_vmnet_client connects to the socket and passes fd=3 to QEMU
exec "$SOCKET_VMNET_CLIENT" "$SOCKET_PATH" "$QEMU_BINARY" "$@"
