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

func TestRechargeCreem_DoesNotCompleteNonCreemOrderEvenWhenTradeNoMatches(t *testing.T) {
	prepareTopUpTestDB(t)

	user := &User{Id: 4, Username: "creem-mismatch-user", Password: "password123"}
	require.NoError(t, DB.Create(user).Error)

	topUp := &TopUp{
		UserId:        user.Id,
		Amount:        20,
		Money:         20,
		TradeNo:       "creem-mismatch-ref",
		PaymentMethod: "alipay",
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	err := RechargeCreem(topUp.TradeNo, "paid@example.com", "paid user")
	require.Error(t, err)

	var reloadedTopUp TopUp
	require.NoError(t, DB.Where("trade_no = ?", topUp.TradeNo).First(&reloadedTopUp).Error)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
	require.Zero(t, reloadedTopUp.CompleteTime)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	require.Zero(t, reloadedUser.Quota)
	require.Empty(t, reloadedUser.Email)
}

func TestGetUserTopUpsLimitsRegularUsersToRecentOrders(t *testing.T) {
	prepareTopUpTestDB(t)

	now := common.GetTimestamp()
	user := &User{Id: 2, Username: "topup-window-user", Password: "password123"}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:     user.Id,
		Amount:     10,
		Money:      10,
		TradeNo:    "recent-order",
		CreateTime: now,
		Status:     common.TopUpStatusPending,
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:     user.Id,
		Amount:     10,
		Money:      10,
		TradeNo:    "old-order",
		CreateTime: now - topUpQueryWindowSeconds - 1,
		Status:     common.TopUpStatusPending,
	}).Error)

	topups, total, err := GetUserTopUps(user.Id, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, topups, 1)
	require.Equal(t, "recent-order", topups[0].TradeNo)
}

func TestSearchTopUpsSanitizesLikePatternAndKeepsPartialSearch(t *testing.T) {
	prepareTopUpTestDB(t)

	now := common.GetTimestamp()
	user := &User{Id: 3, Username: "topup-search-user", Password: "password123"}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:     user.Id,
		Amount:     10,
		Money:      10,
		TradeNo:    "ORDER-ABC-123",
		CreateTime: now,
		Status:     common.TopUpStatusPending,
	}).Error)

	topups, total, err := SearchUserTopUps(user.Id, "ABC", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, topups, 1)
	require.Equal(t, "ORDER-ABC-123", topups[0].TradeNo)

	_, _, err = SearchUserTopUps(user.Id, "%%", &common.PageInfo{Page: 1, PageSize: 10})
	require.Error(t, err)
}
