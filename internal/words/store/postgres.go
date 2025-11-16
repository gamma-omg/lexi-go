package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

const (
	errUniqueViolation     pq.ErrorCode = "23505"
	errForeignKeyViolation pq.ErrorCode = "23503"
)

// PostgresStore implements the *store interfaces using PostgreSQL as the backend.
type PostresStore struct {
	db *sql.DB
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

func NewPostgresStore(cfg PostgresConfig) (*PostresStore, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DB))
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostresStore{db: db}, nil
}

func (s *PostresStore) InsertWord(ctx context.Context, r WordInsertRequest) (int64, error) {
	res := s.db.QueryRowContext(ctx, "INSERT INTO words (lemma, lang, class) VALUES ($1, $2, $3) RETURNING id", r.Lemma, r.Lang, r.Class)

	var id int64
	err := res.Scan(&id)
	if err != nil {
		if isPqErr(err, errUniqueViolation) {
			return 0, ErrExists
		}

		return 0, fmt.Errorf("insert word: %w", err)
	}

	return id, nil
}

func (s *PostresStore) DeleteWord(ctx context.Context, r WordDeleteRequest) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM words WHERE id = $1", r.ID)
	if err != nil {
		return fmt.Errorf("delete word: %w", err)
	}

	return nil
}

func (s *PostresStore) CreateUserPick(ctx context.Context, r UserPickCreateRequest) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2)", r.UserID, r.DefID)
	if err != nil {
		if isPqErr(err, errUniqueViolation) {
			return ErrExists
		}

		return fmt.Errorf("create user pick: %w", err)
	}

	return nil
}

func (s *PostresStore) DeleteUserPick(ctx context.Context, r UserPickDeleteRequest) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM user_picks WHERE id = $1", r.PickID)
	if err != nil {
		return fmt.Errorf("delete user pick: %w", err)
	}

	return nil
}

func (s *PostresStore) GetOrCreateTag(ctx context.Context, tag string) (int64, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id FROM tags WHERE tag = $1", tag)

	var tagID int64
	err := row.Scan(&tagID)
	if err == nil {
		return tagID, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		res := s.db.QueryRowContext(ctx, "INSERT INTO tags (tag) VALUES ($1) RETURNING id", tag)

		var id int64
		err := res.Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("create tag: %w", err)
		}

		return id, nil
	}

	return 0, fmt.Errorf("query tag id: %w", err)
}

func (s *PostresStore) AddTag(ctx context.Context, r UserPickAddTagRequest) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO tags_map (pick_id, tag_id) VALUES ($1, $2)", r.PickID, r.TagID)
	if err != nil {
		if isPqErr(err, errUniqueViolation) {
			return ErrExists
		}
		if isPqErr(err, errForeignKeyViolation) {
			return ErrNotFound
		}

		return fmt.Errorf("insert tag: %w", err)
	}

	return nil
}

func (s *PostresStore) RemoveTag(ctx context.Context, r UserPickRemoveTagRequest) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM tags_map WHERE pick_id = $1 AND tag_id = $2", r.PickID, r.TagID)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}

	return nil
}

func (s *PostresStore) Close() error {
	return s.db.Close()
}

func isPqErr(err error, code pq.ErrorCode) bool {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return false
	}

	return pqErr.Code == code
}
