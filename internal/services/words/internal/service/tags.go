package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
)

type tagSet struct {
	ids model.TagIDMap
}

func newEmptyTagSet() tagSet {
	return tagSet{ids: make(model.TagIDMap)}
}

func (t *tagSet) IDs() []int64 {
	result := make([]int64, 0, len(t.ids))
	for _, id := range t.ids {
		result = append(result, id)
	}
	return result
}

func (t *tagSet) Tags() []string {
	result := make([]string, 0, len(t.ids))
	for tag := range t.ids {
		result = append(result, tag)
	}
	return result
}

func (t *tagSet) GetID(tag string) (int64, bool) {
	id, found := t.ids[tag]
	return id, found
}

func (t *tagSet) AddTag(tag string, id int64) {
	t.ids[tag] = id
}

func (t *tagSet) Merge(other model.TagIDMap) {
	for tag, id := range other {
		t.ids[tag] = id
	}
}

func (t *tagSet) Remove(other model.TagIDMap) {
	for tag := range other {
		delete(t.ids, tag)
	}
}

func (t *tagSet) Len() int {
	return len(t.ids)
}

// tagsStore defines the methods required to manage tags in the data store.
type tagsStore interface {
	CreateTags(ctx context.Context, r store.CreateTagsRequest) (model.TagIDMap, error)
	GetTags(ctx context.Context, r store.GetTagsRequest) (model.TagIDMap, error)
}

// tagManager handles retrieval and creation of tags with caching.
type tagManager struct {
	cache *ristretto.Cache[string, int64]
}

func newTagManager(maxKeys, maxCost int64) *tagManager {
	c, err := ristretto.NewCache(&ristretto.Config[string, int64]{
		NumCounters: maxKeys * 10,
		MaxCost:     maxCost,
		BufferItems: 64,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create tag manager cache: %v", err))
	}

	return &tagManager{cache: c}
}

// GetOrCreateTags retrieves tag IDs for the given tags, creating any that do not already exist.
func (tm *tagManager) GetOrCreateTags(ctx context.Context, ts tagsStore, tags []string) (tagSet, error) {
	existing, missing, err := tm.GetTags(ctx, ts, tags)
	if err != nil {
		return tagSet{}, fmt.Errorf("get tags: %w", err)
	}

	if len(missing) > 0 {
		created, err := ts.CreateTags(ctx, store.CreateTagsRequest{Tags: missing})
		if err != nil {
			return tagSet{}, fmt.Errorf("create tags: %w", err)
		}

		existing.Merge(created)
		tm.storeToCache(created)
	}

	return existing, nil
}

// GetTags retrieves tag IDs for the given tags, returning any that are not found.
func (tm *tagManager) GetTags(ctx context.Context, ts tagsStore, tags []string) (tagSet, []string, error) {
	result := newEmptyTagSet()
	missing := newEmptyTagSet()

	for _, tag := range tags {
		if id, found := tm.cache.Get(tag); found {
			result.AddTag(tag, id)
		} else {
			missing.AddTag(tag, 0)
		}
	}

	if missing.Len() > 0 {
		fetched, err := ts.GetTags(ctx, store.GetTagsRequest{Tags: missing.Tags()})
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return newEmptyTagSet(), nil, fmt.Errorf("failed to get tags: %w", err)
		}

		tm.storeToCache(fetched)
		missing.Remove(fetched)
		result.Merge(fetched)
	}

	return result, missing.Tags(), nil
}

func (tm *tagManager) storeToCache(tags model.TagIDMap) {
	for tag, id := range tags {
		tm.cache.Set(tag, id, 1)
	}
}
