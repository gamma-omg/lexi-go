package router

import (
	"net/http"
	"strings"
)

type Router struct {
	prefix     string
	mux        *http.ServeMux
	middleware []Middleware
}

func New() *Router {
	return &Router{
		prefix: "",
		mux:    http.NewServeMux(),
	}
}

func (rt *Router) Use(mw ...Middleware) {
	rt.middleware = append(rt.middleware, mw...)
}

func (rt *Router) Handle(pattern string, handler http.Handler) {
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	rt.mux.Handle(pattern, handler)
}

func (rt *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	rt.mux.HandleFunc(pattern, handler)
}

func (rt *Router) SubRouter(prefix string) *Router {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		panic("empty subrout")
	}

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	s := &Router{
		prefix:     prefix,
		mux:        http.NewServeMux(),
		middleware: rt.middleware,
	}

	rt.mux.Handle(prefix+"/", http.StripPrefix(prefix, s))
	return s
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var h http.Handler = rt.mux
	for i := len(rt.middleware) - 1; i >= 0; i-- {
		h = rt.middleware[i](h)
	}

	h.ServeHTTP(w, r)
}
