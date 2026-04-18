package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func prepareTopUpTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&TopUp{}, &User{}, &Log{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM top_ups")
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})
}

func TestRecharge_DoesNotCompleteNonStripeOrderEvenWhenTradeNoMatches(t *testing.T) {
	prepareTopUpTestDB(t)

	user := &User{
		Id:       1,
		Username: "stripe-mismatch-user",
		Password: "password123",
	}
	require.NoError(t, DB.Create(user).Error)

	topUp := &TopUp{
		UserId:        user.Id,
		Amount:        20,
		Money:         20,
		TradeNo:       "USR20NOQ4ttgU1776418221",
		PaymentMethod: "alipay",
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	err := Recharge(topUp.TradeNo, "cus_test_123")
	require.Error(t, err)

	var reloadedTopUp TopUp
	require.NoError(t, DB.Where("trade_no = ?", topUp.TradeNo).First(&reloadedTopUp).Error)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
	require.Zero(t, reloadedTopUp.CompleteTime)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	require.Zero(t, reloadedUser.Quota)
	require.Empty(t, reloadedUser.StripeCustomer)
}
