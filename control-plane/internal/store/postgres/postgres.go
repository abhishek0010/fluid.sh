package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/aspectrr/fluid.sh/control-plane/internal/store"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Store is the PostgreSQL-backed persistent store for the control plane.
type Store struct {
	db *gorm.DB
}

// New creates a new PostgreSQL store. If autoMigrate is true, it runs
// GORM auto-migration for all model tables on startup.
func New(ctx context.Context, databaseURL string, autoMigrate bool) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if autoMigrate {
		if err := db.AutoMigrate(
			&store.Host{},
			&store.Sandbox{},
			&store.Command{},
		); err != nil {
			return nil, fmt.Errorf("auto-migrate: %w", err)
		}
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database connection pool.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// ---------------------------------------------------------------------------
// Host CRUD
// ---------------------------------------------------------------------------

// CreateHost inserts a new host record.
func (s *Store) CreateHost(ctx context.Context, host *store.Host) error {
	return s.db.WithContext(ctx).Create(host).Error
}

// GetHost retrieves a host by its ID.
func (s *Store) GetHost(ctx context.Context, hostID string) (*store.Host, error) {
	var host store.Host
	if err := s.db.WithContext(ctx).Where("id = ?", hostID).First(&host).Error; err != nil {
		return nil, err
	}
	return &host, nil
}

// ListHosts returns all host records.
func (s *Store) ListHosts(ctx context.Context) ([]store.Host, error) {
	var hosts []store.Host
	if err := s.db.WithContext(ctx).Find(&hosts).Error; err != nil {
		return nil, err
	}
	return hosts, nil
}

// UpdateHost saves all fields of the given host record.
func (s *Store) UpdateHost(ctx context.Context, host *store.Host) error {
	return s.db.WithContext(ctx).Save(host).Error
}

// UpdateHostHeartbeat updates the last heartbeat time, available resources,
// and sets the host status to ONLINE.
func (s *Store) UpdateHostHeartbeat(ctx context.Context, hostID string, availCPUs int32, availMemMB int64, availDiskMB int64) error {
	return s.db.WithContext(ctx).
		Model(&store.Host{}).
		Where("id = ?", hostID).
		Updates(map[string]interface{}{
			"available_cpus":      availCPUs,
			"available_memory_mb": availMemMB,
			"available_disk_mb":   availDiskMB,
			"status":              store.HostStatusOnline,
			"last_heartbeat":      time.Now(),
		}).Error
}

// ---------------------------------------------------------------------------
// Sandbox CRUD
// ---------------------------------------------------------------------------

// CreateSandbox inserts a new sandbox record.
func (s *Store) CreateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	return s.db.WithContext(ctx).Create(sandbox).Error
}

// GetSandbox retrieves a sandbox by its ID. Soft-deleted sandboxes are excluded.
func (s *Store) GetSandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	var sandbox store.Sandbox
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", sandboxID).
		First(&sandbox).Error; err != nil {
		return nil, err
	}
	return &sandbox, nil
}

// ListSandboxes returns all non-deleted sandbox records.
func (s *Store) ListSandboxes(ctx context.Context) ([]store.Sandbox, error) {
	var sandboxes []store.Sandbox
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Find(&sandboxes).Error; err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// UpdateSandbox saves all fields of the given sandbox record.
func (s *Store) UpdateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	return s.db.WithContext(ctx).Save(sandbox).Error
}

// DeleteSandbox performs a soft delete by setting deleted_at and state to DESTROYED.
func (s *Store) DeleteSandbox(ctx context.Context, sandboxID string) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&store.Sandbox{}).
		Where("id = ?", sandboxID).
		Updates(map[string]interface{}{
			"deleted_at": &now,
			"state":      store.SandboxStateDestroyed,
		}).Error
}

// GetSandboxesByHostID returns all non-deleted sandboxes for a given host.
func (s *Store) GetSandboxesByHostID(ctx context.Context, hostID string) ([]store.Sandbox, error) {
	var sandboxes []store.Sandbox
	if err := s.db.WithContext(ctx).
		Where("host_id = ? AND deleted_at IS NULL", hostID).
		Find(&sandboxes).Error; err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// ListExpiredSandboxes returns non-deleted sandboxes that have exceeded their TTL.
// Sandboxes with ttl_seconds = 0 use the provided defaultTTL. If defaultTTL is
// also 0, those sandboxes are never considered expired.
func (s *Store) ListExpiredSandboxes(ctx context.Context, defaultTTL time.Duration) ([]store.Sandbox, error) {
	var sandboxes []store.Sandbox
	now := time.Now()

	query := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("state IN ?", []store.SandboxState{store.SandboxStateRunning, store.SandboxStateStopped})

	if defaultTTL > 0 {
		defaultTTLSeconds := int32(defaultTTL.Seconds())
		// Sandbox has its own TTL set, and it has expired.
		// OR sandbox has no TTL (0) but default TTL applies and it has expired.
		query = query.Where(
			"(ttl_seconds > 0 AND created_at + (ttl_seconds || ' seconds')::interval < ?) "+
				"OR (ttl_seconds = 0 AND created_at + (? || ' seconds')::interval < ?)",
			now, defaultTTLSeconds, now,
		)
	} else {
		// No default TTL: only expire sandboxes with an explicit TTL.
		query = query.Where(
			"ttl_seconds > 0 AND created_at + (ttl_seconds || ' seconds')::interval < ?",
			now,
		)
	}

	if err := query.Find(&sandboxes).Error; err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// ---------------------------------------------------------------------------
// Command
// ---------------------------------------------------------------------------

// CreateCommand inserts a new command record.
func (s *Store) CreateCommand(ctx context.Context, cmd *store.Command) error {
	return s.db.WithContext(ctx).Create(cmd).Error
}

// ListSandboxCommands returns all commands for a given sandbox, ordered by started_at.
func (s *Store) ListSandboxCommands(ctx context.Context, sandboxID string) ([]store.Command, error) {
	var commands []store.Command
	if err := s.db.WithContext(ctx).
		Where("sandbox_id = ?", sandboxID).
		Order("started_at ASC").
		Find(&commands).Error; err != nil {
		return nil, err
	}
	return commands, nil
}
