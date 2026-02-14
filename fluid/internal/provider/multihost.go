package provider

import "context"

// MultiHostVMInfo extends VM info with host identification.
type MultiHostVMInfo struct {
	Name        string
	UUID        string
	State       string
	Persistent  bool
	DiskPath    string
	HostName    string // Display name of the host
	HostAddress string // IP or hostname of the host
}

// HostError represents an error from querying a specific host.
type HostError struct {
	HostName    string `json:"host_name"`
	HostAddress string `json:"host_address"`
	Error       string `json:"error"`
}

// MultiHostListResult contains the aggregated result from querying all hosts.
type MultiHostListResult struct {
	VMs        []*MultiHostVMInfo
	HostErrors []HostError
}

// MultiHostLister can list VMs across multiple hosts and find which host has a given VM.
type MultiHostLister interface {
	ListVMs(ctx context.Context) (*MultiHostListResult, error)
	FindHostForVM(ctx context.Context, vmName string) (hostName, hostAddress string, err error)
}
