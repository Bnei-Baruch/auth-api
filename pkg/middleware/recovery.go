package middleware

import (
	"fmt"
	"net/http"

	"github.com/Bnei-Baruch/auth-api/pkg/httputil"
)

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				err, ok := p.(error)
				if !ok {
					err = fmt.Errorf("panic: %+v", p)
				}

				httputil.NewInternalError(err).Abort(w, r)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
