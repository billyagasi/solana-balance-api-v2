package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/gagliardetto/solana-go"
	"golang.org/x/sync/errgroup"

	"github.com/billyagasi/solana-balance-api-v2/internal/cache"
	"github.com/billyagasi/solana-balance-api-v2/internal/rpc"
)

type getBalanceReq struct {
	Wallets []string `json:"wallets"`
}

type walletResp struct {
	Wallet  string `json:"wallet"`
	Balance uint64 `json:"balance_lamports"`
	Error   string `json:"error,omitempty"`
}

type getBalanceResp struct {
	Results []walletResp `json:"results"`
	Cached  bool         `json:"cached_any"`
}

func GetBalanceHandler(cli *rpc.Client, c *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req getBalanceReq
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if len(req.Wallets) == 0 {
			http.Error(w, "wallets required", http.StatusBadRequest)
			return
		}
		if len(req.Wallets) > 2000 {
			http.Error(w, "too many wallets (max 2000)", http.StatusBadRequest)
			return
		}

		// Validate shapes early
		uniq := make(map[string]struct{}, len(req.Wallets))
		wallets := make([]string, 0, len(req.Wallets))
		for _, wlt := range req.Wallets {
			wlt = strings.TrimSpace(wlt)
			if wlt == "" {
				continue
			}
			if _, ok := uniq[wlt]; ok {
				continue
			}
			if _, err := solana.PublicKeyFromBase58(wlt); err != nil {
				http.Error(w, "invalid wallet: "+wlt, http.StatusBadRequest)
				return
			}
			uniq[wlt] = struct{}{}
			wallets = append(wallets, wlt)
		}
		if len(wallets) == 0 {
			http.Error(w, "no valid wallets", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		results := make([]walletResp, len(wallets))

		// Concurrency with bound (avoid overwhelming RPC). Limit to 64 in-flight.
		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(64)

		var cachedAny bool
		for i, wlt := range wallets {
			i := i
			wlt := wlt
			g.Go(func() error {
				// Fast path: cache
				if e, ok := c.Get(wlt); ok {
					results[i] = walletResp{Wallet: wlt, Balance: e.Balance}
					cachedAny = true
					return nil
				}
				bal, err := c.Do(wlt, func() (uint64, error) { return cli.GetBalance(ctx, wlt) })
				res := walletResp{Wallet: wlt, Balance: bal}
				if err != nil {
					res.Error = err.Error()
				}
				results[i] = res
				return nil
			})
		}

		_ = g.Wait() // collect all; individual errors recorded per item

		// Keep output stable
		sort.Slice(results, func(i, j int) bool { return results[i].Wallet < results[j].Wallet })

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(getBalanceResp{Results: results, Cached: cachedAny})
	}
}
