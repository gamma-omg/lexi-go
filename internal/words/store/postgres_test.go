package store

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type postgresStartRequest struct {
	user     string
	password string
	db       string
}

type postgresStartResponse struct {
	host string
	port string
}

func startPostgres(ctx context.Context, cfg postgresStartRequest) (postgresStartResponse, func()) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:18-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     cfg.user,
			"POSTGRES_PASSWORD": cfg.password,
			"POSTGRES_DB":       cfg.db,
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
	return postgresStartResponse{
		host: host,
		port: port.Port(),
	}, closer
}

func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("failed to get postgres driver: %v", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://../../../db/migrations",
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

func insert(t *testing.T, dp *sql.DB, query string, args ...interface{}) int64 {
	res := dp.QueryRow(query, args...)

	var id int64
	err := res.Scan(&id)
	require.NoError(t, err)
	return id
}

var pgstore *PostresStore

func TestMain(m *testing.M) {
	res, closer := startPostgres(context.Background(), postgresStartRequest{
		user:     "test",
		password: "test",
		db:       "test",
	})
	defer closer()

	var err error
	pgstore, err = NewPostgresStore(PostgresConfig{
		Host:     res.host,
		Port:     res.port,
		User:     "test",
		Password: "test",
		DB:       "test",
	})
	if err != nil {
		log.Fatalf("failed to create postgres store: %v", err)
	}
	defer pgstore.Close()

	os.Exit(m.Run())
}

func TestInsertWord(t *testing.T) {
	runMigrations(t, pgstore.db)

	id, err := pgstore.InsertWord(context.Background(), WordInsertRequest{
		Lemma: "testword",
		Lang:  "en",
		Class: "noun",
	})
	require.NoError(t, err)

	row := pgstore.db.QueryRow("SELECT lemma, lang, class FROM words WHERE id = $1", id)

	var lemma, lang, class string
	err = row.Scan(&lemma, &lang, &class)
	require.NoError(t, err)
	require.Equal(t, "testword", lemma)
	require.Equal(t, "en", lang)
	require.Equal(t, "noun", class)
}

func TestInsertWord_Exists(t *testing.T) {
	runMigrations(t, pgstore.db)

	_, err := pgstore.InsertWord(t.Context(), WordInsertRequest{
		Lemma: "existingword",
		Lang:  "en",
		Class: "noun",
	})
	require.NoError(t, err)

	_, err = pgstore.InsertWord(t.Context(), WordInsertRequest{
		Lemma: "existingword",
		Lang:  "en",
		Class: "noun",
	})
	require.Error(t, err)
	require.Equal(t, ErrExists, err)
}

func TestDeleteWord(t *testing.T) {
	runMigrations(t, pgstore.db)

	id, err := pgstore.InsertWord(t.Context(), WordInsertRequest{
		Lemma: "wordtodelete",
		Lang:  "en",
		Class: "verb",
	})
	require.NoError(t, err)

	err = pgstore.DeleteWord(t.Context(), WordDeleteRequest{
		ID: id,
	})
	require.NoError(t, err)

	row := pgstore.db.QueryRow("SELECT COUNT(1) FROM words WHERE id = $1", id)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestDeleteWord_NotFound(t *testing.T) {
	runMigrations(t, pgstore.db)

	err := pgstore.DeleteWord(t.Context(), WordDeleteRequest{
		ID: 999999,
	})
	require.NoError(t, err)
}

func TestCreateUserPick(t *testing.T) {
	runMigrations(t, pgstore.db)

	var (
		userID = insert(t, pgstore.db, "INSERT INTO users (email) VALUES ($1) RETURNING id", "test@example.com")
		wordID = insert(t, pgstore.db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "pickword", "en", "noun")
		defID  = insert(t, pgstore.db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing picks.")
	)

	err := pgstore.CreateUserPick(t.Context(), UserPickCreateRequest{
		UserID: userID,
		DefID:  defID,
	})
	require.NoError(t, err)

	row := pgstore.db.QueryRow("SELECT COUNT(1) FROM user_picks WHERE user_id = $1 AND def_id = $2", userID, defID)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestCreateUserPick_Exists(t *testing.T) {
	runMigrations(t, pgstore.db)

	var (
		userID = insert(t, pgstore.db, "INSERT INTO users (email) VALUES ($1) RETURNING id", "test@example.com")
		wordID = insert(t, pgstore.db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "existingpickword", "en", "noun")
		defID  = insert(t, pgstore.db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing existing picks.")
	)

	err := pgstore.CreateUserPick(t.Context(), UserPickCreateRequest{
		UserID: userID,
		DefID:  defID,
	})
	require.NoError(t, err)

	err = pgstore.CreateUserPick(t.Context(), UserPickCreateRequest{
		UserID: userID,
		DefID:  defID,
	})
	require.Error(t, err)
	require.Equal(t, ErrExists, err)
}

func TestDeleteUserPick(t *testing.T) {
	runMigrations(t, pgstore.db)

	var (
		userID = insert(t, pgstore.db, "INSERT INTO users (email) VALUES ($1) RETURNING id", "test@example.com")
		wordID = insert(t, pgstore.db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "pickwordtodelete", "en", "noun")
		defID  = insert(t, pgstore.db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing pick deletion.")
		pickID = insert(t, pgstore.db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
	)

	err := pgstore.DeleteUserPick(t.Context(), UserPickDeleteRequest{
		PickID: pickID,
	})
	require.NoError(t, err)

	row := pgstore.db.QueryRow("SELECT COUNT(1) FROM user_picks WHERE id = $1", pickID)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestDeleteUserPick_NotFound(t *testing.T) {
	runMigrations(t, pgstore.db)

	err := pgstore.DeleteUserPick(t.Context(), UserPickDeleteRequest{
		PickID: 999999,
	})
	require.NoError(t, err)
}

func TestGetOrCreateTag(t *testing.T) {
	runMigrations(t, pgstore.db)

	tagName := "testtag"

	tagID1, err := pgstore.GetOrCreateTag(t.Context(), tagName)
	require.NoError(t, err)
	require.NotZero(t, tagID1)

	tagID2, err := pgstore.GetOrCreateTag(t.Context(), tagName)
	require.NoError(t, err)
	require.Equal(t, tagID1, tagID2)

	row := pgstore.db.QueryRow("SELECT COUNT(1) FROM tags WHERE id = $1 AND tag = $2", tagID1, tagName)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestAddTag(t *testing.T) {
	runMigrations(t, pgstore.db)

	var (
		userID = insert(t, pgstore.db, "INSERT INTO users (email) VALUES ($1) RETURNING id", "test@example.com")
		wordID = insert(t, pgstore.db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedword", "en", "noun")
		defID  = insert(t, pgstore.db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tags.")
		pickID = insert(t, pgstore.db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
	)

	tagID, err := pgstore.GetOrCreateTag(t.Context(), "testtag")
	require.NoError(t, err)

	pgstore.AddTag(t.Context(), UserPickAddTagRequest{
		PickID: pickID,
		TagID:  tagID,
	})

	row := pgstore.db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestAddTag_PickNotFound(t *testing.T) {
	runMigrations(t, pgstore.db)

	err := pgstore.AddTag(t.Context(), UserPickAddTagRequest{
		PickID: 888888,
		TagID:  888888,
	})
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestAddTag_TagNotFound(t *testing.T) {
	runMigrations(t, pgstore.db)

	var (
		userID = insert(t, pgstore.db, "INSERT INTO users (email) VALUES ($1) RETURNING id", "test@example.com")
		wordID = insert(t, pgstore.db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedword", "en", "noun")
		defID  = insert(t, pgstore.db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tags.")
		pickID = insert(t, pgstore.db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
	)

	err := pgstore.AddTag(t.Context(), UserPickAddTagRequest{
		PickID: pickID,
		TagID:  999999,
	})
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestRemoveTag(t *testing.T) {
	runMigrations(t, pgstore.db)

	var (
		userID = insert(t, pgstore.db, "INSERT INTO users (email) VALUES ($1) RETURNING id", "test@example.com")
		wordID = insert(t, pgstore.db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordtoremove", "en", "noun")
		defID  = insert(t, pgstore.db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tag removal.")
		pickID = insert(t, pgstore.db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
		tagID  = insert(t, pgstore.db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagtoremove")
		_      = insert(t, pgstore.db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID)
	)
	err := pgstore.RemoveTag(t.Context(), UserPickRemoveTagRequest{
		PickID: pickID,
		TagID:  tagID,
	})
	require.NoError(t, err)

	row := pgstore.db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestRemoveTag_PickNotFound(t *testing.T) {
	runMigrations(t, pgstore.db)

	err := pgstore.RemoveTag(t.Context(), UserPickRemoveTagRequest{
		PickID: 888888,
		TagID:  888888,
	})
	require.NoError(t, err)
}

func TestRemoveTag_TagNotFound(t *testing.T) {
	runMigrations(t, pgstore.db)

	var (
		userID = insert(t, pgstore.db, "INSERT INTO users (email) VALUES ($1) RETURNING id", "test@example.com")
		wordID = insert(t, pgstore.db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedword", "en", "noun")
		defID  = insert(t, pgstore.db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tags.")
		pickID = insert(t, pgstore.db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
	)

	err := pgstore.RemoveTag(t.Context(), UserPickRemoveTagRequest{
		PickID: pickID,
		TagID:  888888,
	})
	require.NoError(t, err)
}
