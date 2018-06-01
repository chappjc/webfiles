// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"context"
	"net/http"

	"github.com/chappjc/webfiles/middleware"
	"github.com/chappjc/webfiles/response"
)

func (s *Server) WithSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.CookieStore.Get(r, "session")
		if err != nil {
			response.Error(w, 400, "session error")
			return
		}
		ctx := context.WithValue(r.Context(), middleware.CtxSession, session)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	})
}
