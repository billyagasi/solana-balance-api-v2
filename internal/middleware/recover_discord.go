package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"
)

type discordPayload struct {
	Content string `json:"content"`
}

func RecoverToDiscord(webhook string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := string(debug.Stack())
					msg := fmt.Sprintf(
						"PANIC: %v\nURL: %s %s\nTime: %s\nStack:\n%s",
						rec, r.Method, r.URL.String(), time.Now().Format(time.RFC3339), stack,
					)
					_ = postDiscord(webhook, msg)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func postDiscord(webhook, msg string) error {
	body, _ := json.Marshal(discordPayload{
		Content: "```\n" + trimTo(msg, 1800) + "\n```",
	})
	req, _ := http.NewRequest("POST", webhook, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{Timeout: 5 * time.Second}
	_, err := cli.Do(req)
	return err
}

func trimTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
