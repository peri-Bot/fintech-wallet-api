package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	maxRequests = 5
	windowSecs  = 60
)

// RateLimiter returns Gin middleware that enforces a fixed-window rate limit
// of 5 requests per minute per user. It identifies users by the "user_id"
// JSON body field, falling back to the client IP.
func RateLimiter(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// Identify the caller: prefer user_id query param, then IP.
		identifier := c.Query("user_id")
		if identifier == "" {
			identifier = c.ClientIP()
		}

		key := fmt.Sprintf("rate:%s", identifier)

		// INCR the counter; returns 1 on first call within the window.
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "rate limiter unavailable",
			})
			return
		}

		// Set expiry only on the first request of the window.
		if count == 1 {
			rdb.Expire(ctx, key, time.Duration(windowSecs)*time.Second)
		}

		// Attach rate limit headers.
		remaining := int64(maxRequests) - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if count > maxRequests {
			ttl, _ := rdb.TTL(ctx, key).Result()
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": int(ttl.Seconds()),
			})
			return
		}

		c.Next()
	}
}
