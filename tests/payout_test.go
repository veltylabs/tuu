package tests

import (
	"testing"

	"github.com/veltylabs/payment_manager"
	"github.com/veltylabs/tuu"
)

func TestPayoutTerms(t *testing.T) {
	tests := []struct {
		name         string
		plan         *tuu.PayoutPlan
		chargeAmount int64
		chargeCurr   string
		expectedFee  int64
		expectedTax  int64
		expectedDays int
		expectPayout bool
	}{
		{
			name:         "P1 - 149 bp, IVA 1900, 2 days",
			plan:         &tuu.PayoutPlan{PercentBasisPoints: 149, FeeTaxBasisPoints: 1900, BusinessDays: 2},
			chargeAmount: 30000,
			chargeCurr:   "CLP",
			expectedFee:  447,
			expectedTax:  85,
			expectedDays: 2,
			expectPayout: true,
		},
		{
			name:         "P2 - 79 bp + 65, IVA 1900, 2 days",
			plan:         &tuu.PayoutPlan{PercentBasisPoints: 79, FixedFee: 65, FeeTaxBasisPoints: 1900, BusinessDays: 2},
			chargeAmount: 30000,
			chargeCurr:   "CLP",
			expectedFee:  302,
			expectedTax:  57,
			expectedDays: 2,
			expectPayout: true,
		},
		{
			name:         "P3 - 149 bp, 0 days",
			plan:         &tuu.PayoutPlan{PercentBasisPoints: 149, FeeTaxBasisPoints: 1900, BusinessDays: 0},
			chargeAmount: 1001,
			chargeCurr:   "CLP",
			expectedFee:  15,
			expectedTax:  3,
			expectedDays: 0,
			expectPayout: true,
		},
		{
			name:         "P4 - Payout nil",
			plan:         nil,
			chargeAmount: 1000,
			chargeCurr:   "CLP",
			expectPayout: false,
		},
		{
			name:         "P5 - 149 bp, USD",
			plan:         &tuu.PayoutPlan{PercentBasisPoints: 149, FeeTaxBasisPoints: 1900, BusinessDays: 2},
			chargeAmount: 100,
			chargeCurr:   "USD",
			expectedFee:  1,
			expectedTax:  0,
			expectedDays: 2,
			expectPayout: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term := tuu.New(tuu.Config{Payout: tc.plan})

			pt, ok := term.(paymentmanager.PayoutTerms)
			if !tc.expectPayout {
				if ok {
					t.Fatalf("expected term not to implement PayoutTerms, but it did")
				}
				return
			}
			if !ok {
				t.Fatalf("expected term to implement PayoutTerms, but it did not")
			}

			c := paymentmanager.PayoutCharge{Amount: tc.chargeAmount, Currency: tc.chargeCurr}
			payout, err := pt.Payout(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if payout.Fee != tc.expectedFee {
				t.Errorf("expected fee %d, got %d", tc.expectedFee, payout.Fee)
			}
			if payout.FeeTax != tc.expectedTax {
				t.Errorf("expected fee tax %d, got %d", tc.expectedTax, payout.FeeTax)
			}
			if payout.BusinessDays != tc.expectedDays {
				t.Errorf("expected business days %d, got %d", tc.expectedDays, payout.BusinessDays)
			}
		})
	}
}
