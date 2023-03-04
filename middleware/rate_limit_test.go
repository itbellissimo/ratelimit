package middleware

import (
	"fmt"
	"github.com/itbellissimo/ratelimit/pkg/ratelimit"
	"github.com/itbellissimo/ratelimit/pkg/ratelimit/storage"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRateLimit(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {}
	memStorage := storage.NewMemoryCache()
	cfg := getConfig()
	rl := ratelimit.NewRateLimit(&cfg, memStorage)

	mux := http.NewServeMux()
	mux.HandleFunc("/run/http1.1/get", testHandler)
	mux.HandleFunc("/run", testHandler)
	mux.HandleFunc("/reset", testHandler)

	rlm := RateLimit(mux, rl)

	t.Run("middleware /run 123.17.17.11 ID limit: e61f74f3-d46b-4162-a432-5f0447eb1397", func(t *testing.T) {
		method := http.MethodGet
		url := "http://localhost:8087/run"

		req := newRequest(method, url, nil, "123.17.17.11")

		res := httptest.NewRecorder()
		limit := cfg.ByIp.Data[1].Limit
		for i := int64(1); i <= limit+7; i++ {
			rlm.ServeHTTP(res, req)
			if i <= limit && res.Code != http.StatusOK {
				t.Fatalf("Expected %d code response, but got %d", http.StatusOK, res.Code)
			} else if i > limit && res.Code != http.StatusTooManyRequests {
				t.Fatalf("Expected %d code response, but got %d", http.StatusTooManyRequests, res.Code)
			}
		}
	})

	t.Run("middleware /run 123.17.18.11 ID limit: no", func(t *testing.T) {
		method := http.MethodGet
		url := "http://localhost:8087/run"

		req := newRequest(method, url, nil, "123.17.18.11")

		res := httptest.NewRecorder()
		limit := cfg.ByIp.Data[1].Limit
		for i := int64(1); i <= limit+1; i++ {
			rlm.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("Expected %d code response, but got %d", http.StatusOK, res.Code)
			}
		}
	})

	t.Run("middleware /run/http1.1/get 123.17.18.11 ID limit: 87206c45-3098-45c1-86c1-0c28296d163f", func(t *testing.T) {
		method := http.MethodGet
		url := "http://localhost:8087/run/http1.1/get"

		req := newRequest(method, url, nil, "123.45.67.11")

		res := httptest.NewRecorder()
		limit := cfg.ByIp.Data[0].Limit
		for i := int64(1); i <= limit+1; i++ {
			rlm.ServeHTTP(res, req)
			if i <= limit && res.Code != http.StatusOK {
				t.Fatalf("Expected %d code response, but got %d", http.StatusOK, res.Code)
			} else if i > limit && res.Code != http.StatusTooManyRequests {
				t.Fatalf("Expected %d code response, but got %d", http.StatusTooManyRequests, res.Code)
			}
		}
	})
}

