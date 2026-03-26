package common

import "testing"

func TestVerifyAndDeleteCodeWithKeyConsumesCode(t *testing.T) {
	originalRedisEnabled := RedisEnabled
	originalVerificationMinutes := VerificationValidMinutes
	RedisEnabled = false
	VerificationValidMinutes = 10
	defer func() {
		RedisEnabled = originalRedisEnabled
		VerificationValidMinutes = originalVerificationMinutes
		DeleteKey("user@example.com", EmailVerificationPurpose)
	}()

	RegisterVerificationCodeWithKey("user@example.com", "123456", EmailVerificationPurpose)

	if !VerifyCodeWithKey("user@example.com", "123456", EmailVerificationPurpose) {
		t.Fatal("expected code to verify before consumption")
	}
	if !VerifyAndDeleteCodeWithKey("user@example.com", "123456", EmailVerificationPurpose) {
		t.Fatal("expected code to be consumed successfully")
	}
	if VerifyCodeWithKey("user@example.com", "123456", EmailVerificationPurpose) {
		t.Fatal("expected consumed code to become invalid")
	}
	if VerifyAndDeleteCodeWithKey("user@example.com", "123456", EmailVerificationPurpose) {
		t.Fatal("expected consumed code to be single-use")
	}
}
