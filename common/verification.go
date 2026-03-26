package common

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type verificationValue struct {
	code string
	time time.Time
}

const (
	EmailVerificationPurpose = "v"
	PasswordResetPurpose     = "r"
)

var verificationMutex sync.Mutex
var verificationMap map[string]verificationValue
var verificationMapMaxSize = 10
var VerificationValidMinutes = 10

func verificationStorageKey(key string, purpose string) string {
	return "verification:" + purpose + ":" + key
}

func GenerateVerificationCode(length int) string {
	code := uuid.New().String()
	code = strings.Replace(code, "-", "", -1)
	if length == 0 {
		return code
	}
	return code[:length]
}

func RegisterVerificationCodeWithKey(key string, code string, purpose string) {
	if RedisEnabled && RDB != nil {
		ctx := context.Background()
		err := RDB.Set(ctx, verificationStorageKey(key, purpose), code, time.Duration(VerificationValidMinutes)*time.Minute).Err()
		if err == nil {
			return
		}
		SysError("failed to store verification code in redis: " + err.Error())
	}
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap[purpose+key] = verificationValue{
		code: code,
		time: time.Now(),
	}
	if len(verificationMap) > verificationMapMaxSize {
		removeExpiredPairs()
	}
}

func VerifyCodeWithKey(key string, code string, purpose string) bool {
	if RedisEnabled && RDB != nil {
		ctx := context.Background()
		value, err := RDB.Get(ctx, verificationStorageKey(key, purpose)).Result()
		if err == nil {
			return code == value
		}
		if err != nil && err != redis.Nil {
			SysError("failed to verify verification code in redis: " + err.Error())
		}
	}
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	value, okay := verificationMap[purpose+key]
	now := time.Now()
	if !okay || int(now.Sub(value.time).Seconds()) >= VerificationValidMinutes*60 {
		return false
	}
	return code == value.code
}

func VerifyAndDeleteCodeWithKey(key string, code string, purpose string) bool {
	if RedisEnabled && RDB != nil {
		ctx := context.Background()
		storageKey := verificationStorageKey(key, purpose)
		script := redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return 0
end
if value == ARGV[1] then
	redis.call("DEL", KEYS[1])
	return 1
end
return 0
`)
		result, err := script.Run(ctx, RDB, []string{storageKey}, code).Int()
		if err == nil {
			return result == 1
		}
		SysError("failed to consume verification code in redis: " + err.Error())
	}

	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	value, okay := verificationMap[purpose+key]
	now := time.Now()
	if !okay || int(now.Sub(value.time).Seconds()) >= VerificationValidMinutes*60 {
		delete(verificationMap, purpose+key)
		return false
	}
	if code != value.code {
		return false
	}
	delete(verificationMap, purpose+key)
	return true
}

func DeleteKey(key string, purpose string) {
	if RedisEnabled && RDB != nil {
		ctx := context.Background()
		err := RDB.Del(ctx, verificationStorageKey(key, purpose)).Err()
		if err == nil {
			return
		}
		SysError("failed to delete verification code in redis: " + err.Error())
	}
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	delete(verificationMap, purpose+key)
}

// no lock inside, so the caller must lock the verificationMap before calling!
func removeExpiredPairs() {
	now := time.Now()
	for key := range verificationMap {
		if int(now.Sub(verificationMap[key].time).Seconds()) >= VerificationValidMinutes*60 {
			delete(verificationMap, key)
		}
	}
}

func init() {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap = make(map[string]verificationValue)
}
