// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"github.com/go-chi/chi"
	chimw "github.com/go-chi/chi/middleware"
	//"github.com/rs/cors"
)

type webMux struct {
	*chi.Mux
}

func NewRouter(server *Server) webMux {
	// Create the chi router
	mux := chi.NewRouter()

	// Configure global middleware
	mux.Use(chimw.Logger)
	mux.Use(chimw.Recoverer)

	// Enable CORS
	// corsMW := cors.Default()
	// mux.Use(corsMW.Handler)

	// JWT: CheckJWT for each request that uses jwtMW.Handler
	// jwtMW := middleware.NewJwtMiddleware(server.SigningKey)

	mux.Get("/", server.root)

	mux.Get("/file/{fileid}", server.File)
	mux.HandleFunc("/upload", server.UploadFile)

	return webMux{mux}
}
