package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/fn"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
)

// WordsService provides access to the global word list and related operations
type WordsService struct {
	store store.DataStore
	tags  *tagManager
}

type WordsServiceConfig struct {
	TagsCacheSize int64
	TagsMaxCost   int64
}

func NewWordsService(store store.DataStore, cfg WordsServiceConfig) *WordsService {
	return &WordsService{
		store: store,
		tags:  newTagManager(cfg.TagsCacheSize, cfg.TagsMaxCost),
	}
}

type AddWordRequest struct {
	Lemma string
	Lang  model.Lang
	Class model.WordClass
}

// AddWord adds a new word to the global word list. If the word already exists,
// it returns a ServiceError with status code 409. The word is uniquely identified by its lemma, language, and class.
func (s *WordsService) AddWord(ctx context.Context, r AddWordRequest) (id int64, err error) {
	id, err = s.store.InsertWord(ctx, store.InsertWordRequst{
		Lemma: r.Lemma,
		Lang:  r.Lang,
		Class: r.Class,
	})
	if err != nil {
		if errors.Is(err, store.ErrExists) {
			se := serr.NewServiceError(err, http.StatusConflict, "word already exists")
			se.Env["lemma"] = r.Lemma
			se.Env["lang"] = string(r.Lang)
			se.Env["class"] = string(r.Class)
			return 0, se
		}

		return 0, fmt.Errorf("insert word: %w", err)
	}

	return id, nil
}

// DeleteWord deletes a word by its ID. If the word is not found, it returns a ServiceError with status code 404.
func (s *WordsService) DeleteWord(ctx context.Context, id int64) error {
	if err := s.store.DeleteWord(ctx, store.DeleteWordRequest{ID: id}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := serr.NewServiceError(err, http.StatusNotFound, "word not found")
			se.Env["word_id"] = fmt.Sprintf("%d", id)
			return se
		}

		return fmt.Errorf("delete word: %w", err)
	}

	return nil
}

type PickWoardRequest struct {
	UserID string
	WordID int64
	DefID  int64
	Tags   []string
}

