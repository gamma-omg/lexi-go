package store

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	testdb "github.com/gamma-omg/lexi-go/internal/pkg/test/db"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db *sql.DB
var pgstore *PostresStore

func TestMain(m *testing.M) {
	res, closer := testdb.StartPostgres(context.Background(), testdb.PostgresStartRequest{
		User:     "test",
		Password: "test",
		DB:       "test",
	})
	defer closer()

	var err error
	db, err = NewPostgresDB(PostgresConfig{
		Host:     res.Host,
		Port:     res.Port,
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
	testdb.RunMigrations(t, db)

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
	testdb.RunMigrations(t, db)

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
	testdb.RunMigrations(t, db)

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
	testdb.RunMigrations(t, db)

	err := pgstore.DeleteWord(t.Context(), DeleteWordRequest{
		ID: 999999,
	})
	require.NoError(t, err)
}

func TestCreateUserPick(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "pickword", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing picks.").AsInt64()
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
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "existingpickword", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing existing picks.").AsInt64()
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
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "pickwordtodelete", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing pick deletion.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
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

func TestGetUserPicks(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "banana", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A yellow fruit.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
	)

	response, err := pgstore.GetUserPicks(t.Context(), GetUserPicksRequest{
		UserID:   userID,
		PageSize: 100,
	})
	require.NoError(t, err)
	require.Len(t, response.Picks, 1)

	assert.Equal(t, pickID, response.Picks[0].ID)
	assert.Equal(t, userID, response.Picks[0].UserID)
	assert.Equal(t, "banana", response.Picks[0].Word.Lemma)
	assert.Equal(t, wordID, response.Picks[0].Word.ID)
	assert.Equal(t, model.Lang("en"), response.Picks[0].Word.Lang)
	assert.Equal(t, model.WordClass("noun"), response.Picks[0].Word.Class)
	assert.Equal(t, defID, response.Picks[0].Definition.ID)
	assert.Equal(t, "A yellow fruit.", response.Picks[0].Definition.Text)
	assert.Empty(t, response.Picks[0].Tags)
}

func TestGetUserPicks_WithTags(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "banana", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A yellow fruit.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
		tagID  = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "testtag").AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "apple", "en", "noun").AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A common fruit.").AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID).AsInt64()
	)

	response, err := pgstore.GetUserPicks(t.Context(), GetUserPicksRequest{
		UserID:   userID,
		WithTags: []int64{tagID},
		PageSize: 100,
	})
	require.NoError(t, err)
	require.Len(t, response.Picks, 1)
	require.Equal(t, pickID, response.Picks[0].ID)

	assert.Equal(t, userID, response.Picks[0].UserID)
	assert.Equal(t, "banana", response.Picks[0].Word.Lemma)
	assert.Equal(t, wordID, response.Picks[0].Word.ID)
	assert.Equal(t, model.Lang("en"), response.Picks[0].Word.Lang)
	assert.Equal(t, model.WordClass("noun"), response.Picks[0].Word.Class)
	assert.Equal(t, defID, response.Picks[0].Definition.ID)
	assert.Equal(t, "A yellow fruit.", response.Picks[0].Definition.Text)

	require.Len(t, response.Picks[0].Tags, 1)
	assert.Equal(t, tagID, response.Picks[0].Tags[0].ID)
	assert.Equal(t, "testtag", response.Picks[0].Tags[0].Text)
}

func TestGetUserPicks_WithoutTags(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID  = "user-123"
		wordID  = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "nail", "en", "noun").AsInt64()
		defID1  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A thin metal fastener").AsInt64()
		defID2  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "The hard part at the tip of a finger or toe").AsInt64()
		pickID1 = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID1).AsInt64()
		tagID   = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "unwantedtag").AsInt64()
		pickID2 = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID2).AsInt64()
		_       = testdb.Query(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID1, tagID).AsInt64()
	)

	response, err := pgstore.GetUserPicks(t.Context(), GetUserPicksRequest{
		UserID:      userID,
		WithoutTags: []int64{tagID},
		PageSize:    100,
	})
	require.NoError(t, err)
	require.Len(t, response.Picks, 1)
	require.Equal(t, pickID2, response.Picks[0].ID)

	assert.Equal(t, userID, response.Picks[0].UserID)
	assert.Equal(t, "nail", response.Picks[0].Word.Lemma)
	assert.Equal(t, wordID, response.Picks[0].Word.ID)
	assert.Equal(t, model.Lang("en"), response.Picks[0].Word.Lang)
	assert.Equal(t, model.WordClass("noun"), response.Picks[0].Word.Class)
	assert.Equal(t, defID2, response.Picks[0].Definition.ID)
	assert.Equal(t, "The hard part at the tip of a finger or toe", response.Picks[0].Definition.Text)
	assert.Len(t, response.Picks[0].Tags, 0)
}

