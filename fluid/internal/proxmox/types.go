package proxmox

// VMStatus represents the status of a QEMU VM from the Proxmox API.
type VMStatus struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"` // "running", "stopped", "paused"
	QMPStatus string  `json:"qmpstatus,omitempty"`
	CPU       float64 `json:"cpu"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	MaxDisk   int64   `json:"maxdisk"`
	Uptime    int64   `json:"uptime"`
	PID       int     `json:"pid,omitempty"`
	Template  int     `json:"template,omitempty"` // 1 if template
	Lock      string  `json:"lock,omitempty"`     // "clone", "migrate", etc.
}

// VMConfig represents a VM's configuration from the Proxmox API.
type VMConfig struct {
	Name      string `json:"name"`
	Memory    int    `json:"memory"`
	Cores     int    `json:"cores"`
	Sockets   int    `json:"sockets"`
	CPU       string `json:"cpu"`
	Net0      string `json:"net0,omitempty"`
	IDE2      string `json:"ide2,omitempty"` // cloud-init drive
	SCSI0     string `json:"scsi0,omitempty"`
	VirtIO0   string `json:"virtio0,omitempty"`
	Boot      string `json:"boot,omitempty"`
	Agent     string `json:"agent,omitempty"` // "1" if QEMU guest agent enabled
	IPConfig0 string `json:"ipconfig0,omitempty"`
	SSHKeys   string `json:"sshkeys,omitempty"`
	CIUser    string `json:"ciuser,omitempty"`
}

// NodeStatus represents a Proxmox node's resource status.
type NodeStatus struct {
	CPU      float64      `json:"cpu"`
	MaxCPU   int          `json:"maxcpu"`
	Memory   MemoryStatus `json:"memory"`
	RootFS   DiskStatus   `json:"rootfs"`
	Uptime   int64        `json:"uptime"`
	KVersion string       `json:"kversion"`
}

// MemoryStatus is memory info from node status.
type MemoryStatus struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
	Free  int64 `json:"free"`
}

// DiskStatus is disk info from node status.
type DiskStatus struct {
	Total     int64 `json:"total"`
	Used      int64 `json:"used"`
	Available int64 `json:"avail"`
}

// NetworkInterface represents a network interface from the QEMU guest agent.
type NetworkInterface struct {
	Name            string           `json:"name"`
	HardwareAddress string           `json:"hardware-address"`
	IPAddresses     []GuestIPAddress `json:"ip-addresses"`
}

// GuestIPAddress is an IP address from the QEMU guest agent.
type GuestIPAddress struct {
	IPAddressType string `json:"ip-address-type"` // "ipv4" or "ipv6"
	IPAddress     string `json:"ip-address"`
	Prefix        int    `json:"prefix"`
}

// TaskStatus represents the status of an asynchronous Proxmox task.
type TaskStatus struct {
	Status     string `json:"status"`               // "running", "stopped"
	ExitStatus string `json:"exitstatus,omitempty"` // "OK" on success
	Type       string `json:"type"`
	ID         string `json:"id"`
	Node       string `json:"node"`
	PID        int    `json:"pid"`
	StartTime  int64  `json:"starttime"`
	EndTime    int64  `json:"endtime,omitempty"`
}

// VMListEntry represents a VM in the list returned by GET /nodes/{node}/qemu.
type VMListEntry struct {
	VMID     int     `json:"vmid"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Template int     `json:"template,omitempty"`
	MaxMem   int64   `json:"maxmem"`
	MaxDisk  int64   `json:"maxdisk"`
	CPU      float64 `json:"cpu"`
	Mem      int64   `json:"mem"`
	Uptime   int64   `json:"uptime"`
}

// SnapshotEntry represents a snapshot from Proxmox.
type SnapshotEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SnapTime    int64  `json:"snaptime,omitempty"`
	Parent      string `json:"parent,omitempty"`
	VMState     int    `json:"vmstate,omitempty"` // 1 if includes RAM state
}
