package models

import "testing"

func TestWalletType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		walletType WalletType
		want      bool
	}{
		{
			name:      "Valid primary",
			walletType: WalletTypePrimary,
			want:      true,
		},
		{
			name:      "Valid savings",
			walletType: WalletTypeSavings,
			want:      true,
		},
		{
			name:      "Valid investment",
			walletType: WalletTypeInvestment,
			want:      true,
		},
		{
			name:      "Valid business",
			walletType: WalletTypeBusiness,
			want:      true,
		},
		{
			name:      "Valid escrow",
			walletType: WalletTypeEscrow,
			want:      true,
		},
		{
			name:      "Invalid empty",
			walletType: WalletType(""),
			want:      false,
		},
		{
			name:      "Invalid random",
			walletType: WalletType("invalid"),
			want:      false,
		},
		{
			name:      "Invalid checking",
			walletType: WalletType("checking"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.walletType.IsValid(); got != tt.want {
				t.Errorf("WalletType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWalletType_String(t *testing.T) {
	tests := []struct {
		name      string
		walletType WalletType
		want      string
	}{
		{
			name:      "Primary string",
			walletType: WalletTypePrimary,
			want:      "primary",
		},
		{
			name:      "Savings string",
			walletType: WalletTypeSavings,
			want:      "savings",
		},
		{
			name:      "Investment string",
			walletType: WalletTypeInvestment,
			want:      "investment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.walletType.String(); got != tt.want {
				t.Errorf("WalletType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNullableWalletType_Scan(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		wantValid bool
		wantType  WalletType
		wantErr   bool
	}{
		{
			name:      "Valid string value",
			value:     "primary",
			wantValid: true,
			wantType:  WalletTypePrimary,
			wantErr:   false,
		},
		{
			name:      "Valid savings string",
			value:     "savings",
			wantValid: true,
			wantType:  WalletTypeSavings,
			wantErr:   false,
		},
		{
			name:      "NULL value",
			value:     nil,
			wantValid: false,
			wantType:  "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nwt NullableWalletType
			err := nwt.Scan(tt.value)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("NullableWalletType.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if nwt.Valid != tt.wantValid {
				t.Errorf("NullableWalletType.Scan() Valid = %v, want %v", nwt.Valid, tt.wantValid)
			}
			
			if nwt.Type != tt.wantType {
				t.Errorf("NullableWalletType.Scan() Type = %v, want %v", nwt.Type, tt.wantType)
			}
		})
	}
}
