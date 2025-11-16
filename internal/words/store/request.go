package store

import "github.com/gamma-omg/lexi-go.git/internal/words/model"

type WordInsertRequest struct {
	Lemma string
	Lang  model.Lang
	Class model.WordClass
}

type WordInsertResponse struct {
	ID int64
}

type WordDeleteRequest struct {
	ID int64
}

type UserPickCreateRequest struct {
	UserID int64
	DefID  int64
}

type UserPickDeleteRequest struct {
	PickID int64
}

type UserPickAddTagRequest struct {
	PickID int64
	TagID  int64
}

type UserPickRemoveTagRequest struct {
	PickID int64
	TagID  int64
}
