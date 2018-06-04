// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"github.com/chappjc/webfiles/middleware"

	"github.com/go-chi/chi"
	chimw "github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
)

// WebMux is the http path multiplexer
type WebMux struct {
	*chi.Mux
}

// NewRouter creates a new WebMux for the specified Server. This configures the
// middleware, and the route mapping.
func NewRouter(server *Server) WebMux {
	// Create the chi router
	mux := chi.NewRouter()

	// Configure global middleware
	//mux.Use(chimw.Logger)
	mux.Use(chimw.Recoverer)

	// Enable CORS (github.com/rs/cors)
	// corsMW := cors.Default()
	// mux.Use(corsMW.Handler)

	// Regular cookie session (no JWT embedded)
	//mux.Use(server.WithSession)

	// Verify JWT from URI query or HTTP (Authorization) header.
	jwtFromQueryOrHeader := middleware.JWTVerify(server.AuthToken,
		true, server.CookieStore.Options, // inject found token into "jwt" cookie
		jwtauth.TokenFromQuery, jwtauth.TokenFromHeader)
	mux.Use(jwtFromQueryOrHeader)

	// Find session cookie "webfilesJWTSession" with JWT data or create a new
	// token and cookie.
	mux.Use(server.WithJWTCookie)

	// Verify JWT from "jwt" cookie
	mux.Use(middleware.JWTVerify(server.AuthToken, false, nil, jwtauth.TokenFromCookie))

	mux.Get("/", server.root)
	mux.Get("/token", server.Token)
	mux.HandleFunc("/upload", server.UploadFile)
	mux.With(middleware.JWTAuthenticator, server.WithUserFileAuthz).Get("/file/{fileid}", server.File)
	mux.With(middleware.JWTAuthenticator).Get("/user-files", server.FileList)
	return WebMux{mux}
}
