package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/test"
	testdb "github.com/gamma-omg/lexi-go/internal/pkg/test/db"
	"github.com/stretchr/testify/require"
)

var (
	db     *sql.DB
	dbHost string
	dbPort string
)

const (
	dbUser = "test"
	dbPass = "test"
	dbName = "auth_service"

	migrationsFolder = "../db/migrations"
)

func TestMain(m *testing.M) {
	resp, teardown := testdb.StartPostgres(context.Background(), testdb.PostgresStartRequest{
		User:     dbUser,
		Password: dbPass,
		DB:       dbName,
	})
	defer teardown()

	dbHost = resp.Host
	dbPort = resp.Port

	var err error
	db, err = sql.Open("postgres", "host="+dbHost+" port="+dbPort+" user="+dbUser+" password="+dbPass+" dbname="+dbName+" sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func TestRun(t *testing.T) {
	testdb.RunMigrations(t, db, migrationsFolder)

	t.Setenv("JWT_ACCESS_SECRET", "secret")
	t.Setenv("JWT_REFRESH_SECRET", "secret")
	t.Setenv("OAUTH_GOOGLE_CLIENT_ID", "client_id")
	t.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", "client_secret")
	t.Setenv("DB_HOST", dbHost)
	t.Setenv("DB_PORT", dbPort)
	t.Setenv("DB_NAME", dbName)
	t.Setenv("DB_USER", dbUser)
	t.Setenv("DB_PASSWORD", dbPass)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	healthCh := make(chan bool, 1)
	readyCh := make(chan bool, 1)
	go func() {
		errCh <- run(ctx)
	}()

	go func() {
		readyCh <- test.WaitFor(t, ctx, 500*time.Millisecond, func() bool {
			resp, err := http.Get("http://localhost:8080/readyz")
			if err != nil {
				return false
			}

			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		})
	}()

	go func() {
		healthCh <- test.WaitFor(t, ctx, 500*time.Millisecond, func() bool {
			resp, err := http.Get("http://localhost:8080/healthz")
			if err != nil {
				return false
			}

			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		})
	}()

	var isHealthy, isReady bool
	for !isHealthy || !isReady {
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case isHealthy = <-healthCh:
			require.True(t, isHealthy)
		case isReady = <-readyCh:
			require.True(t, isReady)
		case <-ctx.Done():
			t.Fatal("test timed out")
		}
	}
}

func TestRun_Cancel(t *testing.T) {
	testdb.RunMigrations(t, db, migrationsFolder)

	t.Setenv("JWT_ACCESS_SECRET", "secret")
	t.Setenv("JWT_REFRESH_SECRET", "secret")
	t.Setenv("OAUTH_GOOGLE_CLIENT_ID", "client_id")
	t.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", "client_secret")
	t.Setenv("DB_HOST", dbHost)
	t.Setenv("DB_PORT", dbPort)
	t.Setenv("DB_NAME", dbName)
	t.Setenv("DB_USER", dbUser)
	t.Setenv("DB_PASSWORD", dbPass)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
