package tuu

import (
	"github.com/veltylabs/payment_manager"
)

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
// Ejemplo: term := tuu.New(tuu.Config{APIKey: "key"})
func New(cfg Config) *Terminal {
	if cfg.BaseURL == "" {
		cfg.BaseURL = BaseURL
	}
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = DefaultTimeoutMs
	}
	return &Terminal{cfg: cfg}
}

// Provider devuelve el nombre del proveedor ("tuu").
// Ejemplo: name := term.Provider()
func (t *Terminal) Provider() string { return ProviderName }

// Push envía el monto a la máquina; el token es la llave de idempotencia (= id del pago).
// Ejemplo: pushed, err := term.Push(paymentmanager.TerminalPush{...})
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
// Ejemplo: res, err := term.Status("token123")
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
