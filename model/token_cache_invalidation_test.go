package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestInvalidateUserTokensCacheRejectsInvalidUserId(t *testing.T) {
	previousRedisEnabled := common.RedisEnabled
	common.RedisEnabled = true
	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
	})

	err := InvalidateUserTokensCache(0)
	require.Error(t, err)
}

func TestValidateUserTokenReturnsSentinelErrorsWithoutLeakingTokenState(t *testing.T) {
	truncateTables(t)
	initCol()

	now := common.GetTimestamp()
	cases := []struct {
		name  string
		token Token
	}{
		{
			name: "exhausted status",
			token: Token{
				UserId:         1,
				Key:            "exhausted-token-key",
				Status:         common.TokenStatusExhausted,
				ExpiredTime:    -1,
				RemainQuota:    100,
				UnlimitedQuota: false,
			},
		},
		{
			name: "expired status",
			token: Token{
				UserId:         1,
				Key:            "expired-token-key",
				Status:         common.TokenStatusExpired,
				ExpiredTime:    -1,
				RemainQuota:    100,
				UnlimitedQuota: false,
			},
		},
		{
			name: "disabled status",
			token: Token{
				UserId:         1,
				Key:            "disabled-token-key",
				Status:         common.TokenStatusDisabled,
				ExpiredTime:    -1,
				RemainQuota:    100,
				UnlimitedQuota: false,
			},
		},
		{
			name: "expired time",
			token: Token{
				UserId:         1,
				Key:            "expired-time-token-key",
				Status:         common.TokenStatusEnabled,
				ExpiredTime:    now - 1,
				RemainQuota:    100,
				UnlimitedQuota: false,
			},
		},
		{
			name: "empty quota",
			token: Token{
				UserId:         1,
				Key:            "empty-quota-token-key",
				Status:         common.TokenStatusEnabled,
				ExpiredTime:    -1,
				RemainQuota:    0,
				UnlimitedQuota: false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, DB.Create(&tc.token).Error)

			token, err := ValidateUserToken(tc.token.Key)

			require.NotNil(t, token)
			require.ErrorIs(t, err, ErrTokenInvalid)
			require.NotContains(t, err.Error(), tc.token.Key[:3])
			require.NotContains(t, err.Error(), tc.token.Key[len(tc.token.Key)-3:])
			require.NotContains(t, err.Error(), "TokenStatus")
			require.NotContains(t, err.Error(), "RemainQuota")
		})
	}
}

func TestValidateUserTokenReturnsSentinelForMissingOrUnknownToken(t *testing.T) {
	truncateTables(t)
	initCol()

	_, err := ValidateUserToken("")
	require.ErrorIs(t, err, ErrTokenNotProvided)

	_, err = ValidateUserToken("missing-token-key")
	require.ErrorIs(t, err, ErrTokenInvalid)
	require.False(t, errors.Is(err, ErrDatabase))
}

func TestValidateAndFillReturnsSentinelErrors(t *testing.T) {
	truncateTables(t)

	user := &User{Username: "", Password: "password"}
	err := user.ValidateAndFill()
	require.ErrorIs(t, err, ErrUserEmptyCredentials)

	user = &User{Username: "missing-user", Password: "password"}
	err = user.ValidateAndFill()
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.False(t, errors.Is(err, ErrDatabase))

	oldDB := DB
	DB = oldDB.Table("missing_users_table")
	t.Cleanup(func() {
		DB = oldDB
	})

	user = &User{Username: "db-error-user", Password: "password"}
	err = user.ValidateAndFill()
	require.ErrorIs(t, err, ErrDatabase)
}
