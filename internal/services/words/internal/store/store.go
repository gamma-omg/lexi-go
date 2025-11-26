package store

import (
	"context"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
)

type DataStore interface {
	InsertWord(ctx context.Context, r InsertWordRequst) (int64, error)
	DeleteWord(ctx context.Context, r DeleteWordRequest) error
	CreateUserPick(ctx context.Context, r CreateUserPickRequest) (int64, error)
	GetUserPicks(ctx context.Context, r GetUserPicksRequest) (GetUserPicksResponse, error)
	DeleteUserPick(ctx context.Context, r DeleteUserPickRequest) error
	CreateTags(ctx context.Context, r CreateTagsRequest) (model.TagIDMap, error)
	GetTags(ctx context.Context, r GetTagsRequest) (model.TagIDMap, error)
	AddTags(ctx context.Context, r AddTagsRequest) error
	RemoveTags(ctx context.Context, r RemoveTagsRequest) error
	WithinTx(ctx context.Context, fn func(tx DataStore) error) error
}