func TestGetUserPicks_Pagination(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "wordforpagination", "en", "noun").AsInt64()
	)

	for i := 0; i < 5; i++ {
		defID := testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "Definition "+string(rune('A'+i))).AsInt64()
		_, err := pgstore.CreateUserPick(t.Context(), CreateUserPickRequest{
			UserID: userID,
			DefID:  defID,
		})
		require.NoError(t, err)
	}

	response, err := pgstore.GetUserPicks(t.Context(), GetUserPicksRequest{
		UserID:   userID,
		PageSize: 2,
	})
	require.NoError(t, err)
	require.Len(t, response.Picks, 2)

	response, err = pgstore.GetUserPicks(t.Context(), GetUserPicksRequest{
		UserID:   userID,
		Cursor:   *response.NextCursor,
		PageSize: 2,
	})
	require.NoError(t, err)
	require.Len(t, response.Picks, 2)

	response, err = pgstore.GetUserPicks(t.Context(), GetUserPicksRequest{
		UserID:   userID,
		Cursor:   *response.NextCursor,
		PageSize: 2,
	})
	require.NoError(t, err)
	require.Len(t, response.Picks, 1)
}

func TestDeleteUserPick_NotFound(t *testing.T) {
	testdb.RunMigrations(t, db)

	err := pgstore.DeleteUserPick(t.Context(), DeleteUserPickRequest{
		PickID: 999999,
	})
	require.NoError(t, err)
}

func TestCreateTags(t *testing.T) {
	testdb.RunMigrations(t, db)

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
	testdb.RunMigrations(t, db)

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
	testdb.RunMigrations(t, db)

	tags, err := pgstore.CreateTags(t.Context(), CreateTagsRequest{Tags: []string{}})
	require.NoError(t, err)
	require.Len(t, tags, 0)
}

func TestGetTags(t *testing.T) {
	testdb.RunMigrations(t, db)

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
	testdb.RunMigrations(t, db)

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
	testdb.RunMigrations(t, db)

	tags, err := pgstore.GetTags(t.Context(), GetTagsRequest{
		Tags: []string{"missingTag1", "missingTag2"},
	})
	require.NoError(t, err)
	require.Len(t, tags, 0)
}

func TestGetTags_EmptyInput(t *testing.T) {
	testdb.RunMigrations(t, db)

	tags, err := pgstore.GetTags(t.Context(), GetTagsRequest{
		Tags: []string{},
	})
	require.NoError(t, err)
	require.Len(t, tags, 0)
}

func TestAddTags(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordtoadd", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tag addition.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
		tagID1 = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagToAdd1").AsInt64()
		tagID2 = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagToAdd2").AsInt64()
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
	testdb.RunMigrations(t, db)

	var tagID = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagForNonExistentPick").AsInt64()
	err := pgstore.AddTags(t.Context(), AddTagsRequest{
		PickID: 888888,
		TagIDs: []int64{tagID},
	})
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestAddTags_TagNotFound(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordwithmissingtag", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing missing tags.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
	)

	err := pgstore.AddTags(t.Context(), AddTagsRequest{
		PickID: pickID,
		TagIDs: []int64{777777},
	})
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestAddTags_ExistingTag(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordwithexistingtag", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing existing tags.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
		tagID  = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "existingTag").AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID).AsInt64()
	)

	err := pgstore.AddTags(t.Context(), AddTagsRequest{
		PickID: pickID,
		TagIDs: []int64{tagID},
	})
	require.Error(t, err)
	require.Equal(t, ErrExists, err)
}

