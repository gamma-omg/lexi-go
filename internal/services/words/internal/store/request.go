package store

import "github.com/gamma-omg/lexi-go/internal/services/words/internal/model"

type InsertWordRequst struct {
	Lemma string
	Lang  model.Lang
	Class model.WordClass
}

type WordInsertResponse struct {
	ID int64
}

type DeleteWordRequest struct {
	ID int64
}

type CreateUserPickRequest struct {
	UserID string
	DefID  int64
}

type DeleteUserPickRequest struct {
	PickID int64
}

type UserPicksGetRequest struct {
	UserID      string
	WithTags    []int64
	WithoutTags []int64
	NextPage    string
}

type CreateTagsRequest struct {
	Tags []string
}

type GetTagsRequest struct {
	Tags []string
}

type AddTagsRequest struct {
	PickID int64
	TagIDs []int64
}

type RemoveTagsRequest struct {
	PickID int64
	TagIDs []int64
}
