package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/billyagasi/solana-balance-api-v2/internal/api"
	"github.com/billyagasi/solana-balance-api-v2/internal/cache"
	"github.com/billyagasi/solana-balance-api-v2/internal/limiter"
	mw "github.com/billyagasi/solana-balance-api-v2/internal/middleware"
	"github.com/billyagasi/solana-balance-api-v2/internal/rpc"
	"github.com/billyagasi/solana-balance-api-v2/internal/store"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	// Config
	port := getenv("PORT", "8080")
	rpcURL := getenv("RPC_URL", "")
	mongoURI := getenv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getenv("MONGO_DB", "solana_api")
	colName := getenv("MONGO_COLLECTION", "api_keys")
	ratePerMin := limiter.ParseInt(getenv("RATE_LIMIT_PER_MIN", "10"), 10)
	cacheTTL := time.Duration(limiter.ParseInt(getenv("CACHE_TTL_SECONDS", "10"), 10)) * time.Second
	reqTimeout := time.Duration(limiter.ParseInt(getenv("REQUEST_TIMEOUT_SECONDS", "10"), 10)) * time.Second
	maxBodyBytes := int64(limiter.ParseInt(getenv("MAX_BODY_BYTES", "1048576"), 1048576))
	discordWebhook := os.Getenv("DISCORD_WEBHOOK_URL")

	if rpcURL == "" {
		log.Fatal("RPC_URL is required")
	}

	ctx := context.Background()

	// Mongo
	apiKeyStore, err := store.NewMongo(ctx, mongoURI, dbName, colName)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	defer apiKeyStore.Close(ctx)

	// Solana RPC client
	solClient := rpc.NewSolanaClient(rpcURL)

	// Cache + singleflight
	c := cache.New(cacheTTL)

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Compress(5))
	r.Use(chimw.Timeout(reqTimeout))

	// Panic → Discord (kalau webhook diset)
	if discordWebhook != "" {
		r.Use(mw.RecoverToDiscord(discordWebhook))
	} else {
		r.Use(chimw.Recoverer)
	}

	// Rate limiter per IP
	ipLimiter := limiter.NewIPLimiter(ratePerMin, time.Minute)
	r.Use(ipLimiter.Middleware)

	// Limit body size
	r.Use(func(next http.Handler) http.Handler { return http.MaxBytesHandler(next, maxBodyBytes) })

	// Auth + route
	r.Group(func(pr chi.Router) {
		pr.Use(mw.APIKeyAuth(apiKeyStore))
		pr.Post("/api/get-balance", api.GetBalanceHandler(solClient, c))
	})

	// Admin test route: force panic
	r.Get("/admin/force-panic", func(w http.ResponseWriter, r *http.Request) {
		panic("forced panic for testing")
	})

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxShutdown)
}
