Local Mac development:
  - Run Go API directly on Mac (not in Docker)
  - Connect via local socket: qemu:///session
  - No SSH needed, no network config, simplest setup

  Production:
  - Run Go API anywhere (container, VM, bare metal)
  - Connect via SSH: qemu+ssh://user@libvirt-host/system
  - No libvirtd config changes needed on production machines
  - Just needs SSH access with appropriate permissions

  The SSH approach in production is the standard way to manage remote libvirt - it's what tools like virt-manager use. Your Foreman-managed servers already have libvirt running; you just need SSH credentials to reach them.

  For multi-host support, you'd either:
  1. Run one API instance per libvirt host (simplest)
  2. Modify the API to accept a host parameter and maintain connections to multiple libvirt instances (more complex but single endpoint)
