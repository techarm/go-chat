package main

import (
	"net/http"

	"github.com/bmizerany/pat"
	"github.com/techarm/go-ws/internal/handlers"
)

func routes() http.Handler {
	mux := pat.New()
	mux.Get("/", http.HandlerFunc(handlers.Home))
	mux.Get("/chat/:name", http.HandlerFunc(handlers.Chat))
	mux.Get("/ws", http.HandlerFunc(handlers.WSEndPoint))

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Get("/static/", http.StripPrefix("/static", fileServer))
	return mux
}
