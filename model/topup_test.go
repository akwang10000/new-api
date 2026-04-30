package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func prepareTopUpTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&TopUp{}, &User{}))
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM top_ups")
		DB.Exec("DELETE FROM users")
		LOG_DB.Exec("DELETE FROM logs")
	})
}

func TestRecordTopupLogStoresAuditInfoWithNodeName(t *testing.T) {
	prepareTopUpTestDB(t)
	originalNodeName := common.NodeName
	common.NodeName = "checkout-node-a"
	t.Cleanup(func() {
		common.NodeName = originalNodeName
	})

	RecordTopupLog(1, "充值成功", "203.0.113.10", "stripe", "stripe")

	var log Log
	require.NoError(t, LOG_DB.Where("user_id = ?", 1).First(&log).Error)
	require.Equal(t, LogTypeTopup, log.Type)
	require.Equal(t, "203.0.113.10", log.Ip)

	other, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "203.0.113.10", adminInfo["caller_ip"])
	require.Equal(t, "stripe", adminInfo["payment_method"])
	require.Equal(t, "stripe", adminInfo["callback_payment_method"])
	require.Equal(t, common.Version, adminInfo["version"])
	require.Equal(t, "checkout-node-a", adminInfo["node_name"])
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

	err := Recharge(topUp.TradeNo, "cus_test_123", "")
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

	err := RechargeCreem(topUp.TradeNo, "paid@example.com", "paid user", "")
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

func TestManualCompleteTopUpRecordsCallerIPAndPaymentMethod(t *testing.T) {
	prepareTopUpTestDB(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 10
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
	})

	user := &User{Id: 5, Username: "manual-topup-user", Password: "password123"}
	require.NoError(t, DB.Create(user).Error)
	topUp := &TopUp{
		UserId:        user.Id,
		Amount:        20,
		Money:         20,
		TradeNo:       "manual-complete-order",
		PaymentMethod: "alipay",
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	require.NoError(t, ManualCompleteTopUp(topUp.TradeNo, "198.51.100.20"))

	var log Log
	require.NoError(t, LOG_DB.Where("user_id = ? AND type = ?", user.Id, LogTypeTopup).First(&log).Error)
	require.Equal(t, "198.51.100.20", log.Ip)
	other, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "198.51.100.20", adminInfo["caller_ip"])
	require.Equal(t, "alipay", adminInfo["payment_method"])
	require.Equal(t, "admin", adminInfo["callback_payment_method"])
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

func TestSearchAllTopUpsCountUsesHardLimit(t *testing.T) {
	prepareTopUpTestDB(t)

	now := common.GetTimestamp()
	topups := make([]TopUp, 0, searchTopUpCountHardLimit+1)
	for i := 0; i < searchTopUpCountHardLimit+1; i++ {
		topups = append(topups, TopUp{
			UserId:     9,
			Amount:     10,
			Money:      10,
			TradeNo:    "hard-limit-order-" + strconv.Itoa(i),
			CreateTime: now,
			Status:     common.TopUpStatusPending,
		})
	}
	require.NoError(t, DB.CreateInBatches(topups, 500).Error)

	_, total, err := SearchAllTopUps("hard-limit-order", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, searchTopUpCountHardLimit, total)
}
