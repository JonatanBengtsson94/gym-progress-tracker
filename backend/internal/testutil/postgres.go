// Package testutil provides shared test infrastructure for repository and
// integration tests that need a real Postgres database.
package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartPostgres starts a Postgres testcontainer, applies the migration
// files matched by migrationsGlob in lexical order, then seedFile if it is
// non-empty, and returns a connected pool along with a teardown func that
// closes the pool and terminates the container. Callers are expected to
// call StartPostgres from TestMain and invoke the returned teardown func
// after m.Run().
func StartPostgres(ctx context.Context, migrationsGlob string, seedFile string) (*pgxpool.Pool, func(), error) {
	migrationFiles, err := filepath.Glob(migrationsGlob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to glob migrations: %w", err)
	}
	if len(migrationFiles) == 0 {
		return nil, nil, fmt.Errorf("no migration files found matching %q", migrationsGlob)
	}
	sort.Strings(migrationFiles)

	initScripts := migrationFiles
	if seedFile != "" {
		initScripts = append(initScripts, seedFile)
	}

	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:latest",
		postgres.WithDatabase("testDB"),
		postgres.WithUsername("testUser"),
		postgres.WithPassword("testPass"),
		postgres.WithInitScripts(initScripts...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start postgres testcontainer: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to connect pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	teardown := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return pool, teardown, nil
}
