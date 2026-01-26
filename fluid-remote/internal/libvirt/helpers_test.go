package libvirt

import (
	"strings"
	"testing"
)

func TestModifyClonedXMLHelper_UpdatesCloudInitISO(t *testing.T) {
	// Test that modifyClonedXMLHelper updates existing CDROM device to use new cloud-init ISO
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

	newXML, err := modifyClonedXMLHelper(sourceXML, "sbx-clone123",
		"/var/lib/libvirt/images/jobs/sbx-clone123/disk-overlay.qcow2",
		"/var/lib/libvirt/images/jobs/sbx-clone123/cloud-init.iso",
		2, 2048, "default")
	if err != nil {
		t.Fatalf("modifyClonedXMLHelper() error = %v", err)
	}

	// Should have updated name
	if !strings.Contains(newXML, "<name>sbx-clone123</name>") {
		t.Error("modifyClonedXMLHelper() did not update VM name")
	}

	// Should have updated disk path
	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-clone123/disk-overlay.qcow2") {
		t.Error("modifyClonedXMLHelper() did not update disk path")
	}

	// CRITICAL: Should have updated cloud-init ISO path (not the old /tmp/test-vm-seed.img)
	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-clone123/cloud-init.iso") {
		t.Errorf("modifyClonedXMLHelper() did not update cloud-init ISO path in XML:\n%s", newXML)
	}

	// Should NOT contain the old cloud-init ISO path
	if strings.Contains(newXML, "/tmp/test-vm-seed.img") {
		t.Errorf("modifyClonedXMLHelper() still contains old cloud-init ISO path in XML:\n%s", newXML)
	}

	// UUID should be removed
	if strings.Contains(newXML, "12345678-1234-1234-1234-123456789012") {
		t.Error("modifyClonedXMLHelper() did not remove UUID")
	}

	// MAC address should be different from source
	if strings.Contains(newXML, "52:54:00:11:22:33") {
		t.Error("modifyClonedXMLHelper() did not generate new MAC address")
	}
}

func TestModifyClonedXMLHelper_AddsCloudInitCDROM(t *testing.T) {
	// Test that modifyClonedXMLHelper adds CDROM device when source VM has none
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

	newXML, err := modifyClonedXMLHelper(sourceXML, "sbx-new",
		"/var/lib/libvirt/images/jobs/sbx-new/disk.qcow2",
		"/var/lib/libvirt/images/jobs/sbx-new/cloud-init.iso",
		2, 2048, "default")
	if err != nil {
		t.Fatalf("modifyClonedXMLHelper() error = %v", err)
	}

	// Should have added CDROM device with cloud-init ISO
	if !strings.Contains(newXML, `device="cdrom"`) {
		t.Errorf("modifyClonedXMLHelper() did not add CDROM device in XML:\n%s", newXML)
	}

	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-new/cloud-init.iso") {
		t.Errorf("modifyClonedXMLHelper() did not add cloud-init ISO path in XML:\n%s", newXML)
	}

	// Should have added SCSI controller for the CDROM
	if !strings.Contains(newXML, `type="scsi"`) {
		t.Errorf("modifyClonedXMLHelper() did not add SCSI controller in XML:\n%s", newXML)
	}
}

func TestModifyClonedXMLHelper_NoCloudInitISO(t *testing.T) {
	// Test that modifyClonedXMLHelper works without cloud-init ISO (empty string)
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
	newXML, err := modifyClonedXMLHelper(sourceXML, "sbx-no-cloud",
		"/var/lib/libvirt/images/jobs/sbx-no-cloud/disk.qcow2",
		"", // empty cloud-init ISO
		2, 2048, "default")
	if err != nil {
		t.Fatalf("modifyClonedXMLHelper() error = %v", err)
	}

	// Old CDROM path should still be there (unchanged)
	if !strings.Contains(newXML, "/tmp/old-seed.img") {
		t.Errorf("modifyClonedXMLHelper() modified CDROM when cloudInitISO was empty:\n%s", newXML)
	}

	// Name and disk should still be updated
	if !strings.Contains(newXML, "<name>sbx-no-cloud</name>") {
		t.Error("modifyClonedXMLHelper() did not update VM name")
	}
	if !strings.Contains(newXML, "/var/lib/libvirt/images/jobs/sbx-no-cloud/disk.qcow2") {
		t.Error("modifyClonedXMLHelper() did not update disk path")
	}
}

func TestGenerateMACAddressHelper(t *testing.T) {
	mac := generateMACAddressHelper()

	// Should start with QEMU prefix
	if !strings.HasPrefix(mac, "52:54:00:") {
		t.Errorf("generateMACAddressHelper() = %q, want prefix '52:54:00:'", mac)
	}

	// Should be valid format (17 chars: xx:xx:xx:xx:xx:xx)
	if len(mac) != 17 {
		t.Errorf("generateMACAddressHelper() = %q, want 17 chars", mac)
	}

	// Should have 5 colons
	if strings.Count(mac, ":") != 5 {
		t.Errorf("generateMACAddressHelper() = %q, want 5 colons", mac)
	}

	// Generate another one - should be different (random)
	mac2 := generateMACAddressHelper()
	if mac == mac2 {
		t.Errorf("generateMACAddressHelper() returned same MAC twice: %q", mac)
	}
}

func TestParseDomIfAddrIPv4WithMACHelper(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedIP  string
		expectedMAC string
	}{
		{
			name: "valid output with IP and MAC",
			input: `Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0     52:54:00:ab:cd:ef    ipv4         192.168.122.100/24`,
			expectedIP:  "192.168.122.100",
			expectedMAC: "52:54:00:ab:cd:ef",
		},
		{
			name: "output with multiple interfaces",
			input: `Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet0     52:54:00:11:22:33    ipv4         192.168.122.50/24
 vnet1     52:54:00:aa:bb:cc    ipv4         10.0.0.5/24`,
			expectedIP:  "192.168.122.50",
			expectedMAC: "52:54:00:11:22:33",
		},
		{
			name:        "empty output",
			input:       "",
			expectedIP:  "",
			expectedMAC: "",
		},
		{
			name: "no IPv4 address",
			input: `Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------`,
			expectedIP:  "",
			expectedMAC: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, mac := parseDomIfAddrIPv4WithMACHelper(tt.input)
			if ip != tt.expectedIP {
				t.Errorf("parseDomIfAddrIPv4WithMACHelper() IP = %q, want %q", ip, tt.expectedIP)
			}
			if mac != tt.expectedMAC {
				t.Errorf("parseDomIfAddrIPv4WithMACHelper() MAC = %q, want %q", mac, tt.expectedMAC)
			}
		})
	}
}

func TestParseVMStateHelper(t *testing.T) {
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
			name:     "unknown state",
			output:   "weird-state\n",
			expected: VMStateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVMStateHelper(tt.output)
			if result != tt.expected {
				t.Errorf("parseVMStateHelper(%q) = %v, want %v", tt.output, result, tt.expected)
			}
		})
	}
}
