#!/usr/bin/env bash
set -euo pipefail

swag init --dir .,./internal/api,./internal/audit,./internal/config,./internal/store,./internal/tools,./internal/types --generalInfo ./cmd/server/main.go --parseDependency --parseInternal
