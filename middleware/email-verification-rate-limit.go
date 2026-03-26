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
	EmailVerificationRateLimitMark       = "EV"
	EmailVerificationMaxRequests         = 2
	EmailVerificationDuration      int64 = 30
)

func normalizedEmailVerificationKey(c *gin.Context) string {
	email := strings.TrimSpace(strings.ToLower(c.Query("email")))
	if email == "" {
		return ""
	}
	return "emailVerification:" + EmailVerificationRateLimitMark + ":email:" + email
}

func emailVerificationIPKey(c *gin.Context) string {
	return "emailVerification:" + EmailVerificationRateLimitMark + ":ip:" + c.ClientIP()
}

func redisEmailVerificationRateLimitHit(ctx context.Context, key string) (bool, int64, error) {
	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if count == 1 {
		if err := common.RDB.Expire(ctx, key, time.Duration(EmailVerificationDuration)*time.Second).Err(); err != nil {
			return false, 0, err
		}
	}
	if count <= int64(EmailVerificationMaxRequests) {
		return false, 0, nil
	}

	ttl, err := common.RDB.TTL(ctx, key).Result()
	waitSeconds := EmailVerificationDuration
	if err == nil && ttl > 0 {
		waitSeconds = int64(ttl.Seconds())
	}
	return true, waitSeconds, nil
}

func memoryEmailVerificationRateLimitHit(key string) bool {
	return !inMemoryRateLimiter.Request(key, EmailVerificationMaxRequests, EmailVerificationDuration)
}

func abortEmailVerificationRateLimit(c *gin.Context, waitSeconds int64) {
	message := "Too many verification code requests. Please try again later."
	if waitSeconds > 0 {
		message = fmt.Sprintf("Too many verification code requests. Please retry in %d seconds.", waitSeconds)
	}
	c.JSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": message,
	})
	c.Abort()
}

func redisEmailVerificationRateLimiter(c *gin.Context) {
	ctx := context.Background()

	ipLimited, waitSeconds, err := redisEmailVerificationRateLimitHit(ctx, emailVerificationIPKey(c))
	if err != nil {
		memoryEmailVerificationRateLimiter(c)
		return
	}
	if ipLimited {
		abortEmailVerificationRateLimit(c, waitSeconds)
		return
	}

	emailKey := normalizedEmailVerificationKey(c)
	if emailKey == "" {
		c.Next()
		return
	}

	emailLimited, waitSeconds, err := redisEmailVerificationRateLimitHit(ctx, emailKey)
	if err != nil {
		memoryEmailVerificationRateLimiter(c)
		return
	}
	if emailLimited {
		abortEmailVerificationRateLimit(c, waitSeconds)
		return
	}

	c.Next()
}

func memoryEmailVerificationRateLimiter(c *gin.Context) {
	if memoryEmailVerificationRateLimitHit(EmailVerificationRateLimitMark + ":ip:" + c.ClientIP()) {
		abortEmailVerificationRateLimit(c, 0)
		return
	}

	emailKey := normalizedEmailVerificationKey(c)
	if emailKey != "" && memoryEmailVerificationRateLimitHit(emailKey) {
		abortEmailVerificationRateLimit(c, 0)
		return
	}

	c.Next()
}

func EmailVerificationRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.RedisEnabled {
			redisEmailVerificationRateLimiter(c)
		} else {
			inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
			memoryEmailVerificationRateLimiter(c)
		}
	}
}
