#!/bin/bash

# Get a list of all VMs
VM_LIST=$(virsh list --all --name)

# Loop through each VM
for VM in $VM_LIST; do
    if [ "$VM" != "test-vm" ]; then
        echo "Stopping and removing VM: $VM"
        # Stop the VM if it's running
        virsh destroy "$VM" 2>/dev/null
        # Undefine the VM
        virsh undefine "$VM"
    else
        echo "Skipping VM: $VM"
    fi
done

echo "Done."
