//go:build !wasm

package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/veltylabs/payment_manager"
	"github.com/veltylabs/tuu"
	"webtyp.com/json"
)

// request es un pedido que recibió el servidor falso.
type request struct {
	method, path, apiKey, body string
}

// fakeTuu es un servidor TUU de prueba: guarda cada pedido y responde
// status/body configurados por la prueba.
type fakeTuu struct {
	srv      *httptest.Server
	requests []request
	status   int
	body     string
}

func newFake(t *testing.T) *fakeTuu {
	t.Helper()
	f := &fakeTuu{status: 200, body: `{}`}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f.requests = append(f.requests, request{r.Method, r.URL.Path, r.Header.Get(tuu.HeaderAPIKey), string(b)})
		w.WriteHeader(f.status)
		_, _ = w.Write([]byte(f.body))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeTuu) respond(status int, body string) { f.status, f.body = status, body }

func (f *fakeTuu) terminal() paymentmanager.Terminal {
	return tuu.New(tuu.Config{
		APIKey: "test-key",
		BaseURL: f.srv.URL,
		Devices: []tuu.Device{
			{Serial: "EXEMPT-DEVICE", Exempt: true},
			{Serial: "TAXED-DEVICE", Exempt: false},
		},
	})
}

func (f *fakeTuu) last(t *testing.T) request {
	t.Helper()
	if len(f.requests) == 0 {
		t.Fatal("no request reached the fake TUU")
	}
	return f.requests[len(f.requests)-1]
}

