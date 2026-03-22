package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func TestNormalizeNOWPaymentsPaymentStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "finished", status: "finished", want: common.TopUpStatusSuccess},
		{name: "failed", status: "failed", want: common.TopUpStatusExpired},
		{name: "expired", status: "expired", want: common.TopUpStatusExpired},
		{name: "waiting", status: "waiting", want: common.TopUpStatusPending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeNOWPaymentsPaymentStatus(tc.status); got != tc.want {
				t.Fatalf("normalizeNOWPaymentsPaymentStatus(%q) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestValidateNOWPaymentsPayment(t *testing.T) {
	topUp := &model.TopUp{
		UserId:        7,
		Amount:        10,
		Money:         20,
		TradeNo:       "ref_123",
		PaymentMethod: "nowpayments_usdtton",
	}
	payment := &service.NOWPaymentsPaymentResponse{
		OrderID:       topUp.TradeNo,
		PaymentStatus: "finished",
		PayCurrency:   "usdtton",
		PriceCurrency: "usd",
		PriceAmount:   service.NOWPaymentsFloat64(topUp.Money),
	}

	if err := validateNOWPaymentsPayment(topUp, payment); err != nil {
		t.Fatalf("validateNOWPaymentsPayment returned error: %v", err)
	}

	payment.PriceAmount = service.NOWPaymentsFloat64(19)
	if err := validateNOWPaymentsPayment(topUp, payment); err == nil {
		t.Fatal("expected price amount mismatch to fail validation")
	}
}
