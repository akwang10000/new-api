package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func TestGetExpectedBEpusdtCallbackAmount(t *testing.T) {
	topUp := &model.TopUp{Amount: 1, Money: 1}
	got, err := getExpectedBEpusdtCallbackAmount(topUp)
	if err != nil {
		t.Fatalf("getExpectedBEpusdtCallbackAmount returned error: %v", err)
	}
	if got.String() != "1" {
		t.Fatalf("unexpected callback amount: got %s want 1", got.String())
	}
}

func TestValidateBEpusdtCallbackUsesCNYAmount(t *testing.T) {
	topUp := &model.TopUp{
		TradeNo:       "ref_123",
		PaymentMethod: "bepusdt_usdt_trc20",
		Amount:        1,
		Money:         1,
	}

	err := validateBEpusdtCallback(topUp, map[string]string{
		"order_id":      "ref_123",
		"amount":        "1",
		"actual_amount": "0.15",
		"token":         "TFWSj5fS6mqxnBFFxGD1YvPJ8uwc7zSr4L",
		"trade_id":      "trade_123",
		"trade_type":    "usdt.trc20",
		"fiat":          "CNY",
		"status":        "2",
	}, &service.BEpusdtQueryOrderResponse{
		TradeID: "trade_123",
		Status:  2,
	})
	if err != nil {
		t.Fatalf("validateBEpusdtCallback returned error: %v", err)
	}
}

func TestValidateBEpusdtCallbackRejectsTradeTypeMismatch(t *testing.T) {
	topUp := &model.TopUp{
		TradeNo:       "ref_123",
		PaymentMethod: "bepusdt_usdt_trc20",
		Amount:        1,
		Money:         1,
	}

	err := validateBEpusdtCallback(topUp, map[string]string{
		"order_id":      "ref_123",
		"amount":        "1",
		"actual_amount": "0.15",
		"token":         "TFWSj5fS6mqxnBFFxGD1YvPJ8uwc7zSr4L",
		"trade_id":      "trade_123",
		"trade_type":    "usdt.erc20",
		"status":        "2",
	}, &service.BEpusdtQueryOrderResponse{
		TradeID: "trade_123",
		Status:  2,
	})
	if err == nil {
		t.Fatal("expected trade_type mismatch error")
	}
}

func TestGetBEpusdtNotifyURL(t *testing.T) {
	got, err := getBEpusdtNotifyURL("https://robot2.indevs.in")
	if err != nil {
		t.Fatalf("getBEpusdtNotifyURL returned error: %v", err)
	}
	want := "https://robot2.indevs.in/api/bepusdt/webhook"
	if got != want {
		t.Fatalf("unexpected notify url: got %s want %s", got, want)
	}
}

func TestGetBEpusdtAmountsUseSameNumericValue(t *testing.T) {
	charge, err := getBEpusdtChargeAmountCNY(5)
	if err != nil {
		t.Fatalf("getBEpusdtChargeAmountCNY returned error: %v", err)
	}
	if charge != 5 {
		t.Fatalf("unexpected charge amount: got %.2f want 5", charge)
	}

	credit, err := getBEpusdtCreditAmountUSD(5)
	if err != nil {
		t.Fatalf("getBEpusdtCreditAmountUSD returned error: %v", err)
	}
	if credit != 5 {
		t.Fatalf("unexpected credit amount: got %.2f want 5", credit)
	}
}
