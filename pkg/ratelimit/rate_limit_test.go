package ratelimit

import (
	"context"
	"fmt"
	"github.com/itbellissimo/ratelimit/pkg/ratelimit/storage"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// TestNewMemoryCache test NewMemoryCache function
func TestNewRateLimit(t *testing.T) {
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)

	assert.Equal(t, fmt.Sprintf("%T", &rateLimit{}), fmt.Sprintf("%T", rl))
	assert.Equal(t, "RateLimit test rules", rl.config.Title)
	assert.NotNil(t, rl)
	assert.NotNil(t, rl.config)
	assert.NotNil(t, rl.storage)
}

// TestGetConfig test GetConfig function
func TestGetConfig(t *testing.T) {
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)

	assert.Equal(t, &cfg, rl.GetConfig())
}

// TestIdsByIP test IdsByIP function
func TestIdsByIP(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)

	type testIDByIP struct {
		IP       string
		protocol string
		method   string
		url      string
		IDs      []string
	}

	testIDByIPs := []testIDByIP{
		{
			IP:       "123.45.67.1",
			protocol: "http/1.1",
			method:   "GET",
			url:      "/run/http1.1/get",
			IDs:      []string{"87206c45-3098-45c1-86c1-0c28296d163f"},
		},
		{
			IP:       "123.45.67.1",
			protocol: "http/1.1",
			method:   "GET",
			url:      "/run/http1.1/get",
			IDs:      []string{"87206c45-3098-45c1-86c1-0c28296d163f"},
		},
		{
			IP:       "123.17.17.8",
			protocol: "",
			method:   "http/2",
			url:      "/run/http/get",
			IDs:      []string{},
		},
		{
			IP:       "123.17.18.8",
			protocol: "",
			method:   "http/2",
			url:      "/run/any/any",
			IDs:      []string{"1f09b207-3f0c-4bd7-ae74-b602e049ae5d", "d21e62e9-4c9a-49b2-a7be-7a3851219f8b"},
		},
		{
			IP:       "123.17.18.8",
			protocol: "",
			method:   "",
			url:      "*",
			IDs:      []string{"1f09b207-3f0c-4bd7-ae74-b602e049ae5d", "d21e62e9-4c9a-49b2-a7be-7a3851219f8b"},
		},
		{
			IP:       "123.45.67.7",
			protocol: "http/1.1",
			method:   "GET",
			url:      "*",
			IDs:      []string{"87206c45-3098-45c1-86c1-0c28296d163f"},
		},
	}

	for _, testIDByIP := range testIDByIPs {
		ids := rl.IdsByIP(ctx, testIDByIP.protocol, testIDByIP.method, testIDByIP.url, testIDByIP.IP)
		assert.Len(t, ids, len(testIDByIP.IDs))
		assert.ElementsMatchf(t, testIDByIP.IDs, ids, "IDS different ")
	}
}

// TestIncByIDs test IncByIDs function
func TestIncByIDs(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)

	cacheKey := cfg.ByIp.Data[0].ID
	limit := cfg.ByIp.Data[0].Limit
	blockTime := uint64(cfg.ByIp.Data[0].BlockTime)
	xIP := "123.45.67.1"
	xIP2 := "123.45.68.1"

	xIDs := rl.IdsByIP(ctx, "http/1.1", "GET", "/run/http1.1/get", xIP)
	xIDs2 := rl.IdsByIP(ctx, "http/1.1", "GET", "/run/http1.1/get", xIP2)

	for i := int64(1); i <= limit+1; i++ {
		err := rl.storage.Set(ctx, []byte(cacheKey), []byte(strconv.Itoa((int)(i))), &blockTime)
		if err != nil {
			assert.Error(t, err)
		}

		if i < limit {
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit not reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		} else {
			assert.True(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		}
	}
}

// TestIsLimitedByIP test IsLimitedByIP function
func TestIsLimitedByIP(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)

	cacheKey := cfg.ByIp.Data[0].ID
	limit := cfg.ByIp.Data[0].Limit
	blockTime := uint64(cfg.ByIp.Data[0].BlockTime)
	xIP := "123.45.67.1"
	xIP2 := "123.45.68.1"

	xIDs := rl.IdsByIP(ctx, "http/1.1", "GET", "/run/http1.1/get", xIP)
	xIDs2 := rl.IdsByIP(ctx, "http/1.1", "GET", "/run/http1.1/get", xIP2)

	for i := int64(1); i <= limit+1; i++ {
		err := rl.storage.Set(ctx, []byte(cacheKey), []byte(strconv.Itoa((int)(i))), &blockTime)
		if err != nil {
			assert.Error(t, err)
		}

		if i < limit {
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit not reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		} else {
			assert.True(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		}
	}
}

