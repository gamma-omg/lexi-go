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
	Unknown DataSource = "unknown"
	User    DataSource = "user"
	AI      DataSource = "ai"
)

type Model struct {
	CreateAt  time.Time
	UpdatedAt time.Time
}

type Word struct {
	Model
	ID     string
	Lemma  string
	Lang   Lang
	Class  WordClass
	Rarity int
}

type Definition struct {
	Model
	ID     string
	WordID string
	Text   string
	Source DataSource
}

type Example struct {
	Model
	ID     string
	DefID  string
	Source DataSource
}

type Image struct {
	Model
	ID     string
	WordID string
	URL    string
	Source DataSource
}

type UserPick struct {
	Model
	ID     string
	WordID string
	DefID  string
}
