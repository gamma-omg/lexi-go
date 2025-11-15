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
