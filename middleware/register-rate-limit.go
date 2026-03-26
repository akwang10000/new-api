package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

const (
	RegisterAttemptRateLimitMark       = "RG"
	RegisterAttemptMaxRequests         = 3
	RegisterAttemptDuration      int64 = 20 * 60
)

type registerRateLimitPayload struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func registerRateLimitResponse(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": "Too many registration attempts. Please try again later.",
	})
	c.Abort()
}

func registerRateLimitPayloadFromContext(c *gin.Context) (*registerRateLimitPayload, error) {
	var payload registerRateLimitPayload
	if err := common.UnmarshalBodyReusable(c, &payload); err != nil {
		return nil, err
	}
	payload.Username = strings.TrimSpace(strings.ToLower(payload.Username))
	payload.Email = strings.TrimSpace(strings.ToLower(payload.Email))
	return &payload, nil
}

func registerRateLimitMemoryHit(key string) bool {
	return !inMemoryRateLimiter.Request(key, RegisterAttemptMaxRequests, RegisterAttemptDuration)
}

func registerRateLimitRedisHit(ctx context.Context, key string) (bool, error) {
	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := common.RDB.Expire(ctx, key, time.Duration(RegisterAttemptDuration)*time.Second).Err(); err != nil {
			return false, err
		}
	}
	return count > int64(RegisterAttemptMaxRequests), nil
}

func RegisterRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := registerRateLimitPayloadFromContext(c)
		if err != nil {
			c.Next()
			return
		}

		keys := make([]string, 0, 2)
		if payload.Username != "" {
			keys = append(keys, fmt.Sprintf("register:%s:username:%s", RegisterAttemptRateLimitMark, payload.Username))
		}
		if payload.Email != "" {
			keys = append(keys, fmt.Sprintf("register:%s:email:%s", RegisterAttemptRateLimitMark, payload.Email))
		}
		if len(keys) == 0 {
			c.Next()
			return
		}

		if common.RedisEnabled {
			ctx := context.Background()
			for _, key := range keys {
				limited, err := registerRateLimitRedisHit(ctx, key)
				if err != nil {
					common.SysError("register rate limit redis error: " + err.Error())
					inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
					for _, fallbackKey := range keys {
						if registerRateLimitMemoryHit(fallbackKey) {
							registerRateLimitResponse(c)
							return
						}
					}
					c.Next()
					return
				}
				if limited {
					registerRateLimitResponse(c)
					return
				}
			}
			c.Next()
			return
		}

		inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
		for _, key := range keys {
			if registerRateLimitMemoryHit(key) {
				registerRateLimitResponse(c)
				return
			}
		}
		c.Next()
	}
}
