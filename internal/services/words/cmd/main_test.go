package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type dbConfig struct {
	user     string
	password string
	dbName   string
}

type dbInstance struct {
	host string
	port string
}

func setupDatabase(t *testing.T, cfg dbConfig) (dbInstance, func()) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image: "postgres:15-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     cfg.user,
			"POSTGRES_PASSWORD": cfg.password,
			"POSTGRES_DB":       cfg.dbName,
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}

	cont, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := cont.Host(context.Background())
	require.NoError(t, err)

	port, err := cont.MappedPort(context.Background(), "5432/tcp")
	require.NoError(t, err)

	db := dbInstance{
		host: host,
		port: port.Port(),
	}
	return db, func() {
		_ = cont.Terminate(context.Background())
	}
}

func TestRun(t *testing.T) {
	dbCfg := dbConfig{
		user:     "testuser",
		password: "testpass",
		dbName:   "testdb",
	}
	db, teardown := setupDatabase(t, dbCfg)
	defer teardown()

	jwtSecret := "test-secret"
	t.Setenv("AUTH_SECRET", jwtSecret)
	t.Setenv("DB_HOST", db.host)
	t.Setenv("DB_PORT", db.port)
	t.Setenv("DB_USER", dbCfg.user)
	t.Setenv("DB_PASSWORD", dbCfg.password)
	t.Setenv("DB_NAME", dbCfg.dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	ready := make(chan bool, 1)
	healthy := make(chan bool, 1)
	go func() {
		errCh <- run(ctx)
	}()

	go func() {
		ready <- waitFor(ctx, 500*time.Millisecond, func() bool {
			resp, err := http.Get("http://localhost:8080/readyz")
			if err != nil {
				return false
			}

			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		})
	}()

	go func() {
		healthy <- waitFor(ctx, 500*time.Millisecond, func() bool {
			resp, err := http.Get("http://localhost:8080/healthz")
			if err != nil {
				return false
			}

			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		})
	}()

	isReady := false
	isHealthy := false
	for !isReady || !isHealthy {
		select {
		case err := <-errCh:
			require.NoError(t, err)
			return
		case isReady = <-ready:
			require.True(t, isReady)
		case isHealthy = <-healthy:
			require.True(t, isHealthy)
		case <-ctx.Done():
			t.Fatal("test timed out")
		}
	}
}

func TestRun_Cancel(t *testing.T) {
	dbCfg := dbConfig{
		user:     "testuser",
		password: "testpass",
		dbName:   "testdb",
	}
	db, teardown := setupDatabase(t, dbCfg)
	defer teardown()

	jwtSecret := "test-secret"
	t.Setenv("AUTH_SECRET", jwtSecret)
	t.Setenv("DB_HOST", db.host)
	t.Setenv("DB_PORT", db.port)
	t.Setenv("DB_USER", dbCfg.user)
	t.Setenv("DB_PASSWORD", dbCfg.password)
	t.Setenv("DB_NAME", dbCfg.dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx)
	}()

	time.Sleep(2 * time.Second)
	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("service did not shut down in time after context cancellation")
	}
}

func waitFor(ctx context.Context, upd time.Duration, check func() bool) bool {
	ticker := time.NewTicker(upd)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if check() {
				return true
			}
		}
	}
}
