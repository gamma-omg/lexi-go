package otc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/service"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type redisStartResponse struct {
	Host string
	Port string
}

var (
	redisHost string
	redisPort string
)

func startRedis(ctx context.Context) (redisStartResponse, func()) {
	r := testcontainers.ContainerRequest{
		Image:        "redis:8.4-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: r,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	host, err := cont.Host(ctx)
	if err != nil {
		panic(err)
	}

	port, err := cont.MappedPort(ctx, "6379")
	if err != nil {
		panic(err)
	}

	closer := func() {
		cont.Terminate(ctx)
	}

	return redisStartResponse{
		Host: host,
		Port: port.Port(),
	}, closer
}

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, closeRedis := startRedis(ctx)
	defer closeRedis()

	redisHost = resp.Host
	redisPort = resp.Port
	os.Exit(m.Run())
}

func TestRedisOTC(t *testing.T) {
	rds := NewRedis(RedisConfig{
		Host:     redisHost,
		Port:     redisPort,
		Password: "",
		DB:       0,
		TTL:      30 * time.Second,
	})

	code, err := rds.CreateCode(context.Background(), service.TokenPair{
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
	})
	require.NoError(t, err)
	require.NotEmpty(t, code)

	tokPair, err := rds.RedeemCode(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, "access_token", tokPair.AccessToken)
	require.Equal(t, "refresh_token", tokPair.RefreshToken)
}

func TestRedisOTC_Expires(t *testing.T) {
	rds := NewRedis(RedisConfig{
		Host:     redisHost,
		Port:     redisPort,
		Password: "",
		DB:       0,
		TTL:      1 * time.Second,
	})

	code, err := rds.CreateCode(context.Background(), service.TokenPair{
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
	})
	require.NoError(t, err)
	require.NotEmpty(t, code)

	time.Sleep(2 * time.Second)

	_, err = rds.RedeemCode(context.Background(), code)
	require.Error(t, err)
}
