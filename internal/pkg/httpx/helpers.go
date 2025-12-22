package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
)

func ReadJSON(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(out)
}

func WriteJSON(w http.ResponseWriter, status int, resp any) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	return enc.Encode(resp)
}

func HandleErr(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request error",
		"error", err,
		"method", r.Method,
		"url", r.URL.String(),
		"remote_addr", r.RemoteAddr,
	)

	var se *serr.ServiceError
	if errors.As(err, &se) {
		http.Error(w, se.Msg, se.StatusCode)
		return
	}

	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
