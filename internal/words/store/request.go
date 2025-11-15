package store

import "github.com/gamma-omg/lexi-go.git/internal/words/model"

type WordInsertRequest struct {
	Lemma string
	Lang  model.Lang
	Class model.WordClass
}

type WordInsertResponse struct {
	ID string
}

type WordDeleteRequest struct {
	ID string
}

type UserPickCreateRequest struct {
	UserID string
	WordID string
	DefID  string
}

type UserPickDeleteRequest struct {
	UserID string
	WordID string
	DefID  string
}

type UserPickAddTagRequest struct {
	PickID string
	TagID  string
}

type UserPickRemoveTagRequest struct {
	PickID string
	TagID  string
}
