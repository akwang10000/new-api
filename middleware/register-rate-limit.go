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
	RegisterAttemptRateLimitMark             = "RG"
	RegisterIdentityAttemptMaxRequests       = 3
	RegisterIPAttemptMaxRequests             = 10
	RegisterAttemptDuration            int64 = 20 * 60
	RegisterCreateRateLimitMark              = "RC"
	RegisterIPCreateMaxRequests              = 3
	RegisterCreateDuration             int64 = 20 * 60
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

func registerRateLimitMemoryHit(key string, maxRequests int, duration int64) bool {
	return !inMemoryRateLimiter.Request(key, maxRequests, duration)
}

func registerRateLimitRedisHit(ctx context.Context, key string, maxRequests int, duration int64) (bool, error) {
	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := common.RDB.Expire(ctx, key, time.Duration(duration)*time.Second).Err(); err != nil {
			return false, err
		}
	}
	return count > int64(maxRequests), nil
}

func checkRegisterLimit(c *gin.Context, keys map[string]int, duration int64) bool {
	if len(keys) == 0 {
		return true
	}

	if common.RedisEnabled {
		ctx := context.Background()
		for key, maxRequests := range keys {
			limited, err := registerRateLimitRedisHit(ctx, key, maxRequests, duration)
			if err != nil {
				common.SysError("register rate limit redis error: " + err.Error())
				inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
				for fallbackKey, fallbackMaxRequests := range keys {
					if registerRateLimitMemoryHit(fallbackKey, fallbackMaxRequests, duration) {
						registerRateLimitResponse(c)
						return false
					}
				}
				return true
			}
			if limited {
				registerRateLimitResponse(c)
				return false
			}
		}
		return true
	}

	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	for key, maxRequests := range keys {
		if registerRateLimitMemoryHit(key, maxRequests, duration) {
			registerRateLimitResponse(c)
			return false
		}
	}
	return true
}

func RegisterRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		keys := map[string]int{
			fmt.Sprintf("register:%s:ip:%s", RegisterAttemptRateLimitMark, c.ClientIP()): RegisterIPAttemptMaxRequests,
		}

		payload, err := registerRateLimitPayloadFromContext(c)
		if err != nil {
			common.SysError("register rate limit payload parse error: " + err.Error())
		} else {
			if payload.Username != "" {
				keys[fmt.Sprintf("register:%s:username:%s", RegisterAttemptRateLimitMark, payload.Username)] = RegisterIdentityAttemptMaxRequests
			}
			if payload.Email != "" {
				keys[fmt.Sprintf("register:%s:email:%s", RegisterAttemptRateLimitMark, payload.Email)] = RegisterIdentityAttemptMaxRequests
			}
		}

		if !checkRegisterLimit(c, keys, RegisterAttemptDuration) {
			return
		}
		c.Next()
	}
}

func CheckRegisterCreateRateLimit(c *gin.Context) bool {
	keys := map[string]int{
		fmt.Sprintf("register:%s:ip:%s", RegisterCreateRateLimitMark, c.ClientIP()): RegisterIPCreateMaxRequests,
	}
	return checkRegisterLimit(c, keys, RegisterCreateDuration)
}
