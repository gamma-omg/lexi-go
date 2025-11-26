package store

import "github.com/gamma-omg/lexi-go/internal/services/words/internal/model"

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
	UserID string
	DefID  int64
}

type UserPickDeleteRequest struct {
	PickID int64
}

type UserPicksGetRequest struct {
	UserID      string
	WithTags    []int64
	WithoutTags []int64
	NextPage    string
}

type TagsCreateRequest struct {
	Tags []string
}

type TagsGetRequest struct {
	Tags []string
}

type TagsAddRequest struct {
	PickID int64
	TagIDs []int64
}

type TagsRemoveRequest struct {
	PickID int64
	TagIDs []int64
}
