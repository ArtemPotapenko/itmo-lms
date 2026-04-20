package platform

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

type claimsContextKey struct{}

func RequireAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := bearerClaims(secret, r)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRoles(secret string, roles []string, next http.Handler) http.Handler {
	return RequireAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		for _, role := range roles {
			if slices.Contains(claims.Roles, role) {
				next.ServeHTTP(w, r)
				return
			}
		}
		WriteError(w, http.StatusForbidden, "insufficient role")
	}))
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}

func bearerClaims(secret string, r *http.Request) (Claims, error) {
	value := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return ParseToken(secret, value)
}
