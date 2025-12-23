package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

type dbtx interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

type PostgresStore struct {
	db dbtx
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

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetIdentity(ctx context.Context, r GetIdentityRequest) (Identity, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT i.id, i.provider, i.email, i.name, i.picture, i.created_at, i.updated_at,
		        u.id, u.uid, u.created_at, u.updated_at	
		 FROM identities AS i
		 JOIN users AS u ON i.user_id = u.id
		 WHERE i.id=$1 AND i.provider=$2`, r.ID, r.Provider)

	id := Identity{User: User{}}
	err := row.Scan(
		&id.ID,
		&id.Provider,
		&id.Email,
		&id.Name,
		&id.Picture,
		&id.CreatedAt,
		&id.UpdatedAt,
		&id.User.ID,
		&id.User.UID,
		&id.User.CreatedAt,
		&id.User.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return id, ErrNotFound
		}

		return id, fmt.Errorf("scan: %w", err)
	}

	return id, nil
}

func (s *PostgresStore) GetUserIdentity(ctx context.Context, r GetUserIdentityRequest) (Identity, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT i.id, i.provider, i.email, i.name, i.picture, i.created_at, i.updated_at,
		        u.id, u.uid, u.created_at, u.updated_at	
		 FROM identities AS i
		 JOIN users AS u ON i.user_id = u.id
		 WHERE u.uid=$1 AND i.provider=$2`, r.UID, r.Provider)

	id := Identity{User: User{}}
	err := row.Scan(
		&id.ID,
		&id.Provider,
		&id.Email,
		&id.Name,
		&id.Picture,
		&id.CreatedAt,
		&id.UpdatedAt,
		&id.User.ID,
		&id.User.UID,
		&id.User.CreatedAt,
		&id.User.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return id, ErrNotFound
		}

		return id, fmt.Errorf("scan: %w", err)
	}

	return id, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	return id, nil
}

func (s *PostgresStore) CreateUserIdentity(ctx context.Context, r CreateUserIdentityRequest) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "INSERT INTO identities (id, user_id, provider, email, name, picture) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		r.ID,
		r.UserID,
		r.Provider,
		r.Email,
		r.Name,
		r.Picture).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("insert identity: %w", err)
	}

	return id, nil
}

func (s *PostgresStore) WithTx(ctx context.Context, fn func(tx Store) error) error {
	db, ok := s.db.(*sql.DB)
	if !ok {
		return errors.New("already in transaction")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	sx := &PostgresStore{db: tx}
	if err = fn(sx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback: %v after: %w", rbErr, err)
		}

		return fmt.Errorf("transaction: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

type GetUserWithIdentityRequest struct {
	IdentityID int64
	Provider   string
}
