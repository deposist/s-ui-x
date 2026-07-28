package subpage

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

// subRateLimit is the per-IP rate limiter used by the cabinet landing page.
// It mirrors the conservative behaviour of sub/rate_limit.go without
// depending on its unexported helpers, so the upstream package can refactor
// freely without breaking this page.
//
// Tunable live via the existing setting `subRateLimitPerIP`. Defaults to
// 30 requests/minute/IP if the setting is absent or invalid.
var subRateLimit gin.HandlerFunc = rateLimitMiddleware()

type rlBucket struct {
	count   int
	resetAt time.Time
}

var (
	rlMu      sync.Mutex
	rlByIP    = map[string]*rlBucket{}
	rlMaxKeys = 4096
)

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := (&service.SettingService{}).GetSubRateLimitPerIP()
		if err != nil || limit <= 0 {
			limit = 30
		}
		ip := c.ClientIP()
		if ip == "" {
			c.Next()
			return
		}

		rlMu.Lock()
		b, ok := rlByIP[ip]
		now := time.Now()
		if !ok || now.After(b.resetAt) {
			b = &rlBucket{resetAt: now.Add(time.Minute)}
			rlByIP[ip] = b
			if len(rlByIP) > rlMaxKeys {
				// Bound map growth under scanner pressure; the counter is
				// best-effort detection, not a security primitive.
				rlByIP = map[string]*rlBucket{}
			}
		}
		b.count++
		count := b.count
		limitCopy := limit
		rlMu.Unlock()

		c.Header("X-RateLimit-Limit", itoa(limitCopy))
		c.Header("X-RateLimit-Remaining", itoa(max0(limitCopy-count)))
		if count > limitCopy {
			logger.Warningf("subpage: rate limit hit for ip=%s count=%d limit=%d", ip, count, limitCopy)
			c.AbortWithStatus(429)
			return
		}
		c.Next()
	}
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	return formatInt(n)
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// formatInt avoids pulling strconv just for two call sites; keeps the
// dependency graph clean and avoids surprises if strconv is later moved.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}