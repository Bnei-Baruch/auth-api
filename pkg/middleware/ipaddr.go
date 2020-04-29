package middleware

import (
	"net/http"

	"github.com/Bnei-Baruch/auth-api/pkg/httputil"
)

func RealIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rCtx, ok := ContextFromRequest(r)
		if ok {
			rCtx.IP = httputil.GetRealIP(r)
		}
		next.ServeHTTP(w, r)
	})
}
