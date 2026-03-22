package model

import "testing"

func TestValidateBEpusdtTopUp(t *testing.T) {
	testCases := []struct {
		name    string
		topUp   *TopUp
		wantErr bool
	}{
		{
			name: "valid invariant",
			topUp: &TopUp{
				PaymentMethod: "bepusdt_usdt_trc20",
				Amount:        5,
				Money:         5,
			},
			wantErr: false,
		},
		{
			name: "mismatched money",
			topUp: &TopUp{
				PaymentMethod: "bepusdt_usdt_trc20",
				Amount:        1,
				Money:         7.3,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBEpusdtTopUp(tc.topUp)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
