package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/model"
	"github.com/lib/pq"
)

const (
	errUniqueViolation     pq.ErrorCode = "23505"
	errForeignKeyViolation pq.ErrorCode = "23503"
)

// PostgresStore implements the *store interfaces using PostgreSQL as the backend.
type PostresStore struct {
	db dbx
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

type dbx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func NewPostgresDB(cfg PostgresConfig) (*sql.DB, error) {
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

	return db, nil
}

func NewPostgresStore(db *sql.DB) *PostresStore {
	return &PostresStore{db: db}
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

func (s *PostresStore) CreateUserPick(ctx context.Context, r UserPickCreateRequest) (int64, error) {
	res := s.db.QueryRowContext(ctx, "INSERT INTO user_picks (user_id, def_id) VALUES ($1, $2) RETURNING id", r.UserID, r.DefID)

	var id int64
	err := res.Scan(&id)
	if err != nil {
		if isPqErr(err, errUniqueViolation) {
			return 0, ErrExists
		}

		return 0, fmt.Errorf("create user pick: %w", err)
	}

	return id, nil
}

func (s *PostresStore) DeleteUserPick(ctx context.Context, r UserPickDeleteRequest) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM user_picks WHERE id = $1", r.PickID)
	if err != nil {
		return fmt.Errorf("delete user pick: %w", err)
	}

	return nil
}

func (s *PostresStore) CreateTags(ctx context.Context, r TagsCreateRequest) (model.TagIDMap, error) {
	if len(r.Tags) == 0 {
		return model.TagIDMap{}, nil
	}

	_, err := s.db.ExecContext(ctx, "INSERT INTO tags (tag) SELECT UNNEST($1::text[]) ON CONFLICT (tag) DO NOTHING", pq.Array(r.Tags))
	if err != nil {
		return nil, fmt.Errorf("insert tags: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id, tag FROM tags WHERE tag = ANY($1::text[])", pq.Array(r.Tags))
	if err != nil {
		return nil, fmt.Errorf("query tag ids: %w", err)
	}
	defer rows.Close()

	tagIDMap := make(model.TagIDMap)
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}

		tagIDMap[tag] = id
	}

	return tagIDMap, nil
}

func (s *PostresStore) GetTags(ctx context.Context, r TagsGetRequest) (model.TagIDMap, error) {
	if len(r.Tags) == 0 {
		return model.TagIDMap{}, nil
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id, tag FROM tags WHERE tag = ANY($1::text[])", pq.Array(r.Tags))
	if err != nil {
		return nil, fmt.Errorf("query tag ids: %w", err)
	}
	defer rows.Close()

	tagIDMap := make(model.TagIDMap)
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}

		tagIDMap[tag] = id
	}

	return tagIDMap, nil
}

func (s *PostresStore) AddTags(ctx context.Context, r TagsAddRequest) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO tags_map (pick_id, tag_id) SELECT $1, UNNEST($2::int[])", r.PickID, pq.Array(r.TagIDs))
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

func (s *PostresStore) RemoveTags(ctx context.Context, r TagsRemoveRequest) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM tags_map WHERE pick_id = $1 AND tag_id = ANY($2::int[])", r.PickID, pq.Array(r.TagIDs))
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}

	return nil
}

func (s *PostresStore) WithinTx(ctx context.Context, fn func(tx DataStore) error) error {
	db, ok := s.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("begin tx: already in tx")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx begin: %w", err)
	}

	txStore := &PostresStore{db: tx}
	if err := fn(txStore); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx rollback: %v after: %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}

	return nil
}

func isPqErr(err error, code pq.ErrorCode) bool {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return false
	}

	return pqErr.Code == code
}
