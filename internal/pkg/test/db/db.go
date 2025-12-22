package testdb

import (
	"context"
	"database/sql"
	"log"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresStartRequest struct {
	User     string
	Password string
	DB       string
}

type PostgresStartResponse struct {
	Host string
	Port string
}

func StartPostgres(ctx context.Context, cfg PostgresStartRequest) (PostgresStartResponse, func()) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:18-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     cfg.User,
			"POSTGRES_PASSWORD": cfg.Password,
			"POSTGRES_DB":       cfg.DB,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := cont.Host(ctx)
	if err != nil {
		log.Fatalf("failed to get host: %v", err)
	}

	port, err := cont.MappedPort(ctx, "5432/tcp")
	if err != nil {
		log.Fatalf("failed to get port: %v", err)
	}

	closer := func() {
		_ = cont.Terminate(ctx)
	}
	return PostgresStartResponse{
		Host: host,
		Port: port.Port(),
	}, closer
}

func RunMigrations(t *testing.T, db *sql.DB, folder string) {
	t.Helper()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("failed to get postgres driver: %v", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://"+folder,
		"test", driver)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	if err := migrator.Down(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to drop existing db objects: %v", err)
	}

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to run migrations: %v", err)
	}
}

type dbQuery struct {
	t   *testing.T
	row *sql.Row
}

func Query(t *testing.T, db *sql.DB, query string, args ...interface{}) *dbQuery {
	t.Helper()

	row := db.QueryRow(query, args...)
	require.NoError(t, row.Err())

	return &dbQuery{
		t:   t,
		row: row,
	}
}

func (q *dbQuery) AsInt64() int64 {
	q.t.Helper()

	var id int64
	err := q.row.Scan(&id)
	require.NoError(q.t, err)
	return id
}

func (q *dbQuery) AsString() string {
	q.t.Helper()

	var id string
	err := q.row.Scan(&id)
	require.NoError(q.t, err)
	return id
}
