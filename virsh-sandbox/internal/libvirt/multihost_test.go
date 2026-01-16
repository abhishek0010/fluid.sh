package libvirt

import (
	"testing"

	"virsh-sandbox/internal/config"
)

func TestParseVirshState(t *testing.T) {
	tests := []struct {
		input    string
		expected DomainState
	}{
		{"running", DomainStateRunning},
		{"Running", DomainStateRunning},
		{"RUNNING", DomainStateRunning},
		{"paused", DomainStatePaused},
		{"shut off", DomainStateStopped},
		{"shutdown", DomainStateShutdown},
		{"crashed", DomainStateCrashed},
		{"pmsuspended", DomainStateSuspended},
		{"unknown", DomainStateUnknown},
		{"", DomainStateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseVirshState(tt.input)
			if result != tt.expected {
				t.Errorf("parseVirshState(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDiskPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "standard output",
			input: `Type   Device   Target   Source
------------------------------------------------
file   disk     vda      /var/lib/libvirt/images/test.qcow2
file   cdrom    sda      -`,
			expected: "/var/lib/libvirt/images/test.qcow2",
		},
		{
			name: "multiple disks",
			input: `Type   Device   Target   Source
------------------------------------------------
file   disk     vda      /var/lib/libvirt/images/root.qcow2
file   disk     vdb      /var/lib/libvirt/images/data.qcow2`,
			expected: "/var/lib/libvirt/images/root.qcow2",
		},
		{
			name:     "empty output",
			input:    "",
			expected: "",
		},
		{
			name: "no disks",
			input: `Type   Device   Target   Source
------------------------------------------------`,
			expected: "",
		},
		{
			name: "cdrom only",
			input: `Type   Device   Target   Source
------------------------------------------------
file   cdrom    sda      /path/to/iso.iso`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDiskPath(tt.input)
			if result != tt.expected {
				t.Errorf("parseDiskPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"with'quote", "'with'\"'\"'quote'"},
		{"", "''"},
		{"test-vm-01", "'test-vm-01'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := shellEscape(tt.input)
			if result != tt.expected {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewMultiHostDomainManager(t *testing.T) {
	manager := NewMultiHostDomainManager(nil, nil)
	if manager == nil {
		t.Fatal("NewMultiHostDomainManager returned nil")
	}
	if manager.hosts != nil {
		t.Error("Expected nil hosts slice")
	}
}

func TestGetHosts(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
	}
	manager := NewMultiHostDomainManager(hosts, nil)

	result := manager.GetHosts()
	if len(result) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(result))
	}
	if result[0].Name != "host1" {
		t.Errorf("Expected first host name to be 'host1', got %s", result[0].Name)
	}
}
