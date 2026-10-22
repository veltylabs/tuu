---
PLAN: "feat: TUU card terminal adapter (remote payment v2) for payment_manager.Terminal"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.

# Plan — `tuu.Terminal`: push an amount to a TUU card machine

## Why

A clinic charges at its till with a TUU card machine. `payment_manager` (v0.2.0) has a port `Terminal`: it pushes
the amount to a machine and asks for the result. This repo implements it against TUU's **remote payment v2** API,
so the cashier never types the amount. The online payment (`Gateway`, TUU Payment Intent) comes in a later plan.

Each machine has a fixed tax treatment: the clinic's issues an **exempt** document (no IVA), the pharmacy's an
**taxed** one (IVA 19 %). TUU needs it on every request (`dteType` and `extraData.exemptAmount`).

Language: identifiers, constants and error strings in English; **README, docs, diagrams and comments in Spanish**,
each exported symbol with a `// Ejemplo: …` line.

## Delivery rules

- **Commit after every stage.** Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash`.
- Never pause to ask permission or to report progress. When the Delivery check passes: push and open the PR. Stuck
  for 30 minutes: open the PR anyway, listing what fails.
- A PR that does not compile, or whose tests could not run, is a failed delivery.
- Never edit `model_orm.go` by hand: regenerate it with `ormc`.
- The code blocks below are the implementation: copy them.

## Tools and pinned versions

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
export GOPRIVATE='webtyp.com/*,github.com/veltylabs/*' GOFLAGS=-mod=mod
```

## Reference to copy

`github.com/veltylabs/transbank` is the model adapter: copy its `api.go` (the `call` function over
`webtyp.com/fetch` + `webtyp.com/json`) and the shape of `tests/fake_webpay_test.go` (an `httptest` server that
records requests). Rename `Webpay` → `Terminal`, `transbank:` → `tuu:` in messages, and the headers as below.

## TUU API facts (from developers.tuu.cl, 2026-10-22)

- Base URL: `https://integrations.payment.haulmer.com`. Header `X-API-Key: <key>`, `Content-Type: application/json`.
- `POST /RemotePayment/v2/Create`, body:
  `{"idempotencyKey":"p1","amount":1600,"device":"TJ44245N20440","description":"…","dteType":48,"paymentMethod":2,"extraData":{"exemptAmount":0,"sourceName":"veltylabs"}}`
  → 201 `{"idempotencyKey":"p1","dteType":48,"status":"Pending","amount":1600,"deviceId":752,"description":"…"}`.
- `GET /RemotePayment/v2/GetPaymentRequest/{idempotencyKey}` → 200
  `{"idempotencyKey":"p1","status":"Completed","sequenceNumber":"000000051934","amount":1600,"device":"TJ44245N20448","dteType":48,…}`.
- `status` values: `Pending`, `Sent`, `Processing`, `Canceled`, `Failed`, `Completed`.
- `dteType`: `48` taxed receipt, `99` exempt receipt. Exempt → `exemptAmount` = amount; taxed → `exemptAmount` = 0.
- `paymentMethod`: `1` credit, `2` debit.
- Errors: `{"error":true,"code":"MR-110","message":"…"}` with 400/401/404/429/500. 429 = more than 1 request per
  minute to the same machine.
- No authorization code nor last 4 digits are returned: those `TerminalResult` fields stay empty.

## APIs used (`github.com/veltylabs/payment_manager` v0.2.0)

```go
type Terminal interface {
	Provider() string
	Push(req TerminalPush) (TerminalPushed, error)
	Status(token string) (TerminalResult, error)
}
type TerminalPush struct{ RequestId, Device string; Amount int64; Currency, Method, Description string }
type TerminalPushed struct{ Token string }
type TerminalResult struct{ Outcome string; Amount int64; AuthorizationCode, CardLast4 string }
const OutcomePending, OutcomeAuthorized, OutcomeRejected, OutcomeCancelled = "pending", "authorized", "rejected", "cancelled"
const MethodDebit, MethodCredit = "debit", "credit"
const CurrencyCLP = "CLP"
```

