package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tmux-client/internal/store"
)

// Ensure interface compliance.
var (
	_ store.Store     = (*postgresStore)(nil)
	_ store.DataStore = (*postgresStore)(nil)
)

type postgresStore struct {
	db   *gorm.DB
	conf store.Config
}

// New creates a Store backed by Postgres + GORM.
func New(ctx context.Context, cfg store.Config) (store.Store, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("postgres: missing DatabaseURL")
	}

	db, err := gorm.Open(
		postgres.Open(cfg.DatabaseURL),
		&gorm.Config{
			NowFunc: func() time.Time { return time.Now().UTC() },
			Logger:  logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres: sql.DB handle: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	pg := &postgresStore{
		db:   db.WithContext(ctx),
		conf: cfg,
	}

	if cfg.AutoMigrate && !cfg.ReadOnly {
		if err := pg.autoMigrate(ctx); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	if err := pg.Ping(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return pg, nil
}

// NewWithDB wraps an existing *gorm.DB (useful for tests).
func NewWithDB(db *gorm.DB, cfg store.Config) store.Store {
	return &postgresStore{db: db, conf: cfg}
}

func (s *postgresStore) Config() store.Config {
	return s.conf
}

func (s *postgresStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *postgresStore) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *postgresStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&postgresStore{db: tx, conf: s.conf})
	})
}

// --- Sessions ---

func (s *postgresStore) CreateSession(ctx context.Context, session *store.Session) (*store.Session, error) {
	if s.conf.ReadOnly {
		return nil, fmt.Errorf("postgres: CreateSession: %w", store.ErrInvalid)
	}
	if session == nil || session.ID == "" || session.SandboxID == "" {
		return nil, fmt.Errorf("postgres: CreateSession: %w", store.ErrInvalid)
	}
	if session.StartedAt.IsZero() {
		session.StartedAt = time.Now().UTC()
		session.Live = true
	}

	if err := s.db.WithContext(ctx).Create(sessionToModel(session)).Error; err != nil {
		return nil, mapDBError(err)
	}
	return session, nil
}

func (s *postgresStore) ListSessions(ctx context.Context, sandboxID string) ([]*store.Session, error) {
	var models []SessionModel
	if err := s.db.WithContext(ctx).Where("sandbox_id = ?", sandboxID).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	sessions := make([]*store.Session, len(models))
	for i, model := range models {
		sessions[i] = sessionFromModel(&model)
	}
	return sessions, nil
}

func (s *postgresStore) ListLiveSessions(ctx context.Context, sandboxID string) ([]*store.Session, error) {
	var models []SessionModel
	if err := s.db.WithContext(ctx).Where("sandbox_id = ? AND live = ?", sandboxID, true).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	sessions := make([]*store.Session, len(models))
	for i, model := range models {
		sessions[i] = sessionFromModel(&model)
	}
	return sessions, nil
}

func (s *postgresStore) ReleaseSession(ctx context.Context, sessionID string) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("postgres: ReleaseSession: %w", store.ErrInvalid)
	}
	if sessionID == "" {
		return fmt.Errorf("postgres: ReleaseSession: %w", store.ErrInvalid)
	}

	var model SessionModel
	if err := s.db.WithContext(ctx).Where("id = ?", sessionID).First(&model).Error; err != nil {
		return mapDBError(err)
	}

	model.EndedAt = time.Now().UTC()
	model.Live = false
	if err := s.db.WithContext(ctx).Where("id = ?", sessionID).Updates(&model).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

// --- Command ---

func (s *postgresStore) SaveCommand(ctx context.Context, cmd *store.Command) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("postgres: SaveCommand: %w", store.ErrInvalid)
	}
	if cmd == nil || cmd.ID == "" || cmd.SandboxID == "" || cmd.Command == "" {
		return fmt.Errorf("postgres: SaveCommand: %w", store.ErrInvalid)
	}
	if cmd.StartedAt.IsZero() {
		cmd.StartedAt = time.Now().UTC()
	}
	if cmd.EndedAt.IsZero() {
		cmd.EndedAt = time.Now().UTC()
	}

	if err := s.db.WithContext(ctx).Create(commandToModel(cmd)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetCommand(ctx context.Context, id string) (*store.Command, error) {
	var model CommandModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return commandFromModel(&model), nil
}

func (s *postgresStore) ListCommands(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error) {
	tx := s.db.WithContext(ctx).Model(&CommandModel{}).Where("sandbox_id = ?", sandboxID)
	tx = applyListOptions(tx, opt, map[string]string{
		"started_at": "started_at",
		"ended_at":   "ended_at",
	})

	var models []CommandModel
	if err := tx.Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.Command, 0, len(models))
	for i := range models {
		out = append(out, commandFromModel(&models[i]))
	}
	return out, nil
}

