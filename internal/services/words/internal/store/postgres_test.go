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
		"file://../../db/migrations",
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
	t.Helper()

	res := dp.QueryRow(query, args...)

	var id int64
	err := res.Scan(&id)
	require.NoError(t, err)
	return id
}

var db *sql.DB
var pgstore *PostresStore

func TestMain(m *testing.M) {
	res, closer := startPostgres(context.Background(), postgresStartRequest{
		user:     "test",
		password: "test",
		db:       "test",
	})
	defer closer()

	var err error
	db, err = NewPostgresDB(PostgresConfig{
		Host:     res.host,
		Port:     res.port,
		User:     "test",
		Password: "test",
		DB:       "test",
	})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	pgstore = NewPostgresStore(db)
	os.Exit(m.Run())
}

func TestInsertWord(t *testing.T) {
	runMigrations(t, db)

	id, err := pgstore.InsertWord(context.Background(), InsertWordRequst{
		Lemma: "testword",
		Lang:  "en",
		Class: "noun",
	})
	require.NoError(t, err)

	row := db.QueryRow("SELECT lemma, lang, class FROM words WHERE id = $1", id)

	var lemma, lang, class string
	err = row.Scan(&lemma, &lang, &class)
	require.NoError(t, err)
	require.Equal(t, "testword", lemma)
	require.Equal(t, "en", lang)
	require.Equal(t, "noun", class)
}

func TestInsertWord_Exists(t *testing.T) {
	runMigrations(t, db)

	_, err := pgstore.InsertWord(t.Context(), InsertWordRequst{
		Lemma: "existingword",
		Lang:  "en",
		Class: "noun",
	})
	require.NoError(t, err)

	_, err = pgstore.InsertWord(t.Context(), InsertWordRequst{
		Lemma: "existingword",
		Lang:  "en",
		Class: "noun",
	})
	require.Error(t, err)
	require.Equal(t, ErrExists, err)
}

func TestDeleteWord(t *testing.T) {
	runMigrations(t, db)

	id, err := pgstore.InsertWord(t.Context(), InsertWordRequst{
		Lemma: "wordtodelete",
		Lang:  "en",
		Class: "verb",
	})
	require.NoError(t, err)

	err = pgstore.DeleteWord(t.Context(), DeleteWordRequest{
		ID: id,
	})
	require.NoError(t, err)

	row := db.QueryRow("SELECT COUNT(1) FROM words WHERE id = $1", id)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestDeleteWord_NotFound(t *testing.T) {
	runMigrations(t, db)

	err := pgstore.DeleteWord(t.Context(), DeleteWordRequest{
		ID: 999999,
	})
	require.NoError(t, err)
}

func TestCreateUserPick(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "pickword", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing picks.")
	)

	_, err := pgstore.CreateUserPick(t.Context(), CreateUserPickRequest{
		UserID: userID,
		DefID:  defID,
	})
	require.NoError(t, err)

	row := db.QueryRow("SELECT COUNT(1) FROM user_picks WHERE user_id = $1 AND def_id = $2", userID, defID)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestCreateUserPick_Exists(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "existingpickword", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing existing picks.")
	)

	_, err := pgstore.CreateUserPick(t.Context(), CreateUserPickRequest{
		UserID: userID,
		DefID:  defID,
	})
	require.NoError(t, err)

	_, err = pgstore.CreateUserPick(t.Context(), CreateUserPickRequest{
		UserID: userID,
		DefID:  defID,
	})
	require.Error(t, err)
	require.Equal(t, ErrExists, err)
}