If a shape differs in v0.2.0, stop and name it in the PR.

## Stage 1 — Module file (`go.mod`)

The repo was created without `gonew`, so it has no `go.mod`. Create it with exactly:

```
module github.com/veltylabs/tuu

go 1.26.8
```

then `go get github.com/veltylabs/payment_manager@v0.2.0 webtyp.com/fetch webtyp.com/fmt webtyp.com/json webtyp.com/model && go mod tidy`
(after Stage 2 adds code). **Commit** with Stage 2.

## Stage 2 — Config, errors, models (`config.go`, `errors.go`, `model.go`)

```go
// config.go
package tuu

// BaseURL es la API de integraciones de TUU (producción).
const BaseURL = "https://integrations.payment.haulmer.com"

// DefaultTimeoutMs es el tiempo máximo de cada llamada a TUU.
const DefaultTimeoutMs = 30000

// Device es una máquina TUU y el documento que emite.
// Ejemplo: tuu.Device{Serial: "TJ44245N20440", Exempt: true} // consultorio: boleta exenta
type Device struct {
	Serial string // número de serie que muestra la máquina
	Exempt bool   // true: documento exento (sin IVA); false: afecto (IVA 19 %)
}

// Config es la configuración de una cuenta TUU.
// Ejemplo: tuu.New(tuu.Config{APIKey: llave, Devices: []tuu.Device{{Serial: "TJ44245N20440", Exempt: true}}})
type Config struct {
	APIKey    string   // llave del Espacio de Trabajo (Pagos → Configuración → API); nunca en el código
	BaseURL   string   // vacío = BaseURL; otra solo en pruebas
	TimeoutMs int      // 0 = DefaultTimeoutMs
	Devices   []Device // las máquinas de esta cuenta
}
```

```go
// errors.go
var (
	ErrNoAPIKey      = fmt.Err("tuu: Config.APIKey is required")
	ErrUnknownDevice = fmt.Err("tuu: the device is not in Config.Devices")
	ErrCurrency      = fmt.Err("tuu: only CLP is supported")
	ErrMethod        = fmt.Err("tuu: method must be debit or credit")
	ErrUnknownStatus = fmt.Err("tuu: unknown payment request status")
	ErrBadResponse   = fmt.Err("tuu: unreadable response")
)
```

