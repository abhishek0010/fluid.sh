package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// HostStatus represents the connectivity status of a sandbox host.
type HostStatus string

const (
	HostStatusOnline  HostStatus = "ONLINE"
	HostStatusOffline HostStatus = "OFFLINE"
)

// SandboxState represents the lifecycle state of a sandbox.
type SandboxState string

const (
	SandboxStateCreating  SandboxState = "CREATING"
	SandboxStateRunning   SandboxState = "RUNNING"
	SandboxStateStopped   SandboxState = "STOPPED"
	SandboxStateDestroyed SandboxState = "DESTROYED"
	SandboxStateError     SandboxState = "ERROR"
)

// StringSlice is a JSON-serialized []string for use as a GORM column type.
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal StringSlice: %w", err)
	}
	return string(b), nil
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for StringSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// SourceVMJSON represents a source VM entry stored as JSON in the host record.
type SourceVMJSON struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IPAddress string `json:"ip_address"`
	Prepared  bool   `json:"prepared"`
}

// SourceVMSlice is a JSON-serialized []SourceVMJSON for use as a GORM column type.
type SourceVMSlice []SourceVMJSON

func (s SourceVMSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal SourceVMSlice: %w", err)
	}
	return string(b), nil
}

func (s *SourceVMSlice) Scan(value interface{}) error {
	if value == nil {
		*s = SourceVMSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for SourceVMSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// BridgeJSON represents a network bridge entry stored as JSON in the host record.
type BridgeJSON struct {
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
}

// BridgeSlice is a JSON-serialized []BridgeJSON for use as a GORM column type.
type BridgeSlice []BridgeJSON

func (s BridgeSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal BridgeSlice: %w", err)
	}
	return string(b), nil
}

func (s *BridgeSlice) Scan(value interface{}) error {
	if value == nil {
		*s = BridgeSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for BridgeSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// Host represents a sandbox host machine registered with the control plane.
type Host struct {
	ID                string        `gorm:"primaryKey;type:text" json:"id"`
	Hostname          string        `gorm:"type:text;not null" json:"hostname"`
	Version           string        `gorm:"type:text" json:"version"`
	TotalCPUs         int32         `gorm:"not null;default:0" json:"total_cpus"`
	TotalMemoryMB     int64         `gorm:"not null;default:0" json:"total_memory_mb"`
	TotalDiskMB       int64         `gorm:"not null;default:0" json:"total_disk_mb"`
	AvailableCPUs     int32         `gorm:"not null;default:0" json:"available_cpus"`
	AvailableMemoryMB int64         `gorm:"not null;default:0" json:"available_memory_mb"`
	AvailableDiskMB   int64         `gorm:"not null;default:0" json:"available_disk_mb"`
	BaseImages        StringSlice   `gorm:"type:jsonb;default:'[]'" json:"base_images"`
	SourceVMs         SourceVMSlice `gorm:"type:jsonb;default:'[]'" json:"source_vms"`
	Bridges           BridgeSlice   `gorm:"type:jsonb;default:'[]'" json:"bridges"`
	Status            HostStatus    `gorm:"type:text;not null;default:'OFFLINE'" json:"status"`
	LastHeartbeat     time.Time     `gorm:"not null" json:"last_heartbeat"`
	CreatedAt         time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Host) TableName() string {
	return "hosts"
}

// Sandbox represents a VM sandbox managed by the control plane.
type Sandbox struct {
	ID         string       `gorm:"primaryKey;type:text" json:"id"`
	HostID     string       `gorm:"type:text;not null;index" json:"host_id"`
	Name       string       `gorm:"type:text;not null" json:"name"`
	AgentID    string       `gorm:"type:text" json:"agent_id"`
	BaseImage  string       `gorm:"type:text" json:"base_image"`
	Bridge     string       `gorm:"type:text" json:"bridge"`
	TAPDevice  string       `gorm:"type:text" json:"tap_device"`
	MACAddress string       `gorm:"type:text" json:"mac_address"`
	IPAddress  string       `gorm:"type:text" json:"ip_address"`
	State      SandboxState `gorm:"type:text;not null;default:'CREATING'" json:"state"`
	VCPUs      int32        `gorm:"not null;default:1" json:"vcpus"`
	MemoryMB   int32        `gorm:"not null;default:512" json:"memory_mb"`
	TTLSeconds int32        `gorm:"not null;default:0" json:"ttl_seconds"`
	SourceVM   string       `gorm:"type:text" json:"source_vm"`
	CreatedAt  time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  *time.Time   `gorm:"index" json:"deleted_at,omitempty"`
}

func (Sandbox) TableName() string {
	return "sandboxes"
}

// Command represents a command executed within a sandbox.
type Command struct {
	ID         string    `gorm:"primaryKey;type:text" json:"id"`
	SandboxID  string    `gorm:"type:text;not null;index" json:"sandbox_id"`
	Command    string    `gorm:"type:text;not null" json:"command"`
	Stdout     string    `gorm:"type:text" json:"stdout"`
	Stderr     string    `gorm:"type:text" json:"stderr"`
	ExitCode   int32     `gorm:"not null;default:0" json:"exit_code"`
	DurationMS int64     `gorm:"not null;default:0" json:"duration_ms"`
	StartedAt  time.Time `gorm:"not null" json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
}

func (Command) TableName() string {
	return "commands"
}
