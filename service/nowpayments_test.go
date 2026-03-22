package service

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
)

func TestVerifyNOWPaymentsSignature(t *testing.T) {
	originalSecret := setting.NOWPaymentsIPNSecret
	setting.NOWPaymentsIPNSecret = "test-secret"
	t.Cleanup(func() {
		setting.NOWPaymentsIPNSecret = originalSecret
	})

	payload := []byte(`{"payment_status":"finished","payment_id":123,"order_id":"ref_123"}`)
	canonicalPayload, err := canonicalizeNOWPaymentsPayload(payload)
	if err != nil {
		t.Fatalf("canonicalizeNOWPaymentsPayload returned error: %v", err)
	}

	mac := hmac.New(sha512.New, []byte(setting.NOWPaymentsIPNSecret))
	mac.Write(canonicalPayload)
	signature := hex.EncodeToString(mac.Sum(nil))

	if !VerifyNOWPaymentsSignature(payload, signature) {
		t.Fatal("expected valid signature to verify")
	}
	if VerifyNOWPaymentsSignature(payload, "deadbeef") {
		t.Fatal("expected invalid signature to fail verification")
	}
	if VerifyNOWPaymentsSignature(nil, signature) {
		t.Fatal("expected empty payload to fail verification")
	}
}

func TestCanonicalizeNOWPaymentsPayload(t *testing.T) {
	payload := []byte(`{"z":2,"a":{"b":2,"a":1},"c":[{"d":2,"c":1}]}`)
	canonicalPayload, err := canonicalizeNOWPaymentsPayload(payload)
	if err != nil {
		t.Fatalf("canonicalizeNOWPaymentsPayload returned error: %v", err)
	}

	want := `{"a":{"a":1,"b":2},"c":[{"c":1,"d":2}],"z":2}`
	if string(canonicalPayload) != want {
		t.Fatalf("canonical payload = %s, want %s", string(canonicalPayload), want)
	}
}

func TestParseNOWPaymentsUSDTNetworksPreservesConfiguredCode(t *testing.T) {
	originalNetworks := setting.NOWPaymentsUSDTNetworks
	setting.NOWPaymentsUSDTNetworks = `[{"code":"USDTTON","name":"USDT on TON","enabled":true,"sort":1}]`
	t.Cleanup(func() {
		setting.NOWPaymentsUSDTNetworks = originalNetworks
	})

	networks, err := ParseNOWPaymentsUSDTNetworks(setting.NOWPaymentsUSDTNetworks)
	if err != nil {
		t.Fatalf("ParseNOWPaymentsUSDTNetworks returned error: %v", err)
	}
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	if networks[0].Code != "USDTTON" {
		t.Fatalf("expected configured code to be preserved, got %q", networks[0].Code)
	}

	network, ok := GetNOWPaymentsNetworkByCode("usdtton")
	if !ok {
		t.Fatal("expected lowercase lookup to find configured network")
	}
	if network.Code != "USDTTON" {
		t.Fatalf("expected lookup to preserve configured code, got %q", network.Code)
	}
}

func TestNOWPaymentsFloat64UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want float64
	}{
		{name: "number", raw: `1.25`, want: 1.25},
		{name: "string", raw: `"2.50"`, want: 2.5},
		{name: "empty string", raw: `""`, want: 0},
		{name: "null", raw: `null`, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var value NOWPaymentsFloat64
			if err := common.UnmarshalJsonStr(tc.raw, &value); err != nil {
				t.Fatalf("common.UnmarshalJsonStr returned error: %v", err)
			}
			if float64(value) != tc.want {
				t.Fatalf("value = %v, want %v", float64(value), tc.want)
			}
		})
	}
}
