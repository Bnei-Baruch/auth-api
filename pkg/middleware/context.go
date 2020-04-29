package middleware

import (
	"context"
	"net/http"
)

type RequestContext struct {
	IP       string
	IDClaims *IDTokenClaims
}

type requestCtx struct{}

func ContextFromRequest(r *http.Request) (*RequestContext, bool) {
	if r == nil {
		return nil, false
	}
	return ContextFromCtx(r.Context())
}

func ContextFromCtx(ctx context.Context) (*RequestContext, bool) {
	rCtx, ok := ctx.Value(requestCtx{}).(*RequestContext)
	return rCtx, ok
}

func ContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), requestCtx{}, new(RequestContext)))
		next.ServeHTTP(w, r)
	})
}