// TestClearByIDs test ClearByIDs function
func TestClearByIDs(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)

	cacheKey := cfg.ByIp.Data[0].ID
	limit := cfg.ByIp.Data[0].Limit
	blockTime := uint64(cfg.ByIp.Data[0].BlockTime)
	xIP := "123.45.67.1"
	xIP2 := "123.45.68.1"

	xIDs := rl.IdsByIP(ctx, "http/1.1", "GET", "/run/http1.1/get", xIP)
	xIDs2 := rl.IdsByIP(ctx, "http/1.1", "GET", "/run/http1.1/get", xIP2)

	for i := int64(1); i <= limit+1; i++ {
		err := rl.storage.Set(ctx, []byte(cacheKey), []byte(strconv.Itoa((int)(i))), &blockTime)
		if err != nil {
			assert.Error(t, err)
		}

		if i < limit {
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit not reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		} else {
			assert.True(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		}
	}

	// clear
	err := rl.storage.Del(ctx, []byte(cacheKey))
	if err != nil {
		assert.Error(t, err)
	}

	// check limits again
	for i := int64(1); i <= limit+1; i++ {
		err := rl.storage.Set(ctx, []byte(cacheKey), []byte(strconv.Itoa((int)(i))), &blockTime)
		if err != nil {
			assert.Error(t, err)
		}

		if i < limit {
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit not reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		} else {
			assert.True(
				t,
				rl.IsLimitedByIDs(ctx, xIDs),
				"Limit reached. IP in range.",
			)
			assert.False(
				t,
				rl.IsLimitedByIDs(ctx, xIDs2),
				"Limit not reached. IP not in range.",
			)
		}
	}
}

// Test_IsLimited test IsLimited function. Tmp.
func Test_IsLimited(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)
	req := httptest.NewRequest(http.MethodGet, "/someurl", nil)

	isLimit := rl.IsLimited(ctx, req)
	assert.False(t, isLimit)
}

// Test_IsLimitedByApp test IsLimitedByApp function. Tmp.
func Test_IsLimitedByApp(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)
	isLimitedByApp := rl.IsLimitedByApp(ctx, "https", http.MethodGet, "/someurl", map[string][]string{"test": {"undefined"}}, "test")
	assert.False(t, isLimitedByApp)
}

// Test_ClearByIDs test ClearByIDs function. Tmp.
func Test_ClearByIDs(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)
	err := rl.ClearByIDs(ctx, []string{})
	assert.Nil(t, err)

	err = rl.ClearByIDs(ctx, []string{"123", "undefined"})
	assert.Nil(t, err)

	rl.IncByIDs(ctx, []string{"123", "undo"})
	err = rl.ClearByIDs(ctx, []string{"123", "undefined"})
	assert.Nil(t, err)
}

// Test_IncByIDs test IncByIDs function. Tmp.
func Test_IncByIDs(t *testing.T) {
	ctx := context.Background()
	cfg := TmpConfig()
	memStorage := storage.NewMemoryCache()
	rl := NewRateLimit(&cfg, memStorage)
	counter := rl.IncByIDs(ctx, []string{})
	assert.Equal(t, int64(0), counter)

	counter = rl.IncByIDs(ctx, []string{"123", "undefined"})
	assert.Equal(t, int64(0), counter)

	rl.IncByIDs(ctx, []string{"123", "undo"})
	counter = rl.IncByIDs(ctx, []string{"123", "undefined"})
	assert.Equal(t, int64(0), counter)
}

// TmpConfig return fixed Config
func TmpConfig() Config {
	return Config{
		Title: "RateLimit test rules",
		ByIp: ByIp{
			ExcludeIps: []string{},
			Data: []ByIpData{
				{
					ID: "87206c45-3098-45c1-86c1-0c28296d163f",
					Handlers: []LimitHandler{{
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
					Handlers: []LimitHandler{{
						Protocol:       "http/.*",
						ProtocolRegexp: true,
						Method:         "GET",
						Url:            "/run/http/get",
						Regexp:         false,
					}},
					Limit:      3,
					BlockTime:  10,
					Mask:       "123.17.17.0/24",
					ExcludeIps: nil,
				},
				{
					ID: "1f09b207-3f0c-4bd7-ae74-b602e049ae5d",
					Handlers: []LimitHandler{{
						Protocol:       ".*",
						ProtocolRegexp: true,
						Method:         "",
						Url:            "/run/any/any",
						Regexp:         false,
					}},
					Limit:      3,
					BlockTime:  10,
					Mask:       "123.17.18.0/24",
					ExcludeIps: nil,
				},
				{
					ID: "d21e62e9-4c9a-49b2-a7be-7a3851219f8b",
					Handlers: []LimitHandler{{
						Protocol:       ".*",
						ProtocolRegexp: true,
						Method:         "",
						Url:            ".*",
						Regexp:         true,
					}},
					Limit:      3,
					BlockTime:  10,
					Mask:       "123.17.18.0/24",
					ExcludeIps: nil,
				},
			},
		},
	}
}
