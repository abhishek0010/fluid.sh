package provider

// VMRef is a minimal reference to a VM.
type VMRef struct {
	Name string
	UUID string
}

// SnapshotRef references a snapshot created for a VM.
type SnapshotRef struct {
	Name string
	// Kind: "INTERNAL" or "EXTERNAL"
	Kind string
	// Ref is driver-specific; could be an internal UUID or a file path.
	Ref string
}

// FSComparePlan describes a plan for diffing two snapshots' filesystems.
type FSComparePlan struct {
	VMName       string
	FromSnapshot string
	ToSnapshot   string

	// Best-effort mount points (if prepared); may be empty strings.
	FromMount string
	ToMount   string

	// Devices or files used; informative.
	FromRef string
	ToRef   string

	// Free-form notes with instructions if the manager couldn't mount automatically.
	Notes []string
}

// VMState represents possible VM states.
type VMState string

const (
	VMStateRunning   VMState = "running"
	VMStatePaused    VMState = "paused"
	VMStateShutOff   VMState = "shut off"
	VMStateCrashed   VMState = "crashed"
	VMStateSuspended VMState = "pmsuspended"
	VMStateUnknown   VMState = "unknown"
)

// VMValidationResult contains the results of validating a source VM.
type VMValidationResult struct {
	VMName     string   `json:"vm_name"`
	Valid      bool     `json:"valid"`
	State      VMState  `json:"state"`
	MACAddress string   `json:"mac_address,omitempty"`
	IPAddress  string   `json:"ip_address,omitempty"`
	HasNetwork bool     `json:"has_network"`
	Warnings   []string `json:"warnings,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

// ResourceCheckResult contains the results of checking host resources.
type ResourceCheckResult struct {
	Valid               bool     `json:"valid"`
	AvailableMemoryMB   int64    `json:"available_memory_mb"`
	TotalMemoryMB       int64    `json:"total_memory_mb"`
	AvailableCPUs       int      `json:"available_cpus"`
	TotalCPUs           int      `json:"total_cpus"`
	AvailableDiskMB     int64    `json:"available_disk_mb"`
	RequiredMemoryMB    int      `json:"required_memory_mb"`
	RequiredCPUs        int      `json:"required_cpus"`
	NeedsCPUApproval    bool     `json:"needs_cpu_approval"`
	NeedsMemoryApproval bool     `json:"needs_memory_approval"`
	Warnings            []string `json:"warnings,omitempty"`
	Errors              []string `json:"errors,omitempty"`
}
