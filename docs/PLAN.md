---
PLAN: "feat: TUU payout terms (fee, fee IVA, business days) from the contracted plan"
TAG: v0.2.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 913560575629694779
PR: https://github.com/veltylabs/tuu/pull/2
REVIEWED_COMMIT: eca0e548927081a72fc245b64ad24c6abed1fe00
---

> Read AGENTS.md at the repo root before any change.

<!-- Parked. Dispatch after this repo's docs/PLAN.md (v0.1.0) is closed and payment_manager v0.4.0
(docs/PLAN_PAYOUT.md there) is published. The dispatcher renames this file to docs/PLAN.md and writes the tag. -->

# Plan — `tuu.Terminal` tells when and how much TUU deposits

## Why

TUU deposits a card payment N bank business days later, minus its fee and the IVA of the fee. The numbers depend on
the plan each account contracted (TUU 2026: flat 1.49 % or mixed 0.79 % + $65 per sale, deposit in 1, 2, 5 or 10 days;
the later the deposit, the lower the fee). `payment_manager` v0.4.0 asks it through the optional `PayoutTerms`
capability. The plan is configuration of each account, never constants in the code: TUU changes prices.

Example: $30.000, flat 1.49 %, IVA 19 %, 2 days → fee 447, IVA 85, deposit in 2 business days.

Language: identifiers and error strings in English; **README, docs, diagrams and comments in Spanish**, each exported
symbol with a `// Ejemplo: …` line.

## Delivery rules

- **Commit after every stage.** Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash`.
- Never pause to ask permission or to report progress. When the Delivery check passes: push and open the PR. Stuck
  for 30 minutes: open the PR anyway, listing what fails.
- A PR that does not compile, or whose tests could not run, is a failed delivery.
- The code blocks below are the implementation: copy them.

## Tools and pinned versions

```bash
go install webtyp.com/devflow/cmd/gotest@latest
export GOPRIVATE='webtyp.com/*,github.com/veltylabs/*' GOFLAGS=-mod=mod
go get github.com/veltylabs/payment_manager@v0.4.0 webtyp.com/money@v0.2.0 && go mod tidy   # ← tags confirmed by the dispatcher
```

## APIs used

`payment_manager` v0.4.0: `type PayoutTerms interface{ Payout(c PayoutCharge) (Payout, error) }`,
`PayoutCharge{Amount int64; Currency, Method, Channel string}`, `Payout{Fee, FeeTax int64; BusinessDays int}`.
`webtyp.com/money` v0.2.0: `money.New(amount int64, c Currency) (Money, error)`, `money.Share(m Money, bp int64)
(Money, error)` (nearest, ties away from zero). If a shape differs, stop and name it in the PR.

## Stage 1 — Plan in the config (`config.go`)

```go
// PayoutPlan es el plan de abono contratado con TUU para una cuenta. Los valores salen del Espacio de Trabajo de TUU.
// Ejemplo: tuu.PayoutPlan{PercentBasisPoints: 149, FeeTaxBasisPoints: 1900, BusinessDays: 2} // porcentual 1,49 %, abono a 2 días
// Ejemplo: tuu.PayoutPlan{PercentBasisPoints: 79, FixedFee: 65, FeeTaxBasisPoints: 1900, BusinessDays: 2} // mixta 0,79 % + $65
type PayoutPlan struct {
	PercentBasisPoints int64 // comisión porcentual en puntos base (149 = 1,49 %)
	FixedFee           int64 // comisión fija por venta en pesos (mixta); 0 en la porcentual
	FeeTaxBasisPoints  int64 // IVA sobre la comisión (1900 = 19 %)
	BusinessDays       int   // días hábiles bancarios hasta el depósito (0 = al instante)
}
```

`Config` gains `Payout *PayoutPlan // nil: el adaptador no declara abono (payment_manager registra el cobro sin fecha ni comisión)`.

`errors.go` adds `ErrNoPayoutPlan = fmt.Err("tuu: Config.Payout is not set")`. **Commit.**

## Stage 2 — `PayoutTerms` (`payout.go`, new)

`*Terminal` must implement `PayoutTerms` **only when a plan is configured**, so `payment_manager` does not ask an
account without one. Do it with a wrapper type:

```go
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
```

`New` changes its return type to the port so callers get the right capability:

```go
// New crea el adaptador; con Config.Payout también calcula el abono. No llama a TUU.
// Ejemplo: pos := tuu.New(tuu.Config{APIKey: llave, Devices: maquinas, Payout: &plan})
func New(cfg Config) paymentmanager.Terminal {
	// … same defaults as today …
	t := &Terminal{cfg: cfg}
	if cfg.Payout != nil {
		return TerminalWithPayout{t}
	}
	return t
}
```

Tests that called methods on `*tuu.Terminal` keep working through the interface; a test that needs the concrete
type asserts it.

| Case | Plan | Charge | Payout |
|---|---|---|---|
| P1 | 149 bp, IVA 1900, 2 days | 30000 | Fee 447, FeeTax 85, BusinessDays 2 |
| P2 | 79 bp + 65, IVA 1900, 2 days | 30000 | Fee 302 (237 + 65), FeeTax 57 |
| P3 | 149 bp, 0 days | 1001 | Fee 15, FeeTax 3, BusinessDays 0 |
| P4 | `Payout` nil | — | `tuu.New` result does **not** implement `PayoutTerms` |
| P5 | 149 bp | USD 100 | `money.New` accepts USD: Fee 1, FeeTax 0 (no CLP check here; Push already rejects USD) |

(Check every expected number by hand against `money.Share`'s rounding before writing the test; if one differs by 1,
the rounding rule wins and the test is corrected, never the code.) **Commit.**

## Stage 3 — Docs (Spanish)

`README.md`: the two plan examples (porcentual y mixta) and the sentence "los valores se copian del Espacio de Trabajo
de TUU; cambian si TUU cambia sus precios". `docs/ARCHITECTURE.md`: section `## Abono y comisión` with P1. **Commit.**

## Stage 4 — Spanish (only what is still in English)

Every `README.md`, `docs/*` file, diagram and code comment in Spanish; translate only what is still English; each
exported symbol's comment has a `// Ejemplo: …` line. **Commit.**

## Delivery check

```bash
go mod tidy && gotest
```

## Stages

| # | What | Files |
|---|---|---|
| 1 | Plan in config | `config.go`, `errors.go` |
| 2 | PayoutTerms | `payout.go`, `terminal.go`, `tests/payout_test.go` |
| 3 | Docs | `README.md`, `docs/ARCHITECTURE.md` |
| 4 | Spanish | every English `*.md` and `*.go` comment |
