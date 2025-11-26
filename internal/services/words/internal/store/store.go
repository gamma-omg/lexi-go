package store

import (
	"context"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
)

type DataStore interface {
	InsertWord(ctx context.Context, r WordInsertRequest) (int64, error)
	DeleteWord(ctx context.Context, r WordDeleteRequest) error
	CreateUserPick(ctx context.Context, r UserPickCreateRequest) (int64, error)
	DeleteUserPick(ctx context.Context, r UserPickDeleteRequest) error
	CreateTags(ctx context.Context, r TagsCreateRequest) (model.TagIDMap, error)
	GetTags(ctx context.Context, r TagsGetRequest) (model.TagIDMap, error)
	AddTags(ctx context.Context, r TagsAddRequest) error
	RemoveTags(ctx context.Context, r TagsRemoveRequest) error
	WithinTx(ctx context.Context, fn func(tx DataStore) error) error
}
