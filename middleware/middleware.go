// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware"
	"github.com/chappjc/webfiles/response"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func UseLog(_log *logrus.Logger) {
	log = _log
}

type contextKey int

const (
	CtxToken contextKey = iota
	CtxUser
	CtxAuthed
	CtxSession
)

func CheckAuth(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				ctx := context.WithValue(r.Context(), CtxAuthed, false)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			apitoken := strings.TrimPrefix(authHeader, "Bearer ")
			JWToken, err := jwt.Parse(apitoken, func(token *jwt.Token) (interface{}, error) {
				// validate signing algorithm
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(key), nil
			})
			if err != nil {
				ctx := context.WithValue(r.Context(), CtxAuthed, false)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if claims, ok := JWToken.Claims.(jwt.MapClaims); ok && JWToken.Valid {
				// extract user, and user files
				userName := claims["name"]
				fmt.Print(userName)
				ctx := context.WithValue(r.Context(), CtxAuthed, true)
				ctx = context.WithValue(ctx, CtxUser, userName)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

		})
	}
}

func NewJwtMiddleware(key string) *jwtmiddleware.JWTMiddleware {
	return jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return key, nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
}

func NewToken(key, user string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := jwt.New(jwt.SigningMethodHS256) // jwt.SigningMethodRS512

			claims := make(jwt.MapClaims)
			claims["admin"] = false
			claims["name"] = user
			claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // settings.Get().JWTExpirationDelta
			claims["iat"] = time.Now().Unix()
			token.Claims = claims

			// Sign the token
			signedToken, err := token.SignedString([]byte(key))
			if err != nil {
				response.Error(w, 500, "couldn't sign")
				return
			}

			// Embed in context
			ctx := context.WithValue(r.Context(), CtxToken, signedToken)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