func TestDeleteUserPick(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "pickwordtodelete", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing pick deletion.")
		pickID = insert(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
	)

	err := pgstore.DeleteUserPick(t.Context(), DeleteUserPickRequest{
		PickID: pickID,
	})
	require.NoError(t, err)

	row := db.QueryRow("SELECT COUNT(1) FROM user_picks WHERE id = $1", pickID)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestDeleteUserPick_NotFound(t *testing.T) {
	runMigrations(t, db)

	err := pgstore.DeleteUserPick(t.Context(), DeleteUserPickRequest{
		PickID: 999999,
	})
	require.NoError(t, err)
}

func TestCreateTags(t *testing.T) {
	runMigrations(t, db)

	tags, err := pgstore.CreateTags(t.Context(), CreateTagsRequest{
		Tags: []string{"tag1", "tag2", "tag3"},
	})
	require.NoError(t, err)
	require.Len(t, tags, 3)

	for _, tag := range []string{"tag1", "tag2", "tag3"} {
		tagID, ok := tags[tag]
		require.True(t, ok)

		row := db.QueryRow("SELECT id FROM tags WHERE tag = $1", tag)

		var id int
		err := row.Scan(&id)
		require.NoError(t, err)
		require.Equal(t, int(tagID), id)
	}
}

func TestCreateTags_ExistingTags(t *testing.T) {
	runMigrations(t, db)

	_, err := pgstore.CreateTags(t.Context(), CreateTagsRequest{
		Tags: []string{"tag1", "tag2"},
	})
	require.NoError(t, err)

	tags, err := pgstore.CreateTags(t.Context(), CreateTagsRequest{
		Tags: []string{"tag2", "tag3", "tag4"},
	})
	require.NoError(t, err)
	require.Len(t, tags, 3)

	for _, tag := range []string{"tag2", "tag3", "tag4"} {
		tagID, ok := tags[tag]
		require.True(t, ok)
		row := db.QueryRow("SELECT id FROM tags WHERE tag = $1", tag)

		var id int
		err := row.Scan(&id)
		require.NoError(t, err)
		require.Equal(t, int(tagID), id)
	}
}

func TestCreateTags_EmptyInput(t *testing.T) {
	runMigrations(t, db)

	tags, err := pgstore.CreateTags(t.Context(), CreateTagsRequest{Tags: []string{}})
	require.NoError(t, err)
	require.Len(t, tags, 0)
}

func TestGetTags(t *testing.T) {
	runMigrations(t, db)

	_, err := pgstore.CreateTags(t.Context(), CreateTagsRequest{
		Tags: []string{"tagA", "tagB", "tagC"},
	})
	require.NoError(t, err)

	tags, err := pgstore.GetTags(t.Context(), GetTagsRequest{
		Tags: []string{"tagA", "tagC"},
	})
	require.NoError(t, err)
	require.Len(t, tags, 2)

	for _, tag := range []string{"tagA", "tagC"} {
		tagID, ok := tags[tag]
		require.True(t, ok)

		row := db.QueryRow("SELECT id FROM tags WHERE tag = $1", tag)

		var id int
		err := row.Scan(&id)
		require.NoError(t, err)
		require.Equal(t, int(tagID), id)
	}
}

func TestGetTags_PartialMissing(t *testing.T) {
	runMigrations(t, db)

	_, err := pgstore.CreateTags(t.Context(), CreateTagsRequest{
		Tags: []string{"tagX", "tagY"},
	})
	require.NoError(t, err)

	tags, err := pgstore.GetTags(t.Context(), GetTagsRequest{
		Tags: []string{"tagX", "tagZ"},
	})
	require.NoError(t, err)
	require.Len(t, tags, 1)

	tagID, ok := tags["tagX"]
	require.True(t, ok)

	row := db.QueryRow("SELECT id FROM tags WHERE tag = $1", "tagX")

	var id int
	err = row.Scan(&id)
	require.NoError(t, err)
	require.Equal(t, int(tagID), id)
}

func TestGetTags_AllMissing(t *testing.T) {
	runMigrations(t, db)

	tags, err := pgstore.GetTags(t.Context(), GetTagsRequest{
		Tags: []string{"missingTag1", "missingTag2"},
	})
	require.NoError(t, err)
	require.Len(t, tags, 0)
}

func TestGetTags_EmptyInput(t *testing.T) {
	runMigrations(t, db)

	tags, err := pgstore.GetTags(t.Context(), GetTagsRequest{
		Tags: []string{},
	})
	require.NoError(t, err)
	require.Len(t, tags, 0)
}

func TestAddTags(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordtoadd", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tag addition.")
		pickID = insert(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
		tagID1 = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagToAdd1")
		tagID2 = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagToAdd2")
	)

	err := pgstore.AddTags(t.Context(), AddTagsRequest{
		PickID: pickID,
		TagIDs: []int64{tagID1, tagID2},
	})
	require.NoError(t, err)

	row1 := db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID1)
	row2 := db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID2)

	var count1, count2 int
	err = row1.Scan(&count1)
	require.NoError(t, err)
	err = row2.Scan(&count2)
	require.NoError(t, err)

	require.Equal(t, 1, count1)
	require.Equal(t, 1, count2)
}

