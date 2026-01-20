package otc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/service"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	rdb *redis.Client
	ttl time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	TTL      time.Duration
}

func NewRedis(cfg RedisConfig) *Redis {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Redis{
		rdb: rdb,
		ttl: cfg.TTL,
	}
}

type codeEntry struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r *Redis) CreateCode(ctx context.Context, ts service.TokenPair) (string, error) {
	ce := codeEntry{
		AccessToken:  ts.AccessToken,
		RefreshToken: ts.RefreshToken,
	}

	var sb strings.Builder
	err := json.NewEncoder(&sb).Encode(ce)
	if err != nil {
		return "", fmt.Errorf("serializes tokens: %w", err)
	}

	for range 3 {
		code := generateCode()
		ok, err := r.rdb.SetNX(ctx, code, sb.String(), r.ttl).Result()
		if err != nil {
			return "", fmt.Errorf("store code in redis: %w", err)
		}
		if ok {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code")
}

func (r *Redis) RedeemCode(ctx context.Context, code string) (service.TokenPair, error) {
	val, err := r.rdb.GetDel(ctx, code).Result()
	if err != nil {
		if err == redis.Nil {
			return service.TokenPair{}, fmt.Errorf("code not found")
		}

		return service.TokenPair{}, fmt.Errorf("retrieve code from redis: %w", err)
	}

	var ce codeEntry
	err = json.NewDecoder(strings.NewReader(val)).Decode(&ce)
	if err != nil {
		return service.TokenPair{}, fmt.Errorf("deserialize code entry: %w", err)
	}

	return service.TokenPair{
		AccessToken:  ce.AccessToken,
		RefreshToken: ce.RefreshToken,
	}, nil
}

func (r *Redis) Close() error {
	return r.rdb.Close()
}

func generateCode() string {
	return base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))
}
