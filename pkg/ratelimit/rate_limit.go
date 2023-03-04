package ratelimit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Storager interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Set(ctx context.Context, key []byte, value []byte, ttl *uint64) error
	Del(ctx context.Context, list ...[]byte) error
	Has(ctx context.Context, key []byte) bool
	Inc(ctx context.Context, key []byte, ttl *uint64) (int64, error)
	Decr(ctx context.Context, key []byte, ttl *uint64) (int64, error)
	Clear(ctx context.Context) error
}

type LimitHandler struct {
	ID             string `mapstructure:"id"`
	Protocol       string
	ProtocolRegexp bool `mapstructure:"protocol_regexp"`
	Method         string
	Url            string
	Regexp         bool
}

type ByIpData struct {
	ID         string         `mapstructure:"id"`
	Handlers   []LimitHandler `mapstructure:"handlers"`
	Limit      int64
	BlockTime  int64 `mapstructure:"block_time"`
	Mask       string
	ExcludeIps []string `mapstructure:"exclude_ips"`
}

type ByIp struct {
	ExcludeIps []string   `mapstructure:"exclude_ips"`
	Data       []ByIpData `mapstructure:"data"`
}

type Config struct {
	Title string `mapstructure:"title"`
	ByIp  ByIp   `mapstructure:"by_ip"`
}

// rateLimit
type rateLimit struct {
	config  *Config
	storage Storager
}

// go:cover ignore
func (rl *rateLimit) IsLimited(ctx context.Context, req *http.Request) bool {
	// go:cover ignore
	return false
}

func NewRateLimit(cfg *Config, storage Storager) *rateLimit {
	return &rateLimit{
		config:  cfg,
		storage: storage,
	}
}

func (rl *rateLimit) GetConfig() *Config {
	return rl.config
}

func (rl *rateLimit) UpdateConfig(cfg *Config) error {
	rl.config = cfg
	return nil
}

func (rl *rateLimit) IncByIDs(ctx context.Context, ids []string) int64 {
	if len(ids) == 0 {
		return 0
	}

	for _, storeID := range ids {
		for _, byIpData := range rl.config.ByIp.Data {
			ttl := uint64(byIpData.BlockTime)
			if byIpData.ID != storeID {
				continue
			}

			counter, err := rl.storage.Inc(ctx, []byte(storeID), &ttl)
			if err != nil {
				continue
			}

			return counter
		}
	}

	return 0
}

// IsLimitedByIDs check is rate limited by limit IDS
func (rl *rateLimit) IsLimitedByIDs(ctx context.Context, ids []string) bool {
	if len(ids) == 0 {
		return false
	}

	for _, storeID := range ids {
		for _, byIpData := range rl.config.ByIp.Data {
			if byIpData.ID != storeID {
				continue
			}

			c, err := rl.storage.Get(ctx, []byte(storeID))
			if err != nil {
				continue
			}

			counter, err := strconv.ParseInt(string(c), 10, 64)
			if err != nil {
				continue
			}

			if counter >= byIpData.Limit {
				return true
			}
		}
	}

	return false
}

// IdsByIP get date limit IDs
func (rl *rateLimit) IdsByIP(
	ctx context.Context,
	protocol, method, url string,
	strIP string,
) []string {
	protocol = strings.ToLower(protocol)
	method = strings.ToLower(method)
	url = strings.ToLower(url)
	ip := net.ParseIP(strIP)

	res := make([]string, 0)
	for _, byIpData := range rl.config.ByIp.Data {
		var err error

		//@TODO: check this
		if byIpData.Mask == "" {
			//continue
		} else {
			_, IPNet, err := net.ParseCIDR(byIpData.Mask)
			if err != nil {
				fmt.Sprintf("%v ====> %T", err, err)
			}

			if IPNet != nil && !IPNet.Contains(ip) {
				continue
			}
		}

		storeID := byIpData.ID
		if protocol == "*" && method == "*" && url == "*" {
			res = append(res, storeID)
			continue
		}
		//ttl := uint64(byIpData.BlockTime)
		for _, lh := range byIpData.Handlers {
			var reg *regexp.Regexp
			if lh.Regexp {
				reg, err = regexp.Compile(lh.Url)
				if err != nil {
					continue
				}
			}

			var protocolRegexp *regexp.Regexp
			if lh.ProtocolRegexp {
				protocolRegexp, err = regexp.Compile(lh.Protocol)
				if err != nil {
					continue
				}
			}

			if lh.Url == "" {
				continue
			}

			if ((url == "*") || (!lh.Regexp && strings.ToLower(lh.Url) == url) || (lh.Regexp && reg.MatchString(url))) &&
				(lh.Method == "" || (strings.ToLower(lh.Method) == method)) &&
				(lh.Protocol == "" ||
					(!lh.ProtocolRegexp && strings.ToLower(lh.Protocol) == protocol) ||
					(lh.ProtocolRegexp && protocolRegexp.MatchString(protocol))) {

				res = append(res, storeID)
			}
		}
	}
	return res
}

func (rl *rateLimit) IsLimitedByApp(ctx context.Context, protocol, method, url string, query map[string][]string, appName string) bool {
	return false
}

func (rl *rateLimit) ClearByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	for _, storeID := range ids {
		for _, byIpData := range rl.config.ByIp.Data {
			if byIpData.ID != storeID {
				continue
			}

			err := rl.storage.Del(ctx, []byte(storeID))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
