package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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

type WordAddRequest struct {
	Lemma string
	Lang  model.Lang
	Class model.WordClass
}

// AddWord adds a new word to the global word list. If the word already exists,
// it returns a ServiceError with status code 409. The word is uniquely identified by its lemma, language, and class.
func (s *WordsService) AddWord(ctx context.Context, r WordAddRequest) (id int64, err error) {
	id, err = s.store.InsertWord(ctx, store.WordInsertRequest{
		Lemma: r.Lemma,
		Lang:  r.Lang,
		Class: r.Class,
	})
	if err != nil {
		if errors.Is(err, store.ErrExists) {
			se := NewServiceError(err, http.StatusConflict, "word already exists")
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
	if err := s.store.DeleteWord(ctx, store.WordDeleteRequest{ID: id}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := NewServiceError(err, http.StatusNotFound, "word not found")
			se.Env["word_id"] = fmt.Sprintf("%d", id)
			return se
		}

		return fmt.Errorf("delete word: %w", err)
	}

	return nil
}

type UserPickWordRequest struct {
	UserID string
	WordID int64
	DefID  int64
	Tags   []string
}

// PickWord allows a user to pick a word definition for learning. If the pick already exists,
// it returns a ServiceError with status code 409.
func (s *WordsService) PickWord(ctx context.Context, r UserPickWordRequest) error {
	err := s.store.WithinTx(ctx, func(tx store.DataStore) error {
		tags, err := s.tags.GetOrCreateTags(ctx, tx, r.Tags)
		if err != nil {
			return fmt.Errorf("get or create tags: %w", err)
		}

		pickID, err := tx.CreateUserPick(ctx, store.UserPickCreateRequest{
			UserID: r.UserID,
			DefID:  r.DefID,
		})
		if err != nil {
			if errors.Is(err, store.ErrExists) {
				se := NewServiceError(err, http.StatusConflict, "user pick already exists")
				se.Env["user_id"] = r.UserID
				se.Env["word_id"] = fmt.Sprintf("%d", r.WordID)
				se.Env["def_id"] = fmt.Sprintf("%d", r.DefID)
				return se
			}

			return fmt.Errorf("create user pick: %w", err)
		}

		err = tx.AddTags(ctx, store.TagsAddRequest{
			PickID: pickID,
			TagIDs: tags.IDs(),
		})
		if err != nil {
			return fmt.Errorf("add tags to pick: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("pick word: %w", err)
	}

	return nil
}

// UnpickWord allows a user to unpick a previously picked word definition.
// If the pick does not exist, it returns a ServiceError with status code 404.
func (s *WordsService) UnpickWord(ctx context.Context, pickID int64) error {
	if err := s.store.DeleteUserPick(ctx, store.UserPickDeleteRequest{PickID: pickID}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := NewServiceError(err, http.StatusNotFound, "user pick was not found")
			se.Env["pick_id"] = fmt.Sprintf("%d", pickID)
			return se
		}

		return fmt.Errorf("delete user pick: %w", err)
	}

	return nil
}

type UserPickAddTagRequest struct {
	PickID int64
	Tags   []string
}

// AddTags adds tags to a user's picked word. If the pick does not exist,
// it returns a ServiceError with status code 404.
func (s *WordsService) AddTags(ctx context.Context, r UserPickAddTagRequest) error {
	err := s.store.WithinTx(ctx, func(tx store.DataStore) error {
		tagIDs, err := s.tags.GetOrCreateTags(ctx, tx, r.Tags)
		if err != nil {
			return fmt.Errorf("get or create tags: %w", err)
		}

		err = tx.AddTags(ctx, store.TagsAddRequest{
			PickID: r.PickID,
			TagIDs: tagIDs.IDs(),
		})
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				se := NewServiceError(err, http.StatusNotFound, "user pick was not found")
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

type UserPickRemoveTagRequest struct {
	PickID int64
	TagID  int64
}

// RemoveTag removes a tag from a user's picked word. If the pick does not exist,
// it returns a ServiceError with status code 404.
func (s *WordsService) RemoveTag(ctx context.Context, r UserPickRemoveTagRequest) error {
	if err := s.store.RemoveTags(ctx, store.TagsRemoveRequest{
		PickID: r.PickID,
		TagIDs: []int64{r.TagID},
	}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := NewServiceError(err, http.StatusNotFound, "user pick was not found")
			se.Env["pick_id"] = fmt.Sprintf("%d", r.PickID)
			se.Env["tag_id"] = fmt.Sprintf("%d", r.TagID)
			return se
		}

		return fmt.Errorf("remove tag from pick: %w", err)
	}

	return nil
}
