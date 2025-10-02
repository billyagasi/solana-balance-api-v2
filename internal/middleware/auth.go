package middleware

import (
	"context"
	"net/http"

	"github.com/billyagasi/solana-balance-api-v2/internal/store"
)

const headerKey = "X-API-Key"

func APIKeyAuth(store store.APIKeyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(headerKey)
			if key == "" {
				http.Error(w, "missing X-API-Key", http.StatusUnauthorized)
				return
			}
			ok, err := store.IsValidAPIKey(context.Background(), key)
			if err != nil || !ok {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
