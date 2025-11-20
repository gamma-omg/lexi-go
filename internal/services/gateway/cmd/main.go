package main

import (
	"fmt"
	"log/slog"
	"net/http"
)

func main() {
	slog.Info("starting api gateway")

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "This is API gateway")
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error(err.Error())
	}
}
