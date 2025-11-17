package model

import "time"

type Lang string
type WordClass string

const (
	Noun         WordClass = "noun"
	Pronoun      WordClass = "pronoun"
	Verb         WordClass = "verb"
	Adjective    WordClass = "adjective"
	Adverb       WordClass = "adverb"
	Preposition  WordClass = "preposition"
	Conjunction  WordClass = "conjunction"
	Interjection WordClass = "interjection"
)

type DataSource string

const (
	SrcUnknown DataSource = "unknown"
	SrcUser    DataSource = "user"
	SrcAI      DataSource = "ai"
)

type Model struct {
	CreateAt  time.Time
	UpdatedAt time.Time
}

type Word struct {
	Model
	ID     int64
	Lemma  string
	Lang   Lang
	Class  WordClass
	Rarity int
}

type Definition struct {
	Model
	ID     int64
	WordID int64
	Text   string
	Source DataSource
}

type Example struct {
	Model
	ID     int64
	DefID  int64
	Source DataSource
}

type Image struct {
	Model
	ID     int64
	DefID  int64
	URL    string
	Source DataSource
}

type User struct {
	Model
	ID    int64
	Email string
}

type UserPick struct {
	Model
	ID     int64
	UserID int64
	DefID  int64
}
