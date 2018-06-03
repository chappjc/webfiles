// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// UseLog sets an external logger for use by this package.
func UseLog(_log *logrus.Logger) {
	log = _log
}

// JWTAuthenticator allows handers to proceed given a validated token identified
// via jwtauth.FromContext. This should be used after jwtauth.Verify/Verifier.
func JWTAuthenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, claims, err := jwtauth.FromContext(r.Context())
		if err != nil || token == nil || !token.Valid {
			http.Error(w, http.StatusText(401), 401)
			return
		}

		user := claims["user"]
		fmt.Println("user: ", user)

		ctx := context.WithValue(r.Context(), CtxAuthed, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// JWTVerify middleware verifies a JWT from sources defined via the findTokenFns
// functions. If the jwtauth verification has already been successfully
// performed for this request, the verifications functions are not run. If
// injectJWTCookie is true, any located token will be injected as a request
// cookie, "jwt" so that the session cookie middleware may reuse it. If not
// injecting a cookie into the request, cookieOpts may be nil.
func JWTVerify(ja *jwtauth.JWTAuth, injectJWTCookie bool, cookieOpts *sessions.Options,
	findTokenFns ...func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			// Skip additional verification if already done
			token, _, err := jwtauth.FromContext(r.Context())
			if err == nil && token != nil && token.Valid {
				next.ServeHTTP(w, r)
				return
			}
			// Perform JWT verfication and store the token and result in the
			// request context.
			token, err = jwtauth.VerifyRequest(ja, r, findTokenFns...)
			if err == nil && token != nil && injectJWTCookie && cookieOpts != nil {
				r.AddCookie(sessions.NewCookie("jwt", token.Raw, cookieOpts))
			}

			ctx := jwtauth.NewContext(r.Context(), token, err)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(hfn)
	}
}

// NewSignedJWT generates a new JWT, signs the input key and returns the result
// along with the claims map and an error value.
func NewSignedJWT(key, user string) (string, jwt.MapClaims, error) {
	claims := jwt.MapClaims{
		"user": user,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	signedToken, err := token.SignedString([]byte(key))
	return signedToken, claims, err
}
