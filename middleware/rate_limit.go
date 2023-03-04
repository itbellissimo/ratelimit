package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func RateLimit(next http.Handler, rl RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.URL.Path == "/reset" && r.Method == http.MethodPost {
			// reset DTO
			type ResetRequest struct {
				IP string `json:"ip"`
			}

			// decode input or return error
			var input ResetRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, err = fmt.Fprintf(w, "Wrong params Decode. "+err.Error())
				if err != nil {
					return
				}
				return
			}

			if input.IP == "" {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Wrong params. IP param not found.")
				return
			}

			ids := rl.IdsByIP(ctx, "*", "*", "*", input.IP)
			if len(ids) > 0 {
				if err := rl.ClearByIDs(ctx, ids); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Error clean limits by IP")
					return
				}
			}

			fmt.Fprintf(w, "{\"result\": \"Ok\"}\n")
			w.WriteHeader(http.StatusOK)
			return
		}

		ips := r.Header.Get("X-FORWARDED-FOR")
		splitIPs := strings.Split(ips, ",")

		if len(splitIPs) > 0 {
			realIP := splitIPs[0]
			ids := rl.IdsByIP(ctx, r.Proto, r.Method, r.URL.Path, realIP)
			if len(ids) > 0 {
				ids = ids[:1]
				if rl.IsLimitedByIDs(ctx, ids) {
					w.WriteHeader(http.StatusTooManyRequests)
					_, err := w.Write([]byte("Too many requests"))
					if err != nil {
						log.Printf("%s %s %s", r.Method, r.RequestURI, err.Error())
					}
				}

				rl.IncByIDs(ctx, ids)
			}
		}

		appName := r.Header.Get("X-APP")
		if rl.IsLimitedByApp(ctx, r.Proto, r.Method, r.URL.RawPath, r.URL.Query(), appName) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, err := w.Write([]byte("Too many requests by app"))
			if err != nil {
				log.Printf("%s %s %s", r.Method, r.RequestURI, err.Error())
			}
		}

		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

type RateLimiter interface {
	IsLimited(ctx context.Context, req *http.Request) bool
	IsLimitedByIDs(ctx context.Context, ids []string) bool
	IncByIDs(ctx context.Context, ids []string) int64
	IdsByIP(ctx context.Context, protocol, method, url string, strIP string) []string
	ClearByIDs(ctx context.Context, ids []string) error
	IsLimitedByApp(ctx context.Context, protocol, method, url string, query map[string][]string, appName string) bool
}

//
//// RateLimit is a middleware handler that limits count request by IP and sub mask
//type RateLimit struct {
//	handler http.Handler
//	rl      RateLimiter
//}
//
//// ServeHTTP handles the request by passing it to the real
//func (rl *RateLimit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	start := time.Now()
//	rl.handler.ServeHTTP(w, r)
//	log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
//}
//
//// NewRateLimiter constructs a new NewRateLimiter middleware handler
//func NewRateLimiter(h http.Handler) *RateLimit {
//	return &RateLimit{handler: h}
//}