func TestTerminalPush(t *testing.T) {
	// T1: `Push` debit 30000 to exempt device
	t.Run("T1_Exempt_Debit", func(t *testing.T) {
		f := newFake(t)
		f.respond(201, `{"idempotencyKey":"p1","dteType":99,"status":"Pending","amount":30000,"deviceId":752,"description":"desc"}`)
		term := f.terminal()

		req := paymentmanager.TerminalPush{
			RequestId: "p1",
			Device:    "EXEMPT-DEVICE",
			Amount:    30000,
			Currency:  paymentmanager.CurrencyCLP,
			Method:    paymentmanager.MethodDebit,
			Description: "desc",
		}

		res, err := term.Push(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.Token != "p1" {
			t.Errorf("got token %q, want p1", res.Token)
		}

		last := f.last(t)
		if last.method != "POST" || last.path != tuu.CreatePath || last.apiKey != "test-key" {
			t.Errorf("wrong request basics: %+v", last)
		}

		var body tuu.CreateRequest
		if err := json.Decode([]byte(last.body), &body); err != nil {
			t.Fatal(err)
		}

		if body.DteType != tuu.DteExempt || body.PaymentMethod != tuu.MethodDebitCode || body.IdempotencyKey != "p1" || body.ExtraData.ExemptAmount != 30000 {
			t.Errorf("wrong body: %+v", body)
		}
	})

	// T2: `Push` credit to taxed device
	t.Run("T2_Taxed_Credit", func(t *testing.T) {
		f := newFake(t)
		f.respond(201, `{"idempotencyKey":"p2"}`)
		term := f.terminal()

		req := paymentmanager.TerminalPush{
			RequestId: "p2",
			Device:    "TAXED-DEVICE",
			Amount:    15000,
			Currency:  paymentmanager.CurrencyCLP,
			Method:    paymentmanager.MethodCredit,
		}

		_, err := term.Push(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		last := f.last(t)
		var body tuu.CreateRequest
		if err := json.Decode([]byte(last.body), &body); err != nil {
			t.Fatal(err)
		}

		if body.DteType != tuu.DteTaxed || body.PaymentMethod != tuu.MethodCreditCode || body.ExtraData.ExemptAmount != 0 {
			t.Errorf("wrong body: %+v", body)
		}
	})

	// T3: unknown device
	t.Run("T3_UnknownDevice", func(t *testing.T) {
		f := newFake(t)
		term := f.terminal()

		req := paymentmanager.TerminalPush{
			Device:   "UNKNOWN",
			Amount:   1000,
			Currency: paymentmanager.CurrencyCLP,
			Method:   paymentmanager.MethodDebit,
		}

		_, err := term.Push(req)
		if err != tuu.ErrUnknownDevice {
			t.Errorf("got error %v, want %v", err, tuu.ErrUnknownDevice)
		}
		if len(f.requests) > 0 {
			t.Errorf("unexpected request")
		}
	})

	// T4: USD
	t.Run("T4_USD", func(t *testing.T) {
		term := newFake(t).terminal()
		_, err := term.Push(paymentmanager.TerminalPush{Currency: "USD"})
		if err != tuu.ErrCurrency {
			t.Errorf("got error %v, want ErrCurrency", err)
		}
	})

	// T5: method cash
	t.Run("T5_MethodCash", func(t *testing.T) {
		term := newFake(t).terminal()
		_, err := term.Push(paymentmanager.TerminalPush{Currency: paymentmanager.CurrencyCLP, Method: "cash"})
		if err != tuu.ErrMethod {
			t.Errorf("got error %v, want ErrMethod", err)
		}
	})

	// T6: no API key
	t.Run("T6_NoAPIKey", func(t *testing.T) {
		term := tuu.New(tuu.Config{})
		_, err := term.Push(paymentmanager.TerminalPush{})
		if err != tuu.ErrNoAPIKey {
			t.Errorf("got error %v, want ErrNoAPIKey", err)
		}
	})

	// T7: fake answers 429
	t.Run("T7_Error429", func(t *testing.T) {
		f := newFake(t)
		f.respond(429, `{"error":true,"code":"MR-180","message":"Payment Request Queue for the device is full"}`)
		term := f.terminal()

		_, err := term.Push(paymentmanager.TerminalPush{
			Device:   "EXEMPT-DEVICE",
			Amount:   1000,
			Currency: paymentmanager.CurrencyCLP,
			Method:   paymentmanager.MethodDebit,
		})

		if err == nil {
			t.Fatal("expected error")
		}

		msg := err.Error()
		if !contains(msg, "429") || !contains(msg, "MR-180") {
			t.Errorf("got error %q, missing 429 or MR-180", msg)
		}
	})
}

func TestTerminalStatus(t *testing.T) {
	// T8: `Status` with `Completed` amount 30000
	t.Run("T8_Completed", func(t *testing.T) {
		f := newFake(t)
		f.respond(200, `{"status":"Completed","amount":30000}`)
		term := f.terminal()

		res, err := term.Status("p1")
		if err != nil {
			t.Fatal(err)
		}

		if res.Outcome != paymentmanager.OutcomeAuthorized || res.Amount != 30000 {
			t.Errorf("got %+v, want authorized 30000", res)
		}
	})

	// T9: `Status` `Pending`/`Sent`/`Processing`
	t.Run("T9_Pending", func(t *testing.T) {
		statuses := []string{tuu.StatusPending, tuu.StatusSent, tuu.StatusProcessing}
		for _, s := range statuses {
			f := newFake(t)
			f.respond(200, `{"status":"`+s+`"}`)
			term := f.terminal()
			res, err := term.Status("p1")
			if err != nil {
				t.Fatal(err)
			}
			if res.Outcome != paymentmanager.OutcomePending {
				t.Errorf("got outcome %q, want pending for status %q", res.Outcome, s)
			}
		}
	})

	// T10: `Status` `Failed` / `Canceled`
	t.Run("T10_Failed_Canceled", func(t *testing.T) {
		// Failed -> rejected
		f := newFake(t)
		f.respond(200, `{"status":"Failed"}`)
		term := f.terminal()
		res, err := term.Status("p1")
		if err != nil || res.Outcome != paymentmanager.OutcomeRejected {
			t.Errorf("Failed: got %v, err %v", res.Outcome, err)
		}

		// Canceled -> cancelled
		f = newFake(t)
		f.respond(200, `{"status":"Canceled"}`)
		term = f.terminal()
		res, err = term.Status("p1")
		if err != nil || res.Outcome != paymentmanager.OutcomeCancelled {
			t.Errorf("Canceled: got %v, err %v", res.Outcome, err)
		}
	})

	// T11: `Status` `"Weird"`
	t.Run("T11_Weird", func(t *testing.T) {
		f := newFake(t)
		f.respond(200, `{"status":"Weird"}`)
		term := f.terminal()
		_, err := term.Status("p1")
		if err != tuu.ErrUnknownStatus {
			t.Errorf("got error %v, want ErrUnknownStatus", err)
		}
	})
}

func TestTerminalProvider(t *testing.T) {
	// T12: `Provider()`
	term := newFake(t).terminal()
	if p := term.Provider(); p != "tuu" {
		t.Errorf("got %q, want tuu", p)
	}
}

func contains(s, substr string) bool {
	// Simple string check since we can't use strings
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