`model.go` (field names are TUU's JSON keys, camelCase on purpose):

```go
var ExtraDataModel = model.Definition{
	Name: "extra_data",
	Fields: model.Fields{
		{Name: "exemptAmount", Type: model.Int()},
		{Name: "sourceName", Type: model.Text()},
	},
}

var CreateRequestModel = model.Definition{
	Name: "create_request",
	Fields: model.Fields{
		{Name: "idempotencyKey", Type: model.Text()},
		{Name: "amount", Type: model.Int()},
		{Name: "device", Type: model.Text()},
		{Name: "description", Type: model.Text()},
		{Name: "dteType", Type: model.Int()},
		{Name: "paymentMethod", Type: model.Int()},
		{Name: "extraData", Type: model.Struct(&ExtraDataModel)},
	},
}

var PaymentRequestModel = model.Definition{ // respuesta de Create y de GetPaymentRequest
	Name: "payment_request",
	Fields: model.Fields{
		{Name: "idempotencyKey", Type: model.Text()},
		{Name: "status", Type: model.Text()},
		{Name: "amount", Type: model.Int()},
		{Name: "sequenceNumber", Type: model.Text()},
	},
}

var APIErrorModel = model.Definition{
	Name: "api_error",
	Fields: model.Fields{
		{Name: "code", Type: model.Text()},
		{Name: "message", Type: model.Text()},
	},
}
```

Run `go mod tidy && ormc`. If `ormc` refuses the camelCase names, stop and name it in the PR (do not rename the
JSON keys: TUU expects them). **Commit.**

## Stage 3 — `api.go` and `terminal.go`

`api.go`: copy `transbank/api.go`; constants

```go
const (
	CreatePath    = "/RemotePayment/v2/Create"
	GetPath       = "/RemotePayment/v2/GetPaymentRequest/"
	HeaderAPIKey  = "X-API-Key"
	SourceName    = "veltylabs"
	methodGet     = "GET"
	methodPost    = "POST"
)
```

`call(method, path string, in model.Encodable, out model.Decodable) error` sets `HeaderAPIKey`; a non-2xx answer
decodes `APIError` and returns `fmt.Errf("tuu: HTTP %d %s: %s", status, code, message)` (or `fmt.Errf("tuu: HTTP %d", status)`).

`terminal.go`:

```go
// Status de TUU y DTE.
const (
	StatusPending    = "Pending"
	StatusSent       = "Sent"
	StatusProcessing = "Processing"
	StatusCanceled   = "Canceled"
	StatusFailed     = "Failed"
	StatusCompleted  = "Completed"
	DteTaxed         = 48
	DteExempt        = 99
	MethodCreditCode = 1
	MethodDebitCode  = 2
	ProviderName     = "tuu"
)

// Terminal envía cobros a las máquinas TUU de una cuenta.
// Ejemplo: pm, _ := paymentmanager.New(db, paymentmanager.Deps{Terminal: tuu.New(cfg), Charges: …})
type Terminal struct{ cfg Config }

var _ paymentmanager.Terminal = (*Terminal)(nil)

// New crea el adaptador; no llama a TUU.
func New(cfg Config) *Terminal {
	if cfg.BaseURL == "" {
		cfg.BaseURL = BaseURL
	}
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = DefaultTimeoutMs
	}
	return &Terminal{cfg: cfg}
}

func (t *Terminal) Provider() string { return ProviderName }

// Push envía el monto a la máquina; el token es la llave de idempotencia (= id del pago).
func (t *Terminal) Push(req paymentmanager.TerminalPush) (paymentmanager.TerminalPushed, error) {
	if t.cfg.APIKey == "" {
		return paymentmanager.TerminalPushed{}, ErrNoAPIKey
	}
	if req.Currency != paymentmanager.CurrencyCLP {
		return paymentmanager.TerminalPushed{}, ErrCurrency
	}
	method := int64(MethodDebitCode)
	switch req.Method {
	case paymentmanager.MethodDebit:
	case paymentmanager.MethodCredit:
		method = MethodCreditCode
	default:
		return paymentmanager.TerminalPushed{}, ErrMethod
	}
	dev, ok := t.device(req.Device)
	if !ok {
		return paymentmanager.TerminalPushed{}, ErrUnknownDevice
	}
	dte, exempt := int64(DteTaxed), int64(0)
	if dev.Exempt {
		dte, exempt = DteExempt, req.Amount
	}
	body := &CreateRequest{
		IdempotencyKey: req.RequestId, Amount: req.Amount, Device: dev.Serial, Description: req.Description,
		DteType: dte, PaymentMethod: method, ExtraData: ExtraData{ExemptAmount: exempt, SourceName: SourceName},
	}
	var out PaymentRequest
	if err := t.call(methodPost, CreatePath, body, &out); err != nil {
		return paymentmanager.TerminalPushed{}, err
	}
	return paymentmanager.TerminalPushed{Token: out.IdempotencyKey}, nil
}

// Status consulta el cobro y lo traduce al resultado de payment_manager.
func (t *Terminal) Status(token string) (paymentmanager.TerminalResult, error) {
	var out PaymentRequest
	if err := t.call(methodGet, GetPath+token, nil, &out); err != nil {
		return paymentmanager.TerminalResult{}, err
	}
	res := paymentmanager.TerminalResult{Amount: out.Amount}
	switch out.Status {
	case StatusPending, StatusSent, StatusProcessing:
		res.Outcome = paymentmanager.OutcomePending
	case StatusCompleted:
		res.Outcome = paymentmanager.OutcomeAuthorized
	case StatusFailed:
		res.Outcome = paymentmanager.OutcomeRejected
	case StatusCanceled:
		res.Outcome = paymentmanager.OutcomeCancelled
	default:
		return paymentmanager.TerminalResult{}, ErrUnknownStatus
	}
	return res, nil
}

func (t *Terminal) device(serial string) (Device, bool) {
	for _, d := range t.cfg.Devices {
		if d.Serial == serial {
			return d, true
		}
	}
	return Device{}, false
}
```

(Use the exact Go field names `ormc` generated; if `ExtraData` is a pointer or has another name, adapt only the
literal, never the JSON.) **Commit.**

## Stage 4 — Tests (`tests/`, package `tests`, build tag `!wasm`)

A fake TUU (`httptest.NewServer`) that records method, path, `X-API-Key` and body, and answers a configured status
and body.

| Case | Setup / call | Expect |
|---|---|---|
| T1 | `Push` debit 30000 to exempt device | POST `CreatePath`, header key, body `dteType` 99, `exemptAmount` 30000, `paymentMethod` 2, `idempotencyKey` = RequestId; token = `p1` |
| T2 | `Push` credit to taxed device | `dteType` 48, `exemptAmount` 0, `paymentMethod` 1 |
| T3 | unknown device | `ErrUnknownDevice`, no request |
| T4 | USD | `ErrCurrency` |
| T5 | method `cash` | `ErrMethod` |
| T6 | no API key | `ErrNoAPIKey` |
| T7 | fake answers 429 `{"error":true,"code":"MR-180","message":"Payment Request Queue for the device is full"}` | error text contains `429` and `MR-180` |
| T8 | `Status` with `Completed` amount 30000 | `authorized`, 30000 |
| T9 | `Status` `Pending`/`Sent`/`Processing` | `pending` |
| T10 | `Status` `Failed` / `Canceled` | `rejected` / `cancelled` |
| T11 | `Status` `"Weird"` | `ErrUnknownStatus` |
| T12 | `Provider()` | `"tuu"` |

**Commit.**

## Stage 5 — Docs (Spanish)

- `README.md`: replace the STATUS note by "Getting Started" with the wiring example:

```go
pos := tuu.New(tuu.Config{
	APIKey:  secrets.Secret("TUU_API_KEY_CLINICA"),
	Devices: []tuu.Device{{Serial: "TJ44245N20440", Exempt: true}},
})
pm, err := paymentmanager.New(db, paymentmanager.Deps{
	Terminal: pos,
	Charges:  []paymentmanager.ChargeSource{{Module: "clinical_encounter", Reader: encuentros, Device: "TJ44245N20440"}},
})
```

  and one sentence: "la máquina debe estar en Modo Integración (menú de la app Pago)".
- `docs/ARCHITECTURE.md`: new section `## Adaptador de máquina (Terminal)` with the status mapping table and the
  dteType rule; `docs/diagrams/terminal.md`: `flowchart TD` Push → Status polling → outcome (no `subgraph`). Link it
  from `README.md`. **Commit.**

## Stage 6 — Spanish (only what is still in English)

Every `README.md`, `docs/*` file, diagram and code comment in Spanish; translate only what is still English; each
exported symbol's comment has a `// Ejemplo: …` line. **Commit.**

## Delivery check

```bash
go mod tidy && ormc && gotest
```

## Stages

| # | What | Files |
|---|---|---|
| 1 | Module file | `go.mod` |
| 2 | Config, errors, models | `config.go`, `errors.go`, `model.go`, `model_orm.go` (generated) |
| 3 | HTTP and Terminal | `api.go`, `terminal.go` |
| 4 | Tests | `tests/*` |
| 5 | Docs | `README.md`, `docs/ARCHITECTURE.md`, `docs/diagrams/terminal.md` |
| 6 | Spanish | every English `*.md` and `*.go` comment |
