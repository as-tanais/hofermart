package server

import (
	"net/http"
)

type RouteGroup struct {
	mux         *http.ServeMux
	middlewares []func(http.Handler) http.Handler
	prefix      string
}

func NewRouteGroup(mux *http.ServeMux, prefix string) *RouteGroup {
	return &RouteGroup{
		mux:    mux,
		prefix: prefix,
	}
}

func (rg *RouteGroup) Use(middleware func(http.Handler) http.Handler) {
	rg.middlewares = append(rg.middlewares, middleware)
}

func (rg *RouteGroup) Handle(pattern string, handler http.Handler) {
	fullPattern := rg.prefix + pattern

	wrapped := handler
	for i := len(rg.middlewares) - 1; i >= 0; i-- {
		wrapped = rg.middlewares[i](wrapped)
	}

	rg.mux.Handle(fullPattern, wrapped)
}

func (rg *RouteGroup) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	rg.Handle(pattern, http.HandlerFunc(handler))
}