func TestRateLimitReset(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {}
	memStorage := storage.NewMemoryCache()
	cfg := getConfig()
	rl := ratelimit.NewRateLimit(&cfg, memStorage)

	mux := http.NewServeMux()
	mux.HandleFunc("/run/http1.1/get", testHandler)
	mux.HandleFunc("/run", testHandler)
	mux.HandleFunc("/reset", testHandler)

	rlm := RateLimit(mux, rl)

	t.Run("middleware /reset nil body 123.17.18.11 ID limit: 87206c45-3098-45c1-86c1-0c28296d163f", func(t *testing.T) {
		method := http.MethodPost
		url := "http://localhost:8087/reset"
		req := newRequest(method, url, nil, "")

		res := httptest.NewRecorder()
		rlm.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("Expected %d code response, but got %d", http.StatusBadRequest, res.Code)
		}
		assert.Contains(t, res.Body.String(), "Wrong params Decode")
	})

	t.Run("middleware /reset wrong body 123.17.18.11 ID limit: 87206c45-3098-45c1-86c1-0c28296d163f", func(t *testing.T) {
		method := http.MethodPost
		url := "http://localhost:8087/reset"
		body := strings.NewReader("{\"NO\": \"1\"}")
		req := newRequest(method, url, body, "")

		res := httptest.NewRecorder()
		rlm.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("Expected %d code response, but got %d", http.StatusBadRequest, res.Code)
		}
		assert.Contains(t, res.Body.String(), "IP param not found")
	})

	t.Run("middleware /reset 123.17.18.11 ID limit: 87206c45-3098-45c1-86c1-0c28296d163f", func(t *testing.T) {
		method := http.MethodGet
		url := "http://localhost:8087/run/http1.1/get"

		req := newRequest(method, url, nil, "123.45.67.11")

		res := httptest.NewRecorder()
		limit := cfg.ByIp.Data[0].Limit
		for i := int64(1); i <= limit+1; i++ {
			rlm.ServeHTTP(res, req)
			if i <= limit && res.Code != http.StatusOK {
				t.Fatalf("Expected %d code response, but got %d", http.StatusOK, res.Code)
			} else if i > limit && res.Code != http.StatusTooManyRequests {
				t.Fatalf("Expected %d code response, but got %d", http.StatusTooManyRequests, res.Code)
			}
		}

		method = http.MethodPost
		url = "http://localhost:8087/reset"

		body := strings.NewReader("{\"ip\": \"123.45.67.11\"}")

		req = newRequest(method, url, body, "123.45.67.11")

		res = httptest.NewRecorder()
		rlm.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("Expected %d code response, but got %d", http.StatusOK, res.Code)
		}

		assert.Contains(
			t, res.Body.String(),
			"\"result\": \"Ok\"",
			fmt.Sprintf("Expected %s response, but got %s", "\"result\": \"Ok\"", res.Body.String()),
		)

		// check again limit after reset
		method = http.MethodGet
		url = "http://localhost:8087/run/http1.1/get"

		req = newRequest(method, url, nil, "123.45.67.11")

		res = httptest.NewRecorder()
		limit = cfg.ByIp.Data[0].Limit
		for i := int64(1); i <= limit+1; i++ {
			rlm.ServeHTTP(res, req)
			if i <= limit && res.Code != http.StatusOK {
				t.Fatalf("Expected %d code response, but got %d", http.StatusOK, res.Code)
			} else if i > limit && res.Code != http.StatusTooManyRequests {
				t.Fatalf("Expected %d code response, but got %d", http.StatusTooManyRequests, res.Code)
			}
		}
	})
}

func getConfig() ratelimit.Config {
	return ratelimit.Config{
		Title: "RateLimit test rules",
		ByIp: ratelimit.ByIp{
			ExcludeIps: []string{},
			Data: []ratelimit.ByIpData{
				{
					ID: "87206c45-3098-45c1-86c1-0c28296d163f",
					Handlers: []ratelimit.LimitHandler{{
						Protocol:       "http/1.1",
						ProtocolRegexp: false,
						Method:         "GET",
						Url:            "/run/http1.1/get",
						Regexp:         false,
					}},
					Limit:      3,
					BlockTime:  10,
					Mask:       "123.45.67.0/24",
					ExcludeIps: nil,
				},
				{
					ID: "e61f74f3-d46b-4162-a432-5f0447eb1397",
					Handlers: []ratelimit.LimitHandler{{
						Protocol:       ".*",
						ProtocolRegexp: true,
						Method:         "GET",
						Url:            "/run.*",
						Regexp:         true,
					}},
					Limit:      3,
					BlockTime:  10,
					Mask:       "123.17.17.0/24",
					ExcludeIps: nil,
				},
			},
		},
	}
}

func newRequest(method string, url string, body io.Reader, forwardedIP string) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Header.Add("X-Forwarded-For", forwardedIP)

	return req
}

/*

func (mr MyReader) Read(b []byte) (int, error) {
 for {
  b
 }
}



func (r13 rot13Reader) Read(b []byte) (int, error) {
 nb := make([]byte, 13)
 for {
  _, err := r13.r.Read(nb)
  if err == io.EOF {
   return len(b), io.EOF
  }

  for i:=0; i<=len(b)/2; i++ {
   nb[i], nb[len(b) - i] = nb[len(b) - i], nb[i]
  }
 }

 return len(b),  nil
}
*/
