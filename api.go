package tuu

import (
	"webtyp.com/fetch"
	"webtyp.com/fmt"
	"webtyp.com/json"
	"webtyp.com/model"
)

const (
	CreatePath    = "/RemotePayment/v2/Create"
	GetPath       = "/RemotePayment/v2/GetPaymentRequest/"
	HeaderAPIKey  = "X-API-Key"
	SourceName    = "veltylabs"
	methodGet     = "GET"
	methodPost    = "POST"
)

// call hace un pedido a la API y espera la respuesta. in == nil: sin cuerpo.
func (t *Terminal) call(method, path string, in model.Encodable, out model.Decodable) error {
	url := t.cfg.BaseURL + path
	var req *fetch.Request
	switch method {
	case methodPost:
		req = fetch.Post(url)
	default:
		req = fetch.Get(url)
	}
	req.ContentTypeJSON().
		Header(HeaderAPIKey, t.cfg.APIKey).
		Timeout(t.cfg.TimeoutMs)
	if in != nil {
		var body []byte
		_ = json.Encode(in, &body) // los modelos generados no fallan al codificar
		req.Body(body)
	}

	done := make(chan error, 1)
	req.Send(func(res *fetch.Response, err error) {
		if err != nil { // C1
			done <- fmt.Err("tuu: network:", err.Error())
			return
		}
		if res.Status < 200 || res.Status > 299 { // C2
			var apiErr APIError
			if json.Decode(res.Body(), &apiErr) == nil && apiErr.Message != "" { // C3
				done <- fmt.Errf("tuu: HTTP %d %s: %s", res.Status, apiErr.Code, apiErr.Message)
				return
			}
			done <- fmt.Errf("tuu: HTTP %d", res.Status)
			return
		}
		if json.Decode(res.Body(), out) != nil { // C4
			done <- ErrBadResponse
			return
		}
		done <- nil
	})
	return <-done
}
