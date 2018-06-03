// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/chappjc/webfiles/middleware"

	"github.com/dgrijalva/jwt-go"
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

		// Extract the JWT from the cookie, or embed a new one.
		token, ok := jwtCookie.Values["JWTToken"].(string)
		if !ok {
			// reuse validated JWT from jwtauth context, if available already
			Token, _, err := jwtauth.FromContext(r.Context())
			if err == nil && Token != nil && Token.Valid {
				token = Token.Raw
				log.Infof("Reusing token for session from jwtauth: %s", token)
			} else {
				// Generate new token
				fmt.Println("ID: ", jwtCookie.ID)
				token, _, err = middleware.NewSignedJWT(s.SigningKey, jwtCookie.ID)
				if err != nil {
					log.Errorf("Failed to sign JWT: %v", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				log.Infof("Generating new token for session: %s", token)
			}

			// store the token string in the session cookie
			jwtCookie.Values["JWTToken"] = token
		} else {
			log.Infof("Existing token from session cookie: %s", token)
		}

		JWToken, errParse := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			// validate signing algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.SigningKey), nil
		})
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

		// Patch the request with a "jwt" cookie for downstream processing.
		if _, err = r.Cookie("jwt"); err == http.ErrNoCookie {
			r.AddCookie(sessions.NewCookie("jwt", token, jwtCookie.Options))
		}

		// Inject session and JWT in request context.
		ctx := context.WithValue(r.Context(), middleware.CtxJWTCookie, jwtCookie)
		ctx = context.WithValue(ctx, middleware.CtxToken, token)
		ctx = jwtauth.NewContext(ctx, JWToken, errParse)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
