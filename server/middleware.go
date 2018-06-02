// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"

	"github.com/go-chi/jwtauth"

	"github.com/chappjc/webfiles/middleware"
	"github.com/chappjc/webfiles/response"

	"github.com/gorilla/sessions"
)

// WithSession injects a new or existing cookie-managed session into the request
// context.
func (s *Server) WithSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get existing or create new session cookie
		session, err := s.CookieStore.Get(r, "session")
		if err != nil {
			response.Error(w, 400, "session error")
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
			response.Error(w, 400, "session error: "+err.Error())
			return
		} else if os.IsNotExist(err) {
			if jwtCookie.IsNew {
				log.Infof("Couldn't find cookie in our store, but got a new one.")
			} else {
				log.Errorf("Couldn't make a new session.")
			}
		}

		// Extract the JWT from the cookie, or embed a new one.
		token, ok := jwtCookie.Values["JWTToken"].(string)
		if !ok {
			token, _, err = middleware.NewSignedJWT(s.SigningKey, jwtCookie.ID)
			if err != nil {
				log.Errorf("Failed to sign JWT: %v", err)
				response.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			jwtCookie.Values["JWTToken"] = token
			log.Debugln("New token: ", token)
		} else {
			log.Debugln("Existing token: ", token)
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
			response.Error(w, http.StatusBadRequest, errParse.Error())
			return
		}

		// Save cookie store
		if err = jwtCookie.Save(r, w); err != nil {
			log.Errorf("Failed to save JWT cookie: %v", err)
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Patch the request with a "jwt" cookie for downstream processing.
		if _, err = r.Cookie("jwt"); err != nil {
			r.AddCookie(sessions.NewCookie("jwt", token, jwtCookie.Options))
		}

		// Inject session and JWT in request context.
		ctx := context.WithValue(r.Context(), middleware.CtxJWTCookie, jwtCookie)
		ctx = context.WithValue(ctx, middleware.CtxToken, token)
		ctx = jwtauth.NewContext(ctx, JWToken, errParse)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
