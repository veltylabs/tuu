package tuu

import (
	"github.com/veltylabs/payment_manager"
	"webtyp.com/money"
)

// TerminalWithPayout es el Terminal de una cuenta con plan de abono configurado; además cumple
// paymentmanager.PayoutTerms. New lo devuelve cuando Config.Payout no es nil.
// Ejemplo: var pos paymentmanager.Terminal = tuu.New(cfg) // con cfg.Payout → también PayoutTerms
type TerminalWithPayout struct{ *Terminal }

var _ paymentmanager.PayoutTerms = TerminalWithPayout{}

// Payout calcula comisión, IVA de la comisión y días hábiles del cobro según el plan.
func (t TerminalWithPayout) Payout(c paymentmanager.PayoutCharge) (paymentmanager.Payout, error) {
	p := t.cfg.Payout
	if p == nil {
		return paymentmanager.Payout{}, ErrNoPayoutPlan
	}
	amount, err := money.New(c.Amount, money.Currency(c.Currency))
	if err != nil {
		return paymentmanager.Payout{}, err
	}
	pct, err := money.Share(amount, p.PercentBasisPoints)
	if err != nil {
		return paymentmanager.Payout{}, err
	}
	fee := pct.Amount + p.FixedFee
	feeMoney, err := money.New(fee, money.Currency(c.Currency))
	if err != nil {
		return paymentmanager.Payout{}, err
	}
	tax, err := money.Share(feeMoney, p.FeeTaxBasisPoints)
	if err != nil {
		return paymentmanager.Payout{}, err
	}
	return paymentmanager.Payout{Fee: fee, FeeTax: tax.Amount, BusinessDays: p.BusinessDays}, nil
}
