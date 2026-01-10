//go:build libvirt
// +build libvirt

package libvirt

import (
	"strings"
	"testing"
)

func TestRenderDomainXML_CPUMode(t *testing.T) {
	tests := []struct {
		name            string
		params          domainXMLParams
		expectedCPUMode string
	}{
		{
			name: "x86_64 with kvm uses host-passthrough",
			params: domainXMLParams{
				Name:       "test-vm",
				MemoryMB:   1024,
				VCPUs:      2,
				DiskPath:   "/var/lib/libvirt/images/test.qcow2",
				Network:    "default",
				Arch:       "x86_64",
				Machine:    "pc-q35-6.2",
				DomainType: "kvm",
			},
			expectedCPUMode: `<cpu mode="host-passthrough"/>`,
		},
		{
			name: "x86_64 with qemu uses host-passthrough",
			params: domainXMLParams{
				Name:       "test-vm",
				MemoryMB:   1024,
				VCPUs:      2,
				DiskPath:   "/var/lib/libvirt/images/test.qcow2",
				Network:    "default",
				Arch:       "x86_64",
				Machine:    "pc-q35-6.2",
				DomainType: "qemu",
			},
			expectedCPUMode: `<cpu mode="host-passthrough"/>`,
		},
		{
			name: "aarch64 with kvm uses host-passthrough",
			params: domainXMLParams{
				Name:       "test-vm",
				MemoryMB:   1024,
				VCPUs:      2,
				DiskPath:   "/var/lib/libvirt/images/test.qcow2",
				Network:    "default",
				Arch:       "aarch64",
				Machine:    "virt",
				DomainType: "kvm",
			},
			expectedCPUMode: `<cpu mode="host-passthrough"/>`,
		},
		{
			name: "aarch64 with qemu uses custom cortex-a72 model",
			params: domainXMLParams{
				Name:       "test-vm",
				MemoryMB:   1024,
				VCPUs:      2,
				DiskPath:   "/var/lib/libvirt/images/test.qcow2",
				Network:    "default",
				Arch:       "aarch64",
				Machine:    "virt",
				DomainType: "qemu",
			},
			expectedCPUMode: `<cpu mode="custom" match="exact">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xml, err := renderDomainXML(tt.params)
			if err != nil {
				t.Fatalf("renderDomainXML() error = %v", err)
			}

			if !strings.Contains(xml, tt.expectedCPUMode) {
				t.Errorf("renderDomainXML() expected CPU mode %q not found in XML:\n%s", tt.expectedCPUMode, xml)
			}
		})
	}
}

func TestRenderDomainXML_BasicStructure(t *testing.T) {
	params := domainXMLParams{
		Name:       "test-sandbox",
		MemoryMB:   2048,
		VCPUs:      4,
		DiskPath:   "/var/lib/libvirt/images/test-sandbox.qcow2",
		Network:    "default",
		Arch:       "x86_64",
		Machine:    "pc-q35-6.2",
		DomainType: "kvm",
	}

	xml, err := renderDomainXML(params)
	if err != nil {
		t.Fatalf("renderDomainXML() error = %v", err)
	}

	expectedElements := []string{
		`<domain type="kvm">`,
		`<name>test-sandbox</name>`,
		`<memory unit="MiB">2048</memory>`,
		`<vcpu placement="static">4</vcpu>`,
		`<type arch="x86_64" machine="pc-q35-6.2">hvm</type>`,
		`<source file="/var/lib/libvirt/images/test-sandbox.qcow2"/>`,
		`<source network="default"/>`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xml, expected) {
			t.Errorf("renderDomainXML() expected element %q not found in XML:\n%s", expected, xml)
		}
	}
}

func TestRenderDomainXML_Aarch64Features(t *testing.T) {
	params := domainXMLParams{
		Name:       "test-arm-vm",
		MemoryMB:   1024,
		VCPUs:      2,
		DiskPath:   "/var/lib/libvirt/images/test.qcow2",
		Network:    "default",
		Arch:       "aarch64",
		Machine:    "virt",
		DomainType: "qemu",
	}

	xml, err := renderDomainXML(params)
	if err != nil {
		t.Fatalf("renderDomainXML() error = %v", err)
	}

	// aarch64-specific elements
	expectedElements := []string{
		`<os firmware="efi">`,
		`<gic version="2"/>`,
		`<controller type="usb" model="qemu-xhci"/>`,
		`<cpu mode="custom" match="exact">`,
		`<model fallback="allow">cortex-a72</model>`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xml, expected) {
			t.Errorf("renderDomainXML() expected aarch64 element %q not found in XML:\n%s", expected, xml)
		}
	}

	// x86_64-specific elements should NOT be present
	unexpectedElements := []string{
		`<apic/>`,
		`<pae/>`,
		`<input type="tablet" bus="usb"/>`,
	}

	for _, unexpected := range unexpectedElements {
		if strings.Contains(xml, unexpected) {
			t.Errorf("renderDomainXML() unexpected x86_64 element %q found in aarch64 XML:\n%s", unexpected, xml)
		}
	}
}

func TestRenderDomainXML_X86Features(t *testing.T) {
	params := domainXMLParams{
		Name:       "test-x86-vm",
		MemoryMB:   1024,
		VCPUs:      2,
		DiskPath:   "/var/lib/libvirt/images/test.qcow2",
		Network:    "default",
		Arch:       "x86_64",
		Machine:    "pc-q35-6.2",
		DomainType: "kvm",
	}

	xml, err := renderDomainXML(params)
	if err != nil {
		t.Fatalf("renderDomainXML() error = %v", err)
	}

	// x86_64-specific elements
	expectedElements := []string{
		`<apic/>`,
		`<pae/>`,
		`<input type="tablet" bus="usb"/>`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xml, expected) {
			t.Errorf("renderDomainXML() expected x86_64 element %q not found in XML:\n%s", expected, xml)
		}
	}

	// aarch64-specific elements should NOT be present
	unexpectedElements := []string{
		`<os firmware="efi">`,
		`<gic version="2"/>`,
		`<controller type="usb" model="qemu-xhci"/>`,
	}

	for _, unexpected := range unexpectedElements {
		if strings.Contains(xml, unexpected) {
			t.Errorf("renderDomainXML() unexpected aarch64 element %q found in x86_64 XML:\n%s", unexpected, xml)
		}
	}
}

func TestRenderDomainXML_Defaults(t *testing.T) {
	// Test that defaults are applied when fields are empty
	params := domainXMLParams{
		Name:     "test-defaults",
		MemoryMB: 512,
		VCPUs:    1,
		DiskPath: "/var/lib/libvirt/images/test.qcow2",
		Network:  "default",
		// Arch, Machine, and DomainType are empty - should use defaults
	}

	xml, err := renderDomainXML(params)
	if err != nil {
		t.Fatalf("renderDomainXML() error = %v", err)
	}

	// Should default to x86_64, pc-q35-6.2, kvm
	expectedDefaults := []string{
		`<domain type="kvm">`,
		`<type arch="x86_64" machine="pc-q35-6.2">hvm</type>`,
		`<cpu mode="host-passthrough"/>`,
	}

	for _, expected := range expectedDefaults {
		if !strings.Contains(xml, expected) {
			t.Errorf("renderDomainXML() expected default element %q not found in XML:\n%s", expected, xml)
		}
	}
}

func TestRenderDomainXML_WithCloudInitISO(t *testing.T) {
	// Test that cloud-init ISO is properly included in domain XML
	params := domainXMLParams{
		Name:         "test-cloud-init",
		MemoryMB:     2048,
		VCPUs:        2,
		DiskPath:     "/var/lib/libvirt/images/jobs/test-cloud-init/disk-overlay.qcow2",
		CloudInitISO: "/var/lib/libvirt/images/jobs/test-cloud-init/cloud-init.iso",
		Network:      "default",
		Arch:         "aarch64",
		Machine:      "virt",
		DomainType:   "qemu",
	}

	xml, err := renderDomainXML(params)
	if err != nil {
		t.Fatalf("renderDomainXML() error = %v", err)
	}

	// Cloud-init ISO elements should be present
	expectedElements := []string{
		`<disk type="file" device="cdrom">`,
		`<source file="/var/lib/libvirt/images/jobs/test-cloud-init/cloud-init.iso"/>`,
		`<target dev="sda" bus="scsi"/>`,
		`<readonly/>`,
		`<controller type="scsi" model="virtio-scsi"/>`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xml, expected) {
			t.Errorf("renderDomainXML() expected cloud-init element %q not found in XML:\n%s", expected, xml)
		}
	}
}

func TestRenderDomainXML_WithoutCloudInitISO(t *testing.T) {
	// Test that no cloud-init CDROM is included when CloudInitISO is empty
	params := domainXMLParams{
		Name:       "test-no-cloud-init",
		MemoryMB:   2048,
		VCPUs:      2,
		DiskPath:   "/var/lib/libvirt/images/jobs/test/disk-overlay.qcow2",
		Network:    "default",
		Arch:       "x86_64",
		Machine:    "pc-q35-6.2",
		DomainType: "kvm",
		// CloudInitISO is empty
	}

	xml, err := renderDomainXML(params)
	if err != nil {
		t.Fatalf("renderDomainXML() error = %v", err)
	}

	// Cloud-init ISO elements should NOT be present
	unexpectedElements := []string{
		`device="cdrom"`,
		`<controller type="scsi" model="virtio-scsi"/>`,
	}

	for _, unexpected := range unexpectedElements {
		if strings.Contains(xml, unexpected) {
			t.Errorf("renderDomainXML() unexpected cloud-init element %q found in XML when CloudInitISO is empty:\n%s", unexpected, xml)
		}
	}

	// Main disk should still be present
	if !strings.Contains(xml, `<disk type="file" device="disk">`) {
		t.Error("renderDomainXML() main disk not found in XML")
	}
}

func TestCloudInitSeedForClone_UniqueInstanceID(t *testing.T) {
	// This test verifies the concept that each clone should get a unique instance-id
	// The actual buildCloudInitSeedForClone function creates files, so we test the
	// expected behavior through the domain XML params

	vmNames := []string{"sbx-abc123", "sbx-def456", "sbx-ghi789"}

	for _, vmName := range vmNames {
		params := domainXMLParams{
			Name:         vmName,
			MemoryMB:     1024,
			VCPUs:        1,
			DiskPath:     "/var/lib/libvirt/images/jobs/" + vmName + "/disk-overlay.qcow2",
			CloudInitISO: "/var/lib/libvirt/images/jobs/" + vmName + "/cloud-init.iso",
			Network:      "default",
			Arch:         "aarch64",
			Machine:      "virt",
			DomainType:   "qemu",
		}

		xml, err := renderDomainXML(params)
		if err != nil {
			t.Fatalf("renderDomainXML() for %s error = %v", vmName, err)
		}

		// Each sandbox should have its own cloud-init ISO path
		expectedISOPath := "/var/lib/libvirt/images/jobs/" + vmName + "/cloud-init.iso"
		if !strings.Contains(xml, expectedISOPath) {
			t.Errorf("renderDomainXML() for %s expected ISO path %q not found in XML", vmName, expectedISOPath)
		}
	}
}

func TestRenderDomainXML_UserModeNetworking(t *testing.T) {
	tests := []struct {
		name     string
		network  string
		wantUser bool // true if we expect user-mode networking
	}{
		{
			name:     "user network value",
			network:  "user",
			wantUser: true,
		},
		{
			name:     "empty network value",
			network:  "",
			wantUser: true,
		},
		{
			name:     "default network value",
			network:  "default",
			wantUser: false,
		},
		{
			name:     "custom network value",
			network:  "br0",
			wantUser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := domainXMLParams{
				Name:       "test-vm",
				MemoryMB:   2048,
				VCPUs:      2,
				DiskPath:   "/var/lib/libvirt/images/test.qcow2",
				Network:    tt.network,
				Arch:       "aarch64",
				Machine:    "virt",
				DomainType: "qemu",
			}

			xml, err := renderDomainXML(params)
			if err != nil {
				t.Fatalf("renderDomainXML() error = %v", err)
			}

			hasUserInterface := strings.Contains(xml, `<interface type="user">`)
			hasNetworkInterface := strings.Contains(xml, `<interface type="network">`)

			if tt.wantUser {
				if !hasUserInterface {
					t.Errorf("expected user-mode networking but got network interface in XML:\n%s", xml)
				}
				if hasNetworkInterface {
					t.Errorf("expected user-mode networking but found network interface in XML:\n%s", xml)
				}
			} else {
				if hasUserInterface {
					t.Errorf("expected network interface but got user-mode networking in XML:\n%s", xml)
				}
				if !hasNetworkInterface {
					t.Errorf("expected network interface but not found in XML:\n%s", xml)
				}
				// Also verify the network name is correct
				expectedSource := `<source network="` + tt.network + `"/>`
				if !strings.Contains(xml, expectedSource) {
					t.Errorf("expected network source %q not found in XML:\n%s", expectedSource, xml)
				}
			}
		})
	}
}

func TestRenderDomainXML_SocketVMNet(t *testing.T) {
	params := domainXMLParams{
		Name:            "test-socket-vmnet",
		MemoryMB:        2048,
		VCPUs:           2,
		DiskPath:        "/var/lib/libvirt/images/test.qcow2",
		Network:         "socket_vmnet",
		SocketVMNetPath: "/opt/homebrew/var/run/socket_vmnet",
		Emulator:        "/path/to/qemu-wrapper.sh",
		MACAddress:      "52:54:00:ab:cd:ef",
		Arch:            "aarch64",
		Machine:         "virt",
		DomainType:      "qemu",
	}

	xml, err := renderDomainXML(params)
	if err != nil {
		t.Fatalf("renderDomainXML() error = %v", err)
	}

	// Should have qemu namespace
	if !strings.Contains(xml, `xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0"`) {
		t.Error("expected qemu namespace for socket_vmnet")
	}

	// Should have custom emulator
	if !strings.Contains(xml, `<emulator>/path/to/qemu-wrapper.sh</emulator>`) {
		t.Errorf("expected custom emulator in XML:\n%s", xml)
	}

	// Should have qemu:commandline with socket networking
	if !strings.Contains(xml, `<qemu:commandline>`) {
		t.Error("expected qemu:commandline for socket_vmnet")
	}

	// Should have socket,fd=3 netdev
	if !strings.Contains(xml, `socket,id=vnet,fd=3`) {
		t.Errorf("expected socket,fd=3 netdev in XML:\n%s", xml)
	}

	// Should have MAC address
	if !strings.Contains(xml, `mac=52:54:00:ab:cd:ef`) {
		t.Errorf("expected MAC address in XML:\n%s", xml)
	}

	// Should NOT have standard interface element
	if strings.Contains(xml, `<interface type="network">`) || strings.Contains(xml, `<interface type="user">`) {
		t.Errorf("unexpected standard interface in socket_vmnet XML:\n%s", xml)
	}
}

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "52:54:00:ab:cd:ef",
			expected: "52:54:00:ab:cd:ef",
		},
		{
			name:     "shortened octets",
			input:    "52:54:0:ab:cd:ef",
			expected: "52:54:00:ab:cd:ef",
		},
		{
			name:     "multiple shortened octets",
			input:    "52:54:0:a:c:e",
			expected: "52:54:00:0a:0c:0e",
		},
		{
			name:     "invalid format",
			input:    "not-a-mac",
			expected: "not-a-mac",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeMAC(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeMAC(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateMACAddress(t *testing.T) {
	mac := generateMACAddress()

	// Should start with QEMU prefix
	if !strings.HasPrefix(mac, "52:54:00:") {
		t.Errorf("generateMACAddress() = %q, want prefix '52:54:00:'", mac)
	}

	// Should be valid format (17 chars: xx:xx:xx:xx:xx:xx)
	if len(mac) != 17 {
		t.Errorf("generateMACAddress() = %q, want 17 chars", mac)
	}

	// Should have 5 colons
	if strings.Count(mac, ":") != 5 {
		t.Errorf("generateMACAddress() = %q, want 5 colons", mac)
	}

	// Generate another one - should be different (random)
	mac2 := generateMACAddress()
	if mac == mac2 {
		t.Errorf("generateMACAddress() returned same MAC twice: %q", mac)
	}
}

func TestParseVMState(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected VMState
	}{
		{
			name:     "running state",
			output:   "running\n",
			expected: VMStateRunning,
		},
		{
			name:     "running state without newline",
			output:   "running",
			expected: VMStateRunning,
		},
		{
			name:     "shut off state",
			output:   "shut off\n",
			expected: VMStateShutOff,
		},
		{
			name:     "paused state",
			output:   "paused\n",
			expected: VMStatePaused,
		},
		{
			name:     "crashed state",
			output:   "crashed\n",
			expected: VMStateCrashed,
		},
		{
			name:     "pmsuspended state",
			output:   "pmsuspended\n",
			expected: VMStateSuspended,
		},
		{
			name:     "unknown state",
			output:   "some-unknown-state\n",
			expected: VMStateUnknown,
		},
		{
			name:     "empty string",
			output:   "",
			expected: VMStateUnknown,
		},
		{
			name:     "whitespace only",
			output:   "   \n",
			expected: VMStateUnknown,
		},
		{
			name:     "running with extra whitespace",
			output:   "  running  \n",
			expected: VMStateRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVMState(tt.output)
			if result != tt.expected {
				t.Errorf("parseVMState(%q) = %v, want %v", tt.output, result, tt.expected)
			}
		})
	}
}

func TestVMState_StringValues(t *testing.T) {
	// Verify that VMState constants have the expected string values
	tests := []struct {
		state    VMState
		expected string
	}{
		{VMStateRunning, "running"},
		{VMStateShutOff, "shut off"},
		{VMStatePaused, "paused"},
		{VMStateCrashed, "crashed"},
		{VMStateSuspended, "pmsuspended"},
		{VMStateUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("VMState constant %v has value %q, want %q", tt.state, string(tt.state), tt.expected)
			}
		})
	}
}
