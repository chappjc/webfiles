// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"context"
	"net/http"
	"os"
	"strconv"

	"github.com/chappjc/webfiles/middleware"

	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/gorilla/sessions"
)

// WithSession injects a new or existing cookie-managed session into the request
// context.
func (s *Server) WithSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get existing or create new session cookie
		session, err := s.CookieStore.Get(r, "session")
		if err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(r.Context(), middleware.CtxSession, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithUserFileAuthz checks the permission of CtxUser for the file being accessed.
func (s *Server) WithUserFileAuthz(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lookup files associated with this user
		user := middleware.RequestCtxUser(r)
		userFileIDs, err := s.retrieveFileIDsByUser(user)
		if err != nil {
			log.Infof("failed to retrieve file UIDs for user %s (no files uploaded?): %v", user, err)
			next.ServeHTTP(w, r)
			return
		}

		// Extract the file's unique id from the path
		fileID := chi.URLParam(r, "fileid")
		uid, err := strconv.ParseUint(fileID, 16, 64)
		if err != nil {
			log.Errorf("failed to decode UID %s: %v", fileID, err)
			next.ServeHTTP(w, r)
		}

		for _, id := range userFileIDs {
			if id == int64(uid) {
				break
			}
		}

		ctx := context.WithValue(r.Context(), middleware.CtxAuthzed, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithJWTCookie injects a new or existing cookie-managed JWT into the request
// context. The signed token and the session are both embedded.
func (s *Server) WithJWTCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get existing session cookie or make a new one
		jwtCookie, err := s.CookieStore.Get(r, "webfilesJWTSession")
		if err != nil && !os.IsNotExist(err) {
			http.Error(w, "session error: "+err.Error(), http.StatusInternalServerError)
			return
		} else if os.IsNotExist(err) {
			if jwtCookie.IsNew {
				log.Infof("Couldn't find cookie in our store, but got a new one.")
			} else {
				log.Errorf("Couldn't make a new session.")
			}
		}
		// Save the session to generate it's ID
		if err = jwtCookie.Save(r, w); err != nil {
			log.Errorf("Failed to save JWT cookie: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get JWT from first source with valid token:
		// 1. jwtauth context: URL query or HTTP Authorization header
		// 2. existing token in session cookie
		// 3. newly generated token

		// Reuse JWT from jwtauth context, if available already
		Token, _, err := jwtauth.FromContext(r.Context())
		if Token != nil && err == nil {
			// Reparse and validate the token to check expiry
			Token, err = middleware.JWTParse(Token.Raw, s.SigningKey)
		}
		ok := err == nil && Token != nil && Token.Valid

		var token string
		if ok {
			token = Token.Raw
			log.Infof("Reusing token for session from jwtauth: %s", token)
		} else {
			// Extract the JWT from the cookie, or embed a new one.
			token, ok = jwtCookie.Values["JWTToken"].(string)
			if ok {
				// validate the JWT in the cookie
				Token, err := middleware.JWTParse(token, s.SigningKey)
				ok = err == nil && Token != nil && Token.Valid
			}
		}

		// No or invalid JWT from cookie either?
		if ok {
			log.Infof("Existing token from session cookie: %s", token)
		} else {
			// Generate new token
			token, _, err = middleware.NewSignedJWT(s.SigningKey, jwtCookie.ID)
			if err != nil {
				log.Errorf("Failed to sign JWT: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Infof("Generating new token for session: %s", token)

			// store the token string in the session cookie
			jwtCookie.Values["JWTToken"] = token
		}

		// Final validation of token
		JWToken, errParse := middleware.JWTParse(token, s.SigningKey)
		if errParse != nil {
			log.Errorf("Failed to parse signed JWT string: %v", errParse)
			http.Error(w, errParse.Error(), http.StatusBadRequest)
			return
		}

		// Save cookie store
		if err = jwtCookie.Save(r, w); err != nil {
			log.Errorf("Failed to save JWT cookie: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Infof("Session ID: %s", jwtCookie.ID)

		// Patch the request with a "jwt" cookie for downstream processing.
		if _, err = r.Cookie("jwt"); err == http.ErrNoCookie {
			r.AddCookie(sessions.NewCookie("jwt", token, jwtCookie.Options))
		}

		// Inject session and JWT in request context.
		ctx := context.WithValue(r.Context(), middleware.CtxJWTCookie, jwtCookie)
		ctx = context.WithValue(ctx, middleware.CtxToken, token)
		ctx = context.WithValue(ctx, middleware.CtxUser, jwtCookie.ID)
		ctx = jwtauth.NewContext(ctx, JWToken, errParse)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
