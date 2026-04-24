package controller

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
	"gorm.io/gorm"
)

func setupStripeTopUpTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldRedisEnabled := common.RedisEnabled
	oldDB := model.DB
	oldLogDB := model.LOG_DB

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.TopUp{}, &model.SubscriptionOrder{}))

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.RedisEnabled = oldRedisEnabled
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func stripeExpiredEvent(tradeNo string) stripe.Event {
	return stripe.Event{
		Type: stripe.EventTypeCheckoutSessionExpired,
		Data: &stripe.EventData{Object: map[string]interface{}{
			"client_reference_id": tradeNo,
			"status":              "expired",
		}},
	}
}

func TestSessionExpiredOnlyExpiresPendingStripeTopUp(t *testing.T) {
	db := setupStripeTopUpTestDB(t)

	successTopUp := &model.TopUp{
		UserId:          1,
		Amount:          10,
		Money:           10,
		TradeNo:         "stripe-success-ref",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusSuccess,
	}
	nonStripeTopUp := &model.TopUp{
		UserId:          1,
		Amount:          10,
		Money:           10,
		TradeNo:         "alipay-pending-ref",
		PaymentMethod:   "alipay",
		PaymentProvider: model.PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}
	stripeTopUp := &model.TopUp{
		UserId:          1,
		Amount:          10,
		Money:           10,
		TradeNo:         "stripe-pending-ref",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(successTopUp).Error)
	require.NoError(t, db.Create(nonStripeTopUp).Error)
	require.NoError(t, db.Create(stripeTopUp).Error)

	sessionExpired(stripeExpiredEvent(successTopUp.TradeNo))
	sessionExpired(stripeExpiredEvent(nonStripeTopUp.TradeNo))
	sessionExpired(stripeExpiredEvent(stripeTopUp.TradeNo))

	var reloadedSuccess model.TopUp
	require.NoError(t, db.Where("trade_no = ?", successTopUp.TradeNo).First(&reloadedSuccess).Error)
	require.Equal(t, common.TopUpStatusSuccess, reloadedSuccess.Status)

	var reloadedNonStripe model.TopUp
	require.NoError(t, db.Where("trade_no = ?", nonStripeTopUp.TradeNo).First(&reloadedNonStripe).Error)
	require.Equal(t, common.TopUpStatusPending, reloadedNonStripe.Status)

	var reloadedStripe model.TopUp
	require.NoError(t, db.Where("trade_no = ?", stripeTopUp.TradeNo).First(&reloadedStripe).Error)
	require.Equal(t, common.TopUpStatusExpired, reloadedStripe.Status)
}
