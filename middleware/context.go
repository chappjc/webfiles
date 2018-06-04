package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
)

type contextKey int

const (
	CtxToken contextKey = iota
	CtxUser
	CtxAuthed
	CtxAuthzed
	CtxSession
	CtxJWTCookie
)

// RequestCtxToken extracts the CtxToken value from the request context.
func RequestCtxToken(r *http.Request) string {
	signedToken, ok := r.Context().Value(CtxToken).(string)
	if !ok {
		log.Debugf("CtxToken not embedded in request context.")
		return ""
	}
	return signedToken
}

// RequestCtxAuthed extracts the CtxAuthed value from the request context.
func RequestCtxAuthed(r *http.Request) bool {
	authed, ok := r.Context().Value(CtxAuthed).(bool)
	if !ok {
		log.Debugf("CtxAuthed not embedded in request context.")
		return false
	}
	return authed
}

// RequestCtxAuthzed extracts the CtxAuthzed value from the request context.
func RequestCtxAuthzed(r *http.Request) bool {
	authzed, ok := r.Context().Value(CtxAuthzed).(bool)
	if !ok {
		log.Debugf("CtxAuthzed not embedded in request context.")
		return false
	}
	return authzed
}

// RequestCtxUser extracts the CtxUser value from the request context.
func RequestCtxUser(r *http.Request) string {
	user, ok := r.Context().Value(CtxUser).(string)
	if !ok {
		log.Debugf("CtxUser not embedded in request context.")
		return ""
	}
	return user
}

// RequestCtxSession extracts the CtxSession value from the request context.
func RequestCtxSession(r *http.Request) *sessions.Session {
	session, ok := r.Context().Value(CtxSession).(*sessions.Session)
	if !ok {
		log.Debugf("CtxSession not embedded in request context.")
		return nil
	}
	return session
}

// RequestCtxJWTSession extracts the CtxJWTCookie value from the request context.
func RequestCtxJWTSession(r *http.Request) *sessions.Session {
	session, ok := r.Context().Value(CtxJWTCookie).(*sessions.Session)
	if !ok {
		log.Debugf("CtxJWTCookie not embedded in request context.")
		return nil
	}
	return session
}
