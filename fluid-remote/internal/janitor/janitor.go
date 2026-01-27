// Package janitor provides background cleanup of expired sandboxes.
package janitor

import (
	"context"
	"log/slog"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-remote/internal/store"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/vm"
)

// Janitor is a background service that periodically cleans up expired sandboxes.
type Janitor struct {
	store      store.Store
	vmService  *vm.Service
	logger     *slog.Logger
	defaultTTL time.Duration
}

// New creates a new Janitor service.
func New(st store.Store, svc *vm.Service, defaultTTL time.Duration, logger *slog.Logger) *Janitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Janitor{
		store:      st,
		vmService:  svc,
		logger:     logger.With("component", "janitor"),
		defaultTTL: defaultTTL,
	}
}

// Start runs the cleanup loop. It blocks until the context is cancelled.
func (j *Janitor) Start(ctx context.Context, interval time.Duration) {
	j.logger.Info("starting janitor",
		"interval", interval,
		"default_ttl", j.defaultTTL,
	)

	// Run once immediately at startup
	j.cleanup(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.Info("janitor stopped")
			return
		case <-ticker.C:
			j.cleanup(ctx)
		}
	}
}

// cleanup finds and destroys all expired sandboxes.
func (j *Janitor) cleanup(ctx context.Context) {
	expired, err := j.store.ListExpiredSandboxes(ctx, j.defaultTTL)
	if err != nil {
		j.logger.Error("failed to list expired sandboxes", "error", err)
		return
	}

	if len(expired) == 0 {
		return
	}

	j.logger.Info("found expired sandboxes", "count", len(expired))

	for _, sb := range expired {
		j.logger.Info("destroying expired sandbox",
			"id", sb.ID,
			"name", sb.SandboxName,
			"ttl_seconds", sb.TTLSeconds,
			"created_at", sb.CreatedAt,
			"age", time.Since(sb.CreatedAt),
		)

		if _, err := j.vmService.DestroySandbox(ctx, sb.ID); err != nil {
			j.logger.Error("failed to destroy expired sandbox",
				"id", sb.ID,
				"name", sb.SandboxName,
				"error", err,
			)
			// Continue trying to destroy others even if one fails
		} else {
			j.logger.Info("destroyed expired sandbox",
				"id", sb.ID,
				"name", sb.SandboxName,
			)
		}
	}
}
