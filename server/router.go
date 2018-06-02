// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"github.com/chappjc/webfiles/middleware"

	"github.com/go-chi/chi"
	chimw "github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
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

	// Regular cookie session (no JWT embedded)
	//mux.Use(server.WithSession)

	// Verify JWT from URI query or HTTP (Authorization) header.
	mux.Use(middleware.JWTVerify(server.AuthToken, jwtauth.TokenFromQuery, jwtauth.TokenFromHeader))

	// Find session cookie "webfilesJWTSession" with JWT data or create a new
	// token and cookie.
	mux.Use(server.WithJWTCookie)

	// Verify JWT from cookie
	mux.Use(middleware.JWTVerify(server.AuthToken, jwtauth.TokenFromCookie))

	mux.Get("/", server.root)
	mux.Get("/token", server.Token)
	mux.HandleFunc("/upload", server.UploadFile)
	mux.With(middleware.JWTAuthenticator).Get("/file/{fileid}", server.File)
	return webMux{mux}
}
