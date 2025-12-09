package env_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/env"
	"github.com/stretchr/testify/assert"
)

func TestRequireString(t *testing.T) {
	t.Setenv("TEST_REQUIRED_STRING", "required_value")
	assert.Equal(t, "required_value", env.RequireString("TEST_REQUIRED_STRING"))
}

func TestRequireString_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	env.RequireString("NON_EXISTENT_REQUIRED_STRING")
}

func TestString(t *testing.T) {
	t.Setenv("TEST_STRING", "hello")
	assert.Equal(t, "hello", env.String("TEST_STRING", "default"))
	assert.Equal(t, "default", env.String("NON_EXISTENT_STRING", "default"))
}

func TestInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	assert.Equal(t, int(42), env.Int("TEST_INT", 100))
	assert.Equal(t, int(100), env.Int("NON_EXISTENT_INT", 100))
}

func TestInt64(t *testing.T) {
	t.Setenv("TEST_INT64", "4200")
	assert.Equal(t, int64(4200), env.Int64("TEST_INT64", 1000))
	assert.Equal(t, int64(1000), env.Int64("NON_EXISTENT_INT64", 1000))
}

func TestBool(t *testing.T) {
	t.Setenv("TEST_BOOL", "true")
	t.Setenv("TEST_BOOL_1", "1")
	assert.Equal(t, true, env.Bool("TEST_BOOL", false))
	assert.Equal(t, true, env.Bool("TEST_BOOL_1", false))
	assert.Equal(t, false, env.Bool("NON_EXISTENT_BOOL", false))
}

func TestFloat64(t *testing.T) {
	t.Setenv("TEST_FLOAT", "3.14")
	assert.Equal(t, 3.14, env.Float64("TEST_FLOAT", 1.0))
	assert.Equal(t, 1.0, env.Float64("NON_EXISTENT_FLOAT", 1.0))
}

func TestDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "2h45m")
	assert.Equal(t, 2*time.Hour+45*time.Minute, env.Duration("TEST_DURATION", time.Minute))
	assert.Equal(t, time.Minute, env.Duration("NON_EXISTENT_DURATION", time.Minute))
}

func TestURL(t *testing.T) {
	t.Setenv("TEST_URL", "http://example.com")
	expectedURL, _ := url.Parse("http://example.com")
	assert.Equal(t, expectedURL, env.Url("TEST_URL", &url.URL{Scheme: "http", Host: "default.com"}))

	defaultURL, _ := url.Parse("http://default.com")
	assert.Equal(t, defaultURL, env.Url("NON_EXISTENT_URL", &url.URL{Scheme: "http", Host: "default.com"}))
}

func TestURL_Invalid(t *testing.T) {
	t.Setenv("TEST_INVALID_URL", "://invalid-url")
	defaultURL, _ := url.Parse("http://default.com")
	assert.Equal(t, defaultURL, env.Url("TEST_INVALID_URL", &url.URL{Scheme: "http", Host: "default.com"}))
}
