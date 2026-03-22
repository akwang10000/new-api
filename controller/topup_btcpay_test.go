package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func TestNormalizeBTCPayInvoiceState(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		eventType string
		want      string
	}{
		{name: "settled status wins", status: "settled", eventType: "", want: common.TopUpStatusSuccess},
		{name: "expired status", status: "expired", eventType: "", want: common.TopUpStatusExpired},
		{name: "invalid status", status: "invalid", eventType: "", want: common.TopUpStatusExpired},
		{name: "settled event fallback", status: "paid", eventType: "InvoiceSettled", want: common.TopUpStatusSuccess},
		{name: "expired event fallback", status: "processing", eventType: "InvoiceExpired", want: common.TopUpStatusExpired},
		{name: "pending non-terminal", status: "paid", eventType: "InvoiceReceivedPayment", want: common.TopUpStatusPending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeBTCPayInvoiceState(tc.status, tc.eventType)
			if got != tc.want {
				t.Fatalf("normalizeBTCPayInvoiceState(%q, %q) = %q, want %q", tc.status, tc.eventType, got, tc.want)
			}
		})
	}
}

func TestValidateBTCPayInvoice(t *testing.T) {
	topUp := &model.TopUp{
		UserId:        7,
		Amount:        10,
		Money:         73,
		TradeNo:       "ref_123",
		PaymentMethod: PaymentMethodBTCPay,
	}
	invoiceAmount, err := getBTCPayInvoiceAmountUSD(topUp.Money)
	if err != nil {
		t.Fatalf("getBTCPayInvoiceAmountUSD returned error: %v", err)
	}

	invoice := &service.BTCPayInvoiceResponse{
		ID:       "invoice_123",
		Amount:   invoiceAmount,
		Currency: "USD",
		Status:   "settled",
		Metadata: map[string]any{
			"orderId":       topUp.TradeNo,
			"tradeNo":       topUp.TradeNo,
			"userId":        topUp.UserId,
			"topupAmount":   topUp.Amount,
			"paymentMethod": PaymentMethodBTCPay,
			"payMoney":      topUp.Money,
		},
	}

	if err := validateBTCPayInvoice(topUp, invoice); err != nil {
		t.Fatalf("validateBTCPayInvoice returned error: %v", err)
	}

	invoice.Amount = invoiceAmount + 1
	if err := validateBTCPayInvoice(topUp, invoice); err == nil {
		t.Fatal("expected invoice amount mismatch to fail validation")
	}
}
