package controller

import "testing"

func TestValidateBTCPayServerURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{name: "empty allowed", rawURL: "", wantErr: false},
		{name: "https allowed", rawURL: "https://btcpay.example.com", wantErr: false},
		{name: "http allowed", rawURL: "http://localhost:23000", wantErr: false},
		{name: "invalid scheme", rawURL: "ftp://btcpay.example.com", wantErr: true},
		{name: "missing host", rawURL: "https:///api", wantErr: true},
		{name: "query not allowed", rawURL: "https://btcpay.example.com?a=1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBTCPayServerURL(tc.rawURL)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateBEpusdtBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{name: "empty allowed", rawURL: "", wantErr: false},
		{name: "https allowed", rawURL: "https://pay.example.com", wantErr: false},
		{name: "http allowed", rawURL: "http://localhost:8080", wantErr: false},
		{name: "invalid scheme", rawURL: "ftp://pay.example.com", wantErr: true},
		{name: "missing host", rawURL: "https:///api", wantErr: true},
		{name: "query not allowed", rawURL: "https://pay.example.com?a=1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBEpusdtBaseURL(tc.rawURL)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateBEpusdtUSDTNetworks(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "valid", raw: `[{"code":"usdt.trc20","name":"USDT on TRC20","enabled":true,"sort":1}]`, wantErr: false},
		{name: "missing code", raw: `[{"name":"USDT on TRC20","enabled":true}]`, wantErr: true},
		{name: "duplicate code", raw: `[{"code":"usdt.trc20","name":"A","enabled":true},{"code":"USDT.TRC20","name":"B","enabled":true}]`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBEpusdtUSDTNetworks(tc.raw)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateNOWPaymentsUSDTNetworks(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "valid", raw: `[{"code":"USDTTON","name":"USDT on TON","enabled":true,"sort":1}]`, wantErr: false},
		{name: "missing code", raw: `[{"name":"USDT on TON","enabled":true}]`, wantErr: true},
		{name: "duplicate code", raw: `[{"code":"usdtton","name":"A","enabled":true},{"code":"USDTTON","name":"B","enabled":true}]`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNOWPaymentsUSDTNetworks(tc.raw)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateNOWPaymentsCryptoAmountOptions(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "valid", raw: `[5,10,20]`, wantErr: false},
		{name: "zero", raw: `[0,10]`, wantErr: true},
		{name: "duplicate", raw: `[5,5]`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNOWPaymentsCryptoAmountOptions(tc.raw)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
