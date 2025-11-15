package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gamma-omg/lexi-go.git/internal/words/model"
	"github.com/gamma-omg/lexi-go.git/internal/words/store"
)

type wordStore interface {
	InsertWord(ctx context.Context, r store.WordInsertRequest) (string, error)
	DeleteWord(ctx context.Context, r store.WordDeleteRequest) error
}

// WordService provides access to the global word list and related operations
type WordService struct {
	store wordStore
}

type WordAddRequest struct {
	Lemma string
	Lang  model.Lang
	Class model.WordClass
}

// AddWord adds a new word to the global word list. If the word already exists, it returns a ServiceError with status code 409.
// The word is uniquely identified by its lemma, language, and class.
func (s *WordService) AddWord(ctx context.Context, r WordAddRequest) (id string, err error) {
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
			return "", se
		}

		return "", fmt.Errorf("insert word: %w", err)
	}

	return id, nil
}

// DeleteWord deletes a word by its ID. If the word is not found, it returns a ServiceError with status code 404.
func (s *WordService) DeleteWord(ctx context.Context, id string) error {
	if err := s.store.DeleteWord(ctx, store.WordDeleteRequest{ID: id}); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			se := NewServiceError(err, http.StatusNotFound, "word not found")
			se.Env["word_id"] = id
			return se
		}

		return fmt.Errorf("delete word: %w", err)
	}

	return nil
}
