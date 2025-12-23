package store

import (
	"context"
	"errors"
	"time"
)

// Domain model and persistence contracts for the VM sandbox system.
// This package declares the data structures persisted in the DB and the
// storage interfaces that concrete implementations (SQLite/Postgres) must provide.

// Config describes database-related configuration for a Store implementation.
type Config struct {

	// DatabaseURL is the DSN/URL used to connect to the database.
	// Examples:
	// - Postgres: postgres://user:pass@host:5432/dbname?sslmode=disable
	DatabaseURL string `json:"database_url"`

	// MaxOpenConns sets the maximum number of open connections to the database.
	MaxOpenConns int `json:"max_open_conns"`

	// MaxIdleConns sets the maximum number of connections in the idle connection pool.
	MaxIdleConns int `json:"max_idle_conns"`

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// AutoMigrate, when true, allows the store to create/update schema automatically.
	AutoMigrate bool `json:"auto_migrate"`

	// ReadOnly, when true, disallows mutating operations.
	ReadOnly bool `json:"read_only"`
}

// ListOptions supports pagination and ordering for list operations.
type ListOptions struct {
	Limit   int    // Max records to return (0 = default/backend-defined)
	Offset  int    // Records to skip
	OrderBy string // Column to order by (implementation should whitelist)
	Asc     bool   // Ascending if true, descending if false
}

// Common sentinel errors for store implementations.
var (
	ErrNotFound      = errors.New("store: not found")
	ErrAlreadyExists = errors.New("store: already exists")
	ErrConflict      = errors.New("store: conflict")
	ErrInvalid       = errors.New("store: invalid data")
)

type Session struct {
	ID        string         `json:"id" db:"id"`
	Timeout   *time.Duration `json:"timeout" db:"timeout"`
	SandboxID string         `json:"sandbox_id" db:"sandbox_id"`
	Live      bool           `json:"active" db:"live"`
	StartedAt time.Time      `json:"started_at" db:"started_at"`
	EndedAt   time.Time      `json:"ended_at" db:"ended_at"`
}

type TmuxPane struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

type TmuxWindow struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

type File struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

type HumanAsk struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

type HumanAskResponse struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

type HumanAskAsync struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

type HumanAskAsyncResponse struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

type Plan struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
	Items     []PlanItem        `json:"items" db:"items"`
}

type PlanItem struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	SessionID string            `json:"session_id" db:"session_id"`
	WorkDir   string            `json:"work_dir" db:"work_dir"`
	Timeout   *time.Duration    `json:"timeout" db:"timeout"`
	Redacted  map[string]string `json:"redacted" db:"redacted"`
}

// Command captures an executed command inside a sandbox.
type Command struct {
	ID        string             `json:"id" db:"id"`
	SandboxID string             `json:"sandbox_id" db:"sandbox_id"`
	Command   string             `json:"command" db:"command"`
	EnvJSON   *string            `json:"env_json,omitempty" db:"env_json"` // JSON-encoded env map
	Stdout    string             `json:"stdout" db:"stdout"`
	Stderr    string             `json:"stderr" db:"stderr"`
	ExitCode  int                `json:"exit_code" db:"exit_code"`
	StartedAt time.Time          `json:"started_at" db:"started_at"`
	EndedAt   time.Time          `json:"ended_at" db:"ended_at"`
	Metadata  *CommandExecRecord `json:"metadata,omitempty" db:"-"`
}

// CommandExecRecord is a non-persisted helper payload commonly serialized into Metadata fields.
// It can be persisted by serializing to JSON and storing in an auxiliary column if desired.
type CommandExecRecord struct {
	User     string            `json:"user,omitempty"`
	WorkDir  string            `json:"work_dir,omitempty"`
	Timeout  *time.Duration    `json:"timeout,omitempty"`
	Redacted map[string]string `json:"redacted,omitempty"` // placeholders for secrets redaction
}

// PackageInfo captures package name and version.
type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServiceChange represents a system service change.
type ServiceChange struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled,omitempty"`
	State   string `json:"state,omitempty"` // started|stopped|restarted|reloaded
}

// CommandSummary summarizes executed commands affecting the diff.
type CommandSummary struct {
	Cmd      string    `json:"cmd"`
	ExitCode int       `json:"exit_code"`
	At       time.Time `json:"at"`
}

// DataStore declares data operations. This is transaction-friendly and
// can be implemented by both the root Store and a transactional context.
type DataStore interface {
	// Command
	SaveCommand(ctx context.Context, cmd *Command) error
	GetCommand(ctx context.Context, id string) (*Command, error)
	ListCommands(ctx context.Context, sandboxID string, opt *ListOptions) ([]*Command, error)
}

// Store is the root database handle. It can produce transactional views and
// exposes liveness and lifecycle methods in addition to the DataStore.
type Store interface {
	DataStore

	// Config returns the configuration the store was created with.
	Config() Config

	// Ping verifies DB connectivity/health.
	Ping(ctx context.Context) error

	// WithTx runs fn in a transaction. The provided DataStore must be used for
	// all DB calls within fn and is committed if fn returns nil, rolled back otherwise.
	WithTx(ctx context.Context, fn func(tx DataStore) error) error

	// Close releases resources held by the Store.
	Close() error
}