func TestRemoveTags_MultipleTags(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "banana", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A yellow fruit").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
		tagID1 = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tag1").AsInt64()
		tagID2 = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tag2").AsInt64()
		tagID3 = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tag3").AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID1).AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID2).AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID3).AsInt64()
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
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedwordtoremove", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tag removal.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
		tagID  = testdb.Query(t, db, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", "tagtoremove").AsInt64()
		_      = testdb.Query(t, db, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2) RETURNING id", pickID, tagID).AsInt64()
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
	testdb.RunMigrations(t, db)

	err := pgstore.RemoveTags(t.Context(), RemoveTagsRequest{
		PickID: 888888,
		TagIDs: []int64{888888},
	})
	require.NoError(t, err)
}

func TestRemoveTags_TagNotFound(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		userID = "user-123"
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "taggedword", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word used for testing tags.").AsInt64()
		pickID = testdb.Query(t, db, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", userID, defID).AsInt64()
	)

	err := pgstore.RemoveTags(t.Context(), RemoveTagsRequest{
		PickID: pickID,
		TagIDs: []int64{888888},
	})
	require.NoError(t, err)
}

func TestCreateDefinition(t *testing.T) {
	testdb.RunMigrations(t, db)

	wordID := testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "defword", "en", "noun").AsInt64()

	defID, err := pgstore.CreateDefinition(t.Context(), CreateDefinitionRequest{
		WordID: wordID,
		Text:   "A definition for testing.",
		Source: model.SrcAI,
	})
	require.NoError(t, err)

	row := db.QueryRow("SELECT word_id, def FROM definitions WHERE id = $1", defID)

	var dbWordID int64
	var defText string
	err = row.Scan(&dbWordID, &defText)
	require.NoError(t, err)
	assert.Equal(t, wordID, dbWordID)
	assert.Equal(t, "A definition for testing.", defText)
}

func TestCreateDefinition_Exists(t *testing.T) {
	testdb.RunMigrations(t, db)

	wordID := testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "existingdefword", "en", "noun").AsInt64()

	_, err := pgstore.CreateDefinition(t.Context(), CreateDefinitionRequest{
		WordID: wordID,
		Text:   "An existing definition.",
		Source: model.SrcUser,
	})
	require.NoError(t, err)

	_, err = pgstore.CreateDefinition(t.Context(), CreateDefinitionRequest{
		WordID: wordID,
		Text:   "An existing definition.",
		Source: model.SrcAI,
	})
	require.Error(t, err)
	assert.Equal(t, ErrExists, err)
}

func TestCreateDefinition_WordNotFound(t *testing.T) {
	testdb.RunMigrations(t, db)

	_, err := pgstore.CreateDefinition(t.Context(), CreateDefinitionRequest{
		WordID: 999999,
		Text:   "Definition for non-existent word.",
		Source: model.SrcUnknown,
	})
	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestAttachImage(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "imageword", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word for testing image attachment.").AsInt64()
	)

	imgID, err := pgstore.AttachImage(t.Context(), AttachImageRequest{
		DefID:    defID,
		ImageURL: "http://example.com/image.jpg",
		Source:   model.SrcUser,
	})
	require.NoError(t, err)

	row := db.QueryRow("SELECT url FROM images WHERE id = $1", imgID)

	var imageURL string
	err = row.Scan(&imageURL)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/image.jpg", imageURL)
}

func TestAttachImage_DefinitionNotFound(t *testing.T) {
	testdb.RunMigrations(t, db)

	_, err := pgstore.AttachImage(t.Context(), AttachImageRequest{
		DefID:    999999,
		ImageURL: "http://example.com/nonexistent.jpg",
		Source:   model.SrcAI,
	})
	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestAttachImage_Exists(t *testing.T) {
	testdb.RunMigrations(t, db)

	var (
		wordID = testdb.Query(t, db, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", "existingimageword", "en", "noun").AsInt64()
		defID  = testdb.Query(t, db, "INSERT INTO definitions (word_id, def) VALUES ($1, $2) RETURNING id", wordID, "A word with existing image.").AsInt64()
	)

	_, err := pgstore.AttachImage(t.Context(), AttachImageRequest{
		DefID:    defID,
		ImageURL: "http://example.com/newimage.jpg",
		Source:   model.SrcUnknown,
	})
	require.NoError(t, err)

	_, err = pgstore.AttachImage(t.Context(), AttachImageRequest{
		DefID:    defID,
		ImageURL: "http://example.com/newimage.jpg",
		Source:   model.SrcUser,
	})
	require.Error(t, err)
	assert.Equal(t, ErrExists, err)
}
