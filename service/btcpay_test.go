package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/QuantumNous/new-api/setting"
)

func TestVerifyBTCPaySignature(t *testing.T) {
	originalSecret := setting.BTCPayWebhookSecret
	setting.BTCPayWebhookSecret = "test-secret"
	t.Cleanup(func() {
		setting.BTCPayWebhookSecret = originalSecret
	})

	payload := []byte(`{"invoiceId":"invoice_123","type":"InvoiceSettled"}`)
	mac := hmac.New(sha256.New, []byte(setting.BTCPayWebhookSecret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	if !VerifyBTCPaySignature(payload, signature) {
		t.Fatal("expected signature without prefix to verify")
	}
	if !VerifyBTCPaySignature(payload, "sha256="+signature) {
		t.Fatal("expected signature with sha256 prefix to verify")
	}
	if VerifyBTCPaySignature(payload, "sha256=deadbeef") {
		t.Fatal("expected invalid signature to fail verification")
	}
	if VerifyBTCPaySignature(nil, signature) {
		t.Fatal("expected empty payload to fail verification")
	}
}