// --- Migration ---

func (s *postgresStore) autoMigrate(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(
		&CommandModel{},
	)
}

// --- Models & Converters ---

type CommandModel struct {
	ID        string    `gorm:"primaryKey;column:id"`
	SandboxID string    `gorm:"column:sandbox_id;not null;index"`
	Command   string    `gorm:"column:command;not null"`
	EnvJSON   *string   `gorm:"column:env_json;type:jsonb"`
	Stdout    string    `gorm:"column:stdout;not null"`
	Stderr    string    `gorm:"column:stderr;not null"`
	ExitCode  int       `gorm:"column:exit_code;not null"`
	StartedAt time.Time `gorm:"column:started_at;not null;index"`
	EndedAt   time.Time `gorm:"column:ended_at;not null"`
}

func (CommandModel) TableName() string { return "commands" }

func commandToModel(cmd *store.Command) *CommandModel {
	return &CommandModel{
		ID:        cmd.ID,
		SandboxID: cmd.SandboxID,
		Command:   cmd.Command,
		EnvJSON:   copyString(cmd.EnvJSON),
		Stdout:    cmd.Stdout,
		Stderr:    cmd.Stderr,
		ExitCode:  cmd.ExitCode,
		StartedAt: cmd.StartedAt,
		EndedAt:   cmd.EndedAt,
	}
}

func commandFromModel(m *CommandModel) *store.Command {
	return &store.Command{
		ID:        m.ID,
		SandboxID: m.SandboxID,
		Command:   m.Command,
		EnvJSON:   copyString(m.EnvJSON),
		Stdout:    m.Stdout,
		Stderr:    m.Stderr,
		ExitCode:  m.ExitCode,
		StartedAt: m.StartedAt,
		EndedAt:   m.EndedAt,
	}
}

type SessionModel struct {
	ID        string         `gorm:"primaryKey;column:id"`
	SandboxID string         `gorm:"column:sandbox_id;not null;index"`
	Timeout   *time.Duration `gorm:"column:timeout;not null"`
	Live      bool           `gorm:"column:live;not null"`
	StartedAt time.Time      `gorm:"column:started_at;not null;index"`
	EndedAt   time.Time      `gorm:"column:ended_at"`
}

func (SessionModel) TableName() string { return "sessions" }

func sessionToModel(s *store.Session) *SessionModel {
	return &SessionModel{
		ID:        s.ID,
		SandboxID: s.SandboxID,
		Live:      s.Live,
		StartedAt: s.StartedAt,
		EndedAt:   s.EndedAt,
	}
}

func sessionFromModel(m *SessionModel) *store.Session {
	return &store.Session{
		ID:        m.ID,
		SandboxID: m.SandboxID,
		Live:      m.Live,
		StartedAt: m.StartedAt,
		EndedAt:   m.EndedAt,
	}
}

// --- Helpers ---

func applyListOptions(tx *gorm.DB, opt *store.ListOptions, whitelist map[string]string) *gorm.DB {
	orderApplied := false
	if opt != nil {
		if col, ok := whitelist[opt.OrderBy]; ok {
			dir := "DESC"
			if opt.Asc {
				dir = "ASC"
			}
			tx = tx.Order(fmt.Sprintf("%s %s", col, dir))
			orderApplied = true
		}
		if opt.Limit > 0 {
			tx = tx.Limit(opt.Limit)
			if opt.Offset > 0 {
				tx = tx.Offset(opt.Offset)
			}
		}
	}
	if !orderApplied {
		tx = tx.Order("created_at DESC")
	}
	return tx
}

func copyString(src *string) *string {
	if src == nil {
		return nil
	}
	val := *src
	return &val
}

func copyInt(src *int) *int {
	if src == nil {
		return nil
	}
	val := *src
	return &val
}

func copyTime(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	val := *src
	return &val
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return store.ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return store.ErrAlreadyExists
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return store.ErrAlreadyExists
		case "23503":
			return store.ErrInvalid
		}
	}
	return err
}
