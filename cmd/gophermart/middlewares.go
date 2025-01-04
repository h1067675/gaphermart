package main

import (
	"context"
	"net/http"

	"github.com/h1067675/gophermart/internal/authorization"
	"github.com/h1067675/gophermart/internal/logger"
)

func (c *Connect) CookieAuthorizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var (
			err    error
			userid int
			cookie *http.Cookie
			ctx    context.Context
		)
		logger.Log.Info("checking authorization")
		cookie, err = request.Cookie("token")
		if err != nil {
			logger.Log.Info("user is not logged")
		} else {
			userid, err = authorization.CheckToken(cookie.Value)
			if err != nil {
				logger.Log.WithError(err).Error("user is not logged")
			} else {
				ctx = context.WithValue(request.Context(), KeyUserID, userid)
			}
		}
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}