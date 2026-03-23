package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting"
)

func TestVerifyBEpusdtSignature(t *testing.T) {
	originalToken := setting.BEpusdtToken
	originalSecret := setting.BEpusdtWebhookSecret
	setting.BEpusdtToken = "test-token"
	setting.BEpusdtWebhookSecret = "test-secret"
	t.Cleanup(func() {
		setting.BEpusdtToken = originalToken
		setting.BEpusdtWebhookSecret = originalSecret
	})

	params := map[string]string{
		"order_id": "ref_123",
		"amount":   "12.34",
		"status":   "2",
	}
	signature := signBEpusdtParamsWithToken(params, setting.BEpusdtWebhookSecret)
	if signature == "" {
		t.Fatal("expected signature to be generated")
	}
	if !VerifyBEpusdtSignature(params, signature) {
		t.Fatal("expected signature to verify")
	}
	if VerifyBEpusdtSignature(params, "deadbeef") {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestCanonicalBEpusdtDecimal(t *testing.T) {
	testCases := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "integer", value: 1, want: "1"},
		{name: "single decimal", value: 7.3, want: "7.3"},
		{name: "trimmed trailing zeros", value: 7.30, want: "7.3"},
		{name: "rounded", value: 7.356, want: "7.36"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanonicalBEpusdtDecimal(tc.value, 2)
			if got != tc.want {
				t.Fatalf("CanonicalBEpusdtDecimal(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseBEpusdtUSDTNetworks(t *testing.T) {
	raw := `[{"code":"usdt.trc20","name":"USDT on TRC20","enabled":true,"sort":2},{"code":"usdt.bep20","name":"USDT on BEP20","enabled":true,"sort":1}]`
	networks, err := ParseBEpusdtUSDTNetworks(raw)
	if err != nil {
		t.Fatalf("ParseBEpusdtUSDTNetworks returned error: %v", err)
	}
	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(networks))
	}
	if networks[0].Code != "usdt.bep20" {
		t.Fatalf("expected sorted networks, got first code %q", networks[0].Code)
	}
}

func TestQueryBEpusdtOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pay/check-status/trade-123" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"trade_id":"trade-123","status":2}`))
	}))
	defer server.Close()

	client := &BEpusdtClient{
		baseURL: server.URL,
		token:   "test-token",
		client:  server.Client(),
	}

	resp, err := client.QueryOrder("trade-123")
	if err != nil {
		t.Fatalf("QueryOrder returned error: %v", err)
	}
	if resp.TradeID != "trade-123" {
		t.Fatalf("unexpected trade_id: %s", resp.TradeID)
	}
	if resp.Status != 2 {
		t.Fatalf("unexpected status: %d", resp.Status)
	}
}
