package middleware

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/config"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const maxBuckets = 50_000

type bucket struct {
	tokens float64
	at     time.Time
}

type limiter struct {
	mu      sync.Mutex
	rate    config.Rate
	buckets map[string]bucket
}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.buckets) >= maxBuckets {
		l.buckets = map[string]bucket{}
	}
	b, seen := l.buckets[key]
	if !seen {
		b = bucket{tokens: l.rate.Burst, at: now}
	}
	b.tokens = min(l.rate.Burst, b.tokens+now.Sub(b.at).Minutes()*l.rate.PerMinute)
	b.at = now
	allowed := b.tokens >= 1
	if allowed {
		b.tokens--
	}
	l.buckets[key] = b
	return allowed
}

func RateLimit(rate config.Rate, key func(*gin.Context) string) gin.HandlerFunc {
	l := &limiter{rate: rate, buckets: map[string]bucket{}}
	return func(c *gin.Context) {
		if l.allow(key(c), time.Now()) {
			c.Next()
			return
		}
		slog.WarnContext(c.Request.Context(), "rate limit reached", "route", c.FullPath())
		response.Fail(c, apperr.New(apperr.RateLimited, "Too many requests. Slow down a little."))
	}
}
