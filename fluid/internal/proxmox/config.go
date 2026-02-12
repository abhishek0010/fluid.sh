package proxmox

import "fmt"

// Config holds all settings needed to connect to a Proxmox VE cluster.
type Config struct {
	Host      string // Base URL, e.g., "https://pve.example.com:8006"
	TokenID   string // API token ID, e.g., "root@pam!fluid"
	Secret    string // API token secret
	Node      string // Target node name, e.g., "pve1"
	VerifySSL bool   // Verify TLS certificates
	Storage   string // Storage for VM disks, e.g., "local-lvm"
	Bridge    string // Network bridge, e.g., "vmbr0"
	CloneMode string // "full" or "linked"
	VMIDStart int    // Start of VMID range for sandboxes
	VMIDEnd   int    // End of VMID range for sandboxes
}

// Validate checks that required config fields are set.
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("proxmox host is required")
	}
	if c.TokenID == "" {
		return fmt.Errorf("proxmox token_id is required")
	}
	if c.Secret == "" {
		return fmt.Errorf("proxmox secret is required")
	}
	if c.Node == "" {
		return fmt.Errorf("proxmox node is required")
	}
	if c.VMIDStart <= 0 {
		c.VMIDStart = 9000
	}
	if c.VMIDEnd <= 0 {
		c.VMIDEnd = 9999
	}
	if c.VMIDEnd <= c.VMIDStart {
		return fmt.Errorf("proxmox vmid_end (%d) must be greater than vmid_start (%d)", c.VMIDEnd, c.VMIDStart)
	}
	if c.CloneMode == "" {
		c.CloneMode = "full"
	}
	if c.CloneMode != "full" && c.CloneMode != "linked" {
		return fmt.Errorf("proxmox clone_mode must be 'full' or 'linked', got %q", c.CloneMode)
	}
	return nil
}
