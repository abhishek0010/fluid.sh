//go:build libvirt

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

func TestModifyClonedXML_UpdatesCloudInitISO(t *testing.T) {
	// Test that modifyClonedXML updates existing CDROM device to use new cloud-init ISO
	sourceXML := `<domain type='kvm'>
  <name>test-vm</name>
  <uuid>12345678-1234-1234-1234-123456789012</uuid>
  <memory unit='KiB'>2097152</memory>
  <vcpu placement='static'>2</vcpu>
  <os>
    <type arch='x86_64' machine='pc-q35-6.2'>hvm</type>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='/var/lib/libvirt/images/base.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='/tmp/test-vm-seed.img'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>
    <interface type='network'>
      <mac address='52:54:00:11:22:33'/>
      <source network='default'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>
  </devices>
</domain>`

	newXML, err := modifyClonedXML(sourceXML, "sbx-clone123", "/var/lib/libvirt/images/jobs/sbx-clone123/disk-overlay.qcow2", "/var/lib/libvirt/images/jobs/sbx-clone123/cloud-init.iso")
	if err != nil {
		t.Fatalf("modifyClonedXML() error = %v", err)
	}

	// Should have updated name
	if !strings.Contains(newXML, "<name>sbx-clone123</name>") {
		t.Error("modifyClonedXML() did not update VM name")
	}

	// Should have updated disk path
	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-clone123/disk-overlay.qcow2") {
		t.Error("modifyClonedXML() did not update disk path")
	}

	// CRITICAL: Should have updated cloud-init ISO path (not the old /tmp/test-vm-seed.img)
	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-clone123/cloud-init.iso") {
		t.Errorf("modifyClonedXML() did not update cloud-init ISO path in XML:\n%s", newXML)
	}

	// Should NOT contain the old cloud-init ISO path
	if strings.Contains(newXML, "/tmp/test-vm-seed.img") {
		t.Errorf("modifyClonedXML() still contains old cloud-init ISO path in XML:\n%s", newXML)
	}

	// UUID should be removed
	if strings.Contains(newXML, "12345678-1234-1234-1234-123456789012") {
		t.Error("modifyClonedXML() did not remove UUID")
	}

	// MAC address should be different from source
	if strings.Contains(newXML, "52:54:00:11:22:33") {
		t.Error("modifyClonedXML() did not generate new MAC address")
	}
}

func TestModifyClonedXML_AddsCloudInitCDROM(t *testing.T) {
	// Test that modifyClonedXML adds CDROM device when source VM has none
	sourceXML := `<domain type='kvm'>
  <name>test-vm-no-cdrom</name>
  <uuid>12345678-1234-1234-1234-123456789012</uuid>
  <memory unit='KiB'>2097152</memory>
  <vcpu placement='static'>2</vcpu>
  <os>
    <type arch='x86_64' machine='pc-q35-6.2'>hvm</type>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='/var/lib/libvirt/images/base.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <interface type='network'>
      <mac address='52:54:00:11:22:33'/>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
  </devices>
</domain>`

	newXML, err := modifyClonedXML(sourceXML, "sbx-new", "/var/lib/libvirt/images/jobs/sbx-new/disk.qcow2", "/var/lib/libvirt/images/jobs/sbx-new/cloud-init.iso")
	if err != nil {
		t.Fatalf("modifyClonedXML() error = %v", err)
	}

	// Should have added CDROM device with cloud-init ISO
	if !strings.Contains(newXML, `device="cdrom"`) {
		t.Errorf("modifyClonedXML() did not add CDROM device in XML:\n%s", newXML)
	}

	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-new/cloud-init.iso") {
		t.Errorf("modifyClonedXML() did not add cloud-init ISO path in XML:\n%s", newXML)
	}

	// Should have added SCSI controller for the CDROM
	if !strings.Contains(newXML, `type="scsi"`) {
		t.Errorf("modifyClonedXML() did not add SCSI controller in XML:\n%s", newXML)
	}
}

func TestModifyClonedXML_NoCloudInitISO(t *testing.T) {
	// Test that modifyClonedXML works without cloud-init ISO (empty string)
	sourceXML := `<domain type='kvm'>
  <name>test-vm</name>
  <uuid>12345678-1234-1234-1234-123456789012</uuid>
  <memory unit='KiB'>2097152</memory>
  <vcpu placement='static'>2</vcpu>
  <os>
    <type arch='x86_64' machine='pc-q35-6.2'>hvm</type>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='/var/lib/libvirt/images/base.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='/tmp/old-seed.img'/>
      <target dev='sda' bus='sata'/>
    </disk>
    <interface type='network'>
      <mac address='52:54:00:11:22:33'/>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
  </devices>
</domain>`

	// Empty cloudInitISO - should not modify CDROM
	newXML, err := modifyClonedXML(sourceXML, "sbx-no-cloud", "/var/lib/libvirt/images/jobs/sbx-no-cloud/disk.qcow2", "")
	if err != nil {
		t.Fatalf("modifyClonedXML() error = %v", err)
	}

	// Old CDROM path should still be there (unchanged)
	if !strings.Contains(newXML, "/tmp/old-seed.img") {
		t.Errorf("modifyClonedXML() modified CDROM when cloudInitISO was empty:\n%s", newXML)
	}

	// Name and disk should still be updated
	if !strings.Contains(newXML, "<name>sbx-no-cloud</name>") {
		t.Error("modifyClonedXML() did not update VM name")
	}
	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-no-cloud/disk.qcow2") {
		t.Error("modifyClonedXML() did not update disk path")
	}
}
