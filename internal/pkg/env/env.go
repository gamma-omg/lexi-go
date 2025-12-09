package env

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

func RequireString(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("environment variable %q is required", key))
	}

	return val
}

func String(key, def string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	return val
}

func Int(key string, def int) int {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	var val int
	_, err := fmt.Sscanf(valStr, "%d", &val)
	if err != nil {
		return def
	}

	return val
}

func Int64(key string, def int64) int64 {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	var val int64
	_, err := fmt.Sscanf(valStr, "%d", &val)
	if err != nil {
		return def
	}

	return val
}

func Bool(key string, def bool) bool {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	switch valStr {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}

	return def
}

func Float64(key string, def float64) float64 {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	var val float64
	_, err := fmt.Sscanf(valStr, "%f", &val)
	if err != nil {
		return def
	}

	return val
}

func Duration(key string, def time.Duration) time.Duration {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	val, err := time.ParseDuration(valStr)
	if err != nil {
		return def
	}

	return val
}

func Url(key string, def *url.URL) *url.URL {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	parsed, err := url.Parse(val)
	if err != nil {
		return def
	}

	return parsed
}
