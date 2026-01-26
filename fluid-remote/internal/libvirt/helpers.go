package libvirt

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// generateMACAddressHelper generates a random MAC address with the locally administered bit set.
// Uses the 52:54:00 prefix which is commonly used by QEMU/KVM.
func generateMACAddressHelper() string {
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", buf[0], buf[1], buf[2])
}

// modifyClonedXMLHelper takes the XML from a source domain and adapts it for a new cloned domain.
// It sets a new name, UUID, disk path, MAC address, and cloud-init ISO path.
// If cloudInitISO is provided, any existing CDROM device is updated to use it, ensuring the
// cloned VM gets a unique instance-id and fresh network configuration via cloud-init.
func modifyClonedXMLHelper(sourceXML, newName, newDiskPath, cloudInitISO string, cpu, memoryMB int, network string) (string, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(sourceXML); err != nil {
		return "", fmt.Errorf("parse source XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return "", fmt.Errorf("invalid XML: no root element")
	}

	// Update VM name
	nameElem := root.SelectElement("name")
	if nameElem == nil {
		return "", fmt.Errorf("invalid XML: missing <name> element")
	}
	nameElem.SetText(newName)

	// Remove UUID
	if uuidElem := root.SelectElement("uuid"); uuidElem != nil {
		root.RemoveChild(uuidElem)
	}

	// Update CPU
	if cpu > 0 {
		if vcpuElem := root.SelectElement("vcpu"); vcpuElem != nil {
			vcpuElem.SetText(strconv.Itoa(cpu))
		}
	}

	// Update Memory
	if memoryMB > 0 {
		memKiB := strconv.Itoa(memoryMB * 1024)
		if memElem := root.SelectElement("memory"); memElem != nil {
			memElem.SetText(memKiB)
		}
		if currMemElem := root.SelectElement("currentMemory"); currMemElem != nil {
			currMemElem.SetText(memKiB)
		}
	}

	// Update disk path for the main virtual disk (vda)
	var diskReplaced bool
	for _, disk := range root.FindElements("./devices/disk[@device='disk']") {
		if target := disk.SelectElement("target"); target != nil {
			if bus := target.SelectAttr("bus"); bus != nil && bus.Value == "virtio" {
				if source := disk.SelectElement("source"); source != nil {
					source.SelectAttr("file").Value = newDiskPath
					diskReplaced = true
					break
				}
			}
		}
	}
	if !diskReplaced {
		return "", fmt.Errorf("could not find a virtio disk in the source XML to replace")
	}

	// Handle cloud-init CDROM: update existing or add new one
	// This is critical for cloned VMs - they need a unique instance-id to trigger
	// cloud-init re-initialization, including DHCP network configuration
	if cloudInitISO != "" {
		devices := root.SelectElement("devices")
		if devices == nil {
			return "", fmt.Errorf("invalid XML: missing <devices> element")
		}

		// Look for existing CDROM device to update
		var cdromUpdated bool
		for _, disk := range root.FindElements("./devices/disk[@device='cdrom']") {
			if source := disk.SelectElement("source"); source != nil {
				if fileAttr := source.SelectAttr("file"); fileAttr != nil {
					fileAttr.Value = cloudInitISO
					cdromUpdated = true
					break
				}
			}
		}

		// If no existing CDROM, add one with SCSI controller
		if !cdromUpdated {
			// Add SCSI controller if not present
			hasScsiController := false
			for _, ctrl := range root.FindElements("./devices/controller[@type='scsi']") {
				if model := ctrl.SelectAttr("model"); model != nil && model.Value == "virtio-scsi" {
					hasScsiController = true
					break
				}
			}
			if !hasScsiController {
				scsiCtrl := devices.CreateElement("controller")
				scsiCtrl.CreateAttr("type", "scsi")
				scsiCtrl.CreateAttr("model", "virtio-scsi")
			}

			// Add CDROM device
			cdrom := devices.CreateElement("disk")
			cdrom.CreateAttr("type", "file")
			cdrom.CreateAttr("device", "cdrom")

			driver := cdrom.CreateElement("driver")
			driver.CreateAttr("name", "qemu")
			driver.CreateAttr("type", "raw")

			source := cdrom.CreateElement("source")
			source.CreateAttr("file", cloudInitISO)

			target := cdrom.CreateElement("target")
			target.CreateAttr("dev", "sda")
			target.CreateAttr("bus", "scsi")

			cdrom.CreateElement("readonly")
		}
	}

	// Update network interface: set new MAC and remove PCI address
	if iface := root.FindElement("./devices/interface"); iface != nil {
		macElem := iface.SelectElement("mac")
		if macElem != nil {
			if addrAttr := macElem.SelectAttr("address"); addrAttr != nil {
				addrAttr.Value = generateMACAddressHelper()
			}
		} else {
			macElem = iface.CreateElement("mac")
			macElem.CreateAttr("address", generateMACAddressHelper())
		}

		if addrElem := iface.SelectElement("address"); addrElem != nil {
			iface.RemoveChild(addrElem)
		}

		// Update network source if provided
		if network != "" && iface.SelectAttrValue("type", "") == "network" {
			if source := iface.SelectElement("source"); source != nil {
				if netAttr := source.SelectAttr("network"); netAttr != nil {
					netAttr.Value = network
				} else {
					source.CreateAttr("network", network)
				}
			} else {
				source := iface.CreateElement("source")
				source.CreateAttr("network", network)
			}
		}
	} else {
		// Handle socket_vmnet case (qemu:commandline)
		var cmdline *etree.Element
		for _, child := range root.ChildElements() {
			if child.Tag == "commandline" && child.Space == "qemu" {
				cmdline = child
				break
			}
		}

		if cmdline != nil {
			for _, child := range cmdline.ChildElements() {
				if child.Tag == "arg" && child.Space == "qemu" {
					if valAttr := child.SelectAttr("value"); valAttr != nil {
						if strings.HasPrefix(valAttr.Value, "virtio-net-pci") && strings.Contains(valAttr.Value, "mac=") {
							parts := strings.Split(valAttr.Value, ",")
							newParts := make([]string, 0, len(parts))
							macUpdated := false
							for _, part := range parts {
								if strings.HasPrefix(part, "mac=") {
									newParts = append(newParts, "mac="+generateMACAddressHelper())
									macUpdated = true
								} else {
									newParts = append(newParts, part)
								}
							}
							if macUpdated {
								valAttr.Value = strings.Join(newParts, ",")
								break
							}
						}
					}
				}
			}
		}
	}

	// Remove existing graphics password
	if graphics := root.FindElement("./devices/graphics"); graphics != nil {
		graphics.RemoveAttr("passwd")
	}

	// Remove existing sound devices
	for _, sound := range root.FindElements("./devices/sound") {
		root.SelectElement("devices").RemoveChild(sound)
	}

	doc.Indent(2)
	newXML, err := doc.WriteToString()
	if err != nil {
		return "", fmt.Errorf("failed to write modified XML: %w", err)
	}

	return newXML, nil
}

// parseDomIfAddrIPv4WithMACHelper parses virsh domifaddr output and returns both IP and MAC address.
func parseDomIfAddrIPv4WithMACHelper(s string) (ip string, mac string) {
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "Name") || strings.HasPrefix(l, "-") {
			continue
		}
		parts := strings.Fields(l)
		if len(parts) >= 4 && parts[2] == "ipv4" {
			mac = parts[1]
			addr := parts[3]
			if i := strings.IndexByte(addr, '/'); i > 0 {
				ip = addr[:i]
			} else {
				ip = addr
			}
			return ip, mac
		}
	}
	return "", ""
}

// parseVMStateHelper converts virsh domstate output to VMState.
// VMState is defined in virsh.go/virsh-stub.go
func parseVMStateHelper(output string) VMState {
	state := strings.TrimSpace(output)
	switch state {
	case "running":
		return VMStateRunning
	case "paused":
		return VMStatePaused
	case "shut off":
		return VMStateShutOff
	case "crashed":
		return VMStateCrashed
	case "pmsuspended":
		return VMStateSuspended
	default:
		return VMStateUnknown
	}
}
