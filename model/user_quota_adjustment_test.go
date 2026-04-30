package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestEditDoesNotChangeQuota(t *testing.T) {
	truncateTables(t)

	user := &User{Id: 101, Username: "quota-edit-user", Password: "password", Quota: 1000, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(user).Error)

	updated := &User{Id: user.Id, Username: user.Username, DisplayName: "edited", Group: "default", Quota: 999999, Status: common.UserStatusEnabled}
	require.NoError(t, updated.Edit(false))

	var stored User
	require.NoError(t, DB.Where("id = ?", user.Id).First(&stored).Error)
	require.Equal(t, 1000, stored.Quota)
	require.Equal(t, "edited", stored.DisplayName)
}

func TestDecreaseUserQuotaCanBypassBatchUpdates(t *testing.T) {
	truncateTables(t)

	previousBatchUpdateEnabled := common.BatchUpdateEnabled
	common.BatchUpdateEnabled = true
	t.Cleanup(func() {
		common.BatchUpdateEnabled = previousBatchUpdateEnabled
	})

	user := &User{Id: 102, Username: "quota-bypass-user", Password: "password", Quota: 1000, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(user).Error)

	require.NoError(t, DecreaseUserQuota(user.Id, 250, true))

	var stored User
	require.NoError(t, DB.Where("id = ?", user.Id).First(&stored).Error)
	require.Equal(t, 750, stored.Quota)
}
