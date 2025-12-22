package store

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	testdb "github.com/gamma-omg/lexi-go/internal/pkg/test/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	db  *sql.DB
	pgs *PostgresStore
)

const migrationsFolder = "../../db/migrations"

func TestMain(m *testing.M) {
	res, close := testdb.StartPostgres(context.Background(), testdb.PostgresStartRequest{
		User:     "test",
		Password: "test",
		DB:       "test",
	})
	defer close()

	var err error
	db, err = NewPostgresDB(PostgresConfig{
		Host:     res.Host,
		Port:     res.Port,
		User:     "test",
		Password: "test",
		DB:       "test",
	})
	if err != nil {
		log.Fatal("failed to connect to postgres:", err)
	}

	pgs = NewPostgresStore(db)
	os.Exit(m.Run())
}

func TestGetIdentity(t *testing.T) {
	testdb.RunMigrations(t, db, migrationsFolder)
	var (
		userID = testdb.Query(t, db, "INSERT INTO users DEFAULT VALUES RETURNING id").AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO identities (user_id, id, email, name, picture, provider) VALUES ($1, $2, $3, $4, $5, $6)",
			userID,
			"identity_1",
			"test@example.com",
			"Test User",
			"http://example.com/picture.jpg",
			"google")
	)

	id, err := pgs.GetIdentity(t.Context(), GetIdentityRequest{
		ID:       "identity_1",
		Provider: "google",
	})
	require.NoError(t, err)

	assert.Equal(t, "identity_1", id.ID)
	assert.Equal(t, userID, id.User.ID)
	assert.Equal(t, "google", id.Provider)
	assert.Equal(t, "test@example.com", id.Email)
	assert.Equal(t, "Test User", id.Name)
	assert.Equal(t, "http://example.com/picture.jpg", id.Picture)
}

func TestCreateUser(t *testing.T) {
	testdb.RunMigrations(t, db, migrationsFolder)

	userID, err := pgs.CreateUser(t.Context())
	require.NoError(t, err)

	var dbUID string
	err = db.QueryRowContext(t.Context(), "SELECT uid FROM users WHERE id=$1", userID).Scan(&dbUID)
	require.NoError(t, err)
}

func TestCreateUserIdentity(t *testing.T) {
	testdb.RunMigrations(t, db, migrationsFolder)
	userID := testdb.Query(t, db, "INSERT INTO users DEFAULT VALUES RETURNING id").AsInt64()

	req := CreateUserIdentityRequest{
		UserID:   userID,
		ID:       "identity_1",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "http://example.com/picture.jpg",
		Provider: "github",
	}
	id, err := pgs.CreateUserIdentity(t.Context(), req)
	require.NoError(t, err)

	var dbEmail, dbProvider, dbName, dbPicture string
	err = db.QueryRowContext(t.Context(), "SELECT email, provider, name, picture FROM identities WHERE id=$1", id).Scan(
		&dbEmail,
		&dbProvider,
		&dbName,
		&dbPicture,
	)
	require.NoError(t, err)

	assert.Equal(t, req.Email, dbEmail)
	assert.Equal(t, req.Provider, dbProvider)
	assert.Equal(t, req.Name, dbName)
	assert.Equal(t, req.Picture, dbPicture)
}

func TestWithTx(t *testing.T) {
	testdb.RunMigrations(t, db, migrationsFolder)

	err := pgs.WithTx(t.Context(), func(tx Store) error {
		userID, err := tx.CreateUser(t.Context())
		if err != nil {
			return err
		}

		_, err = tx.CreateUserIdentity(t.Context(), CreateUserIdentityRequest{
			UserID:   userID,
			ID:       "identity_tx",
			Provider: "google",
			Email:    "test@example.com",
			Name:     "Test User",
			Picture:  "http://example.com/picture.jpg",
		})
		return err
	})
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM identities WHERE id=$1", "identity_tx").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestWithTxRollback(t *testing.T) {
	testdb.RunMigrations(t, db, migrationsFolder)

	err := pgs.WithTx(t.Context(), func(tx Store) error {
		userID, err := tx.CreateUser(t.Context())
		if err != nil {
			return err
		}

		_, err = tx.CreateUserIdentity(t.Context(), CreateUserIdentityRequest{
			UserID:   userID,
			ID:       "identity_tx_rollback",
			Provider: "google",
			Email:    "test@example.com",
			Name:     "Test User",
			Picture:  "http://example.com/picture.jpg",
		})
		if err != nil {
			return err
		}

		return assert.AnError
	})
	require.Error(t, err)

	var count int
	err = db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM identities WHERE id=$1", "identity_tx_rollback").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