func TestAddTags_PickNotFound(t *testing.T) {
	runMigrations(t, db)

	var tagID = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagForNonExistentPick")
	err := pgstore.AddTags(t.Context(), AddTagsRequest{
		PickID: 888888,
		TagIDs: []int64{tagID},
	})
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestAddTags_TagNotFound(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordwithmissingtag", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing missing tags.")
		pickID = insert(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
	)

	err := pgstore.AddTags(t.Context(), AddTagsRequest{
		PickID: pickID,
		TagIDs: []int64{777777},
	})
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestAddTags_ExistingTag(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordwithexistingtag", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing existing tags.")
		pickID = insert(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
		tagID  = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "existingTag")
		_      = insert(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID)
	)

	err := pgstore.AddTags(t.Context(), AddTagsRequest{
		PickID: pickID,
		TagIDs: []int64{tagID},
	})
	require.Error(t, err)
	require.Equal(t, ErrExists, err)
}

func TestRemoveTags_MultipleTags(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "banana", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A yellow fruit")
		pickID = insert(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
		tagID1 = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tag1")
		tagID2 = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tag2")
		tagID3 = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tag3")
		_      = insert(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID1)
		_      = insert(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID2)
		_      = insert(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID3)
	)

	err := pgstore.RemoveTags(t.Context(), RemoveTagsRequest{
		PickID: pickID,
		TagIDs: []int64{tagID1, tagID3},
	})
	require.NoError(t, err)

	row1 := db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID1)
	row2 := db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID2)
	row3 := db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID3)

	var count1, count2, count3 int
	err = row1.Scan(&count1)
	require.NoError(t, err)
	err = row2.Scan(&count2)
	require.NoError(t, err)
	err = row3.Scan(&count3)
	require.NoError(t, err)

	require.Equal(t, 0, count1)
	require.Equal(t, 1, count2)
	require.Equal(t, 0, count3)
}

func TestRemoveTags_SingleTag(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordtoremove", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tag removal.")
		pickID = insert(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
		tagID  = insert(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagtoremove")
		_      = insert(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID)
	)
	err := pgstore.RemoveTags(t.Context(), RemoveTagsRequest{
		PickID: pickID,
		TagIDs: []int64{tagID},
	})
	require.NoError(t, err)

	row := db.QueryRow("SELECT COUNT(1) FROM tags_map WHERE pick_id = $1 AND tag_id = $2", pickID, tagID)

	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestRemoveTag_PickNotFound(t *testing.T) {
	runMigrations(t, db)

	err := pgstore.RemoveTags(t.Context(), RemoveTagsRequest{
		PickID: 888888,
		TagIDs: []int64{888888},
	})
	require.NoError(t, err)
}

func TestRemoveTags_TagNotFound(t *testing.T) {
	runMigrations(t, db)

	var (
		userID = "user-123"
		wordID = insert(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedword", "en", "noun")
		defID  = insert(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tags.")
		pickID = insert(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID)
	)

	err := pgstore.RemoveTags(t.Context(), RemoveTagsRequest{
		PickID: pickID,
		TagIDs: []int64{888888},
	})
	require.NoError(t, err)
}
