package store

import (
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
)

type InsertWordRequst struct {
	Lemma string
	Lang  model.Lang
	Class model.WordClass
}

type InsertWordResponse struct {
	ID int64
}

type DeleteWordRequest struct {
	ID int64
}

type CreateUserPickRequest struct {
	UserID string
	DefID  int64
}

type GetUserPicksCursor struct {
	LastPickID int64
}

type GetUserPicksRequest struct {
	UserID      string
	WithTags    []int64
	WithoutTags []int64
	PageSize    int
	Cursor      GetUserPicksCursor
}

type GetUserPicksResponse struct {
	Picks      []model.UserPick
	NextCursor *GetUserPicksCursor
}

type DeleteUserPickRequest struct {
	PickID int64
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

type CreateDefinitionRequest struct {
	WordID int64
	Text   string
	Rarity float32
	Source model.DataSource
}

type AttachImageRequest struct {
	DefID    int64
	ImageURL string
	Source   model.DataSource
}