// PickWord allows a user to pick a word definition for learning. If the pick already exists,
// it returns a ServiceError with status code 409.
func (s *WordsService) PickWord(ctx context.Context, r PickWoardRequest) (int64, error) {
	var pickID int64
	err := s.store.WithTx(ctx, func(tx store.DataStore) error {
		tags, err := s.tags.GetOrCreateTags(ctx, tx, r.Tags)
		if err != nil {
			return fmt.Errorf("get or create tags: %w", err)
		}

		pickID, err = tx.CreateUserPick(ctx, store.CreateUserPickRequest{
			UserID: r.UserID,
			DefID:  r.DefID,
		})
		if err != nil {
			if errors.Is(err, store.ErrExists) {
				se := serr.NewServiceError(err, http.StatusConflict, "user pick already exists")
				se.Env["user_id"] = r.UserID
				se.Env["word_id"] = fmt.Sprintf("%d", r.WordID)
				se.Env["def_id"] = fmt.Sprintf("%d", r.DefID)
				return se
			}

			return fmt.Errorf("create user pick: %w", err)
		}

		err = tx.AddTags(ctx, store.AddTagsRequest{
			PickID: pickID,
			TagIDs: tags.IDs(),
		})
		if err != nil {
			return fmt.Errorf("add tags to pick: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("pick word: %w", err)
	}

	return pickID, nil
}

type GetUserPicksRequest struct {
	UserID      string
	WithTags    []string
	WithoutTags []string
	NextCursor  string
	PageSize    int
}

type GetUserPicksResponse struct {
	Picks      []UserPick
	NextCursor string
}

type UserPick struct {
	ID     int64
	UserID string
	Word   string
	Lang   model.Lang
	Class  model.WordClass
	Def    string
	Tags   []string
}

// GetUserPicks retrieves a paginated list of a user's picked words, optionally filtered by tags.
func (s *WordsService) GetUserPicks(ctx context.Context, r GetUserPicksRequest) (resp GetUserPicksResponse, err error) {
	withTags, missing, err := s.tags.GetTags(ctx, s.store, r.WithTags)
	if err != nil {
		err = fmt.Errorf("get with-tags: %w", err)
		return
	}
	if len(missing) > 0 {
		// No picks can match if some of the requested tags do not exist
		return
	}

	withoutTags, _, err := s.tags.GetTags(ctx, s.store, r.WithoutTags)
	if err != nil {
		err = fmt.Errorf("get without-tags: %w", err)
		return
	}

	cursor := store.GetUserPicksCursor{}
	if r.NextCursor != "" {
		cursor, err = decodeCursor[store.GetUserPicksCursor](r.NextCursor)
		if err != nil {
			se := serr.NewServiceError(err, http.StatusBadRequest, "invalid pagination cursor")
			se.Env["cursor"] = r.NextCursor
			err = se
			return
		}
	}

	response, err := s.store.GetUserPicks(ctx, store.GetUserPicksRequest{
		UserID:      r.UserID,
		WithTags:    withTags.IDs(),
		WithoutTags: withoutTags.IDs(),
		Cursor:      cursor,
		PageSize:    r.PageSize,
	})
	if err != nil {
		err = fmt.Errorf("get user picks: %w", err)
		return
	}

	resp.Picks = fn.Map(response.Picks, func(pick model.UserPick) UserPick {
		return UserPick{
			ID:     pick.ID,
			UserID: pick.UserID,
			Word:   pick.Word.Lemma,
			Lang:   pick.Word.Lang,
			Class:  pick.Word.Class,
			Def:    pick.Definition.Text,
			Tags:   fn.Map(pick.Tags, func(tag model.Tag) string { return tag.Text }),
		}
	})

	resp.NextCursor, err = encodeCursor(response.NextCursor)
	if err != nil {
		err = fmt.Errorf("encode cursor: %w", err)
		return
	}

	return
}

// UnpickWord allows a user to unpick a previously picked word definition.
// If the pick does not exist, it returns a ServiceError with status code 404.
func (s *WordsService) UnpickWord(ctx context.Context, pickID int64) error {
	if err := s.store.DeleteUserPick(ctx, store.DeleteUserPickRequest{PickID: pickID}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := serr.NewServiceError(err, http.StatusNotFound, "user pick was not found")
			se.Env["pick_id"] = fmt.Sprintf("%d", pickID)
			return se
		}

		return fmt.Errorf("delete user pick: %w", err)
	}

	return nil
}

type AddTagsRequest struct {
	PickID int64
	Tags   []string
}

// AddTags adds tags to a user's picked word. If the pick does not exist,
// it returns a ServiceError with status code 404.
func (s *WordsService) AddTags(ctx context.Context, r AddTagsRequest) error {
	err := s.store.WithTx(ctx, func(tx store.DataStore) error {
		tagIDs, err := s.tags.GetOrCreateTags(ctx, tx, r.Tags)
		if err != nil {
			return fmt.Errorf("get or create tags: %w", err)
		}

		err = tx.AddTags(ctx, store.AddTagsRequest{
			PickID: r.PickID,
			TagIDs: tagIDs.IDs(),
		})
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				se := serr.NewServiceError(err, http.StatusNotFound, "user pick was not found")
				se.Env["pick_id"] = fmt.Sprintf("%d", r.PickID)
				return se
			}
			return fmt.Errorf("add tags to pick: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("add tags: %w", err)
	}

	return nil
}

type RemoveTagsRequest struct {
	PickID int64
	Tags   []string
}

// RemoveTag removes a tag from a user's picked word. If the pick does not exist,
// it returns a ServiceError with status code 404.
func (s *WordsService) RemoveTags(ctx context.Context, r RemoveTagsRequest) error {
	tags, _, err := s.tags.GetTags(ctx, s.store, r.Tags)
	if err != nil {
		return fmt.Errorf("get tags: %w", err)
	}

	if tags.Len() == 0 {
		// No tags to remove
		return nil
	}

	if err := s.store.RemoveTags(ctx, store.RemoveTagsRequest{
		PickID: r.PickID,
		TagIDs: tags.IDs(),
	}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := serr.NewServiceError(err, http.StatusNotFound, "user pick was not found")
			se.Env["pick_id"] = fmt.Sprintf("%d", r.PickID)
			return se
		}

		return fmt.Errorf("remove tag from pick: %w", err)
	}

	return nil
}

type CreateDefinitionRequest struct {
	WordID int64
	Text   string
	Rarity float32
	Source model.DataSource
}

// CreateDefinition creates a new definition for a given word.
func (s *WordsService) CreateDefinition(ctx context.Context, r CreateDefinitionRequest) (int64, error) {
	defID, err := s.store.CreateDefinition(ctx, store.CreateDefinitionRequest{
		WordID: r.WordID,
		Text:   r.Text,
		Rarity: r.Rarity,
		Source: r.Source,
	})
	if err != nil {
		if errors.Is(err, store.ErrExists) {
			se := serr.NewServiceError(err, http.StatusConflict, "duplicate definition")
			se.Env["word_id"] = fmt.Sprintf("%d", r.WordID)
			se.Env["text"] = r.Text
			return 0, se
		}
		if errors.Is(err, store.ErrNotFound) {
			se := serr.NewServiceError(err, http.StatusNotFound, "word not found")
			se.Env["word_id"] = fmt.Sprintf("%d", r.WordID)
			return 0, se
		}

		return 0, fmt.Errorf("create definition: %w", err)
	}

	return defID, nil
}

type AttachImageRequest struct {
	DefID    int64
	Source   model.DataSource
	ImageURL *url.URL
}

type AttachImageResponse struct {
	ImageID  int64
	ImageURL *url.URL
}

// AttachImage attaches an image to a word definition.
func (s *WordsService) AttachImage(ctx context.Context, r AttachImageRequest) (AttachImageResponse, error) {
	imageID, err := s.store.AttachImage(ctx, store.AttachImageRequest{
		DefID:    r.DefID,
		ImageURL: r.ImageURL.String(),
		Source:   r.Source,
	})
	if err != nil {
		if errors.Is(err, store.ErrExists) {
			se := serr.NewServiceError(err, http.StatusConflict, "image already attached to definition")
			se.Env["def_id"] = fmt.Sprintf("%d", r.DefID)
			se.Env["image_url"] = r.ImageURL.String()
			return AttachImageResponse{}, se
		}
		if errors.Is(err, store.ErrNotFound) {
			se := serr.NewServiceError(err, http.StatusNotFound, "definition not found")
			se.Env["def_id"] = fmt.Sprintf("%d", r.DefID)
			return AttachImageResponse{}, se
		}

		return AttachImageResponse{}, fmt.Errorf("attach image: %w", err)
	}

	return AttachImageResponse{
		ImageID:  imageID,
		ImageURL: r.ImageURL,
	}, nil
}

func decodeCursor[T any](s string) (T, error) {
	var cursor T
	d := json.NewDecoder(strings.NewReader(s))
	err := d.Decode(&cursor)
	return cursor, err
}

func encodeCursor[T any](c T) (string, error) {
	var sb strings.Builder
	e := json.NewEncoder(&sb)
	err := e.Encode(c)
	return sb.String(), err
}
