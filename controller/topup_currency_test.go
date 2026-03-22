package controller

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestGetPayMoneyCNYFromUSDDisplay(t *testing.T) {
	originalPrice := operation_setting.Price
	originalRate := operation_setting.USDExchangeRate
	originalGeneral := *operation_setting.GetGeneralSetting()
	originalPaymentSetting := *operation_setting.GetPaymentSetting()
	defer func() {
		operation_setting.Price = originalPrice
		operation_setting.USDExchangeRate = originalRate
		*operation_setting.GetGeneralSetting() = originalGeneral
		*operation_setting.GetPaymentSetting() = originalPaymentSetting
	}()

	operation_setting.Price = 1
	operation_setting.USDExchangeRate = 7.3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}

	got, err := getPayMoneyCNY(10, "default")
	if err != nil {
		t.Fatalf("getPayMoneyCNY returned error: %v", err)
	}
	if math.Abs(got-73.0) > 0.0001 {
		t.Fatalf("unexpected CNY amount: got %.4f want 73.0", got)
	}
}

func TestGetPayMoneyCNYFromCustomDisplay(t *testing.T) {
	originalPrice := operation_setting.Price
	originalRate := operation_setting.USDExchangeRate
	originalGeneral := *operation_setting.GetGeneralSetting()
	originalPaymentSetting := *operation_setting.GetPaymentSetting()
	defer func() {
		operation_setting.Price = originalPrice
		operation_setting.USDExchangeRate = originalRate
		*operation_setting.GetGeneralSetting() = originalGeneral
		*operation_setting.GetPaymentSetting() = originalPaymentSetting
	}()

	operation_setting.Price = 1
	operation_setting.USDExchangeRate = 7.3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCustom
	operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate = 2
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}

	got, err := getPayMoneyCNY(10, "default")
	if err != nil {
		t.Fatalf("getPayMoneyCNY returned error: %v", err)
	}
	if math.Abs(got-36.5) > 0.0001 {
		t.Fatalf("unexpected CNY amount: got %.4f want 36.5", got)
	}
}
