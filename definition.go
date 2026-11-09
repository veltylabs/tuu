package tuu

import "webtyp.com/model"

// ExtraDataModel define los datos extra para un cobro.
// Ejemplo: var out tuu.ExtraData
var ExtraDataModel = model.Definition{
	Name: "extra_data",
	Fields: model.Fields{
		{Name: "exemptAmount", Type: model.Int()},
		{Name: "sourceName", Type: model.Text()},
	},
}

// CreateRequestModel define la petición de cobro a TUU.
// Ejemplo: var req tuu.CreateRequest
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

// PaymentRequestModel define la respuesta de Create y GetPaymentRequest de TUU.
// Ejemplo: var res tuu.PaymentRequest
var PaymentRequestModel = model.Definition{ // respuesta de Create y de GetPaymentRequest
	Name: "payment_request",
	Fields: model.Fields{
		{Name: "idempotencyKey", Type: model.Text()},
		{Name: "status", Type: model.Text()},
		{Name: "amount", Type: model.Int()},
		{Name: "sequenceNumber", Type: model.Text()},
	},
}

// APIErrorModel define un error de la API de TUU.
// Ejemplo: var err tuu.APIError
var APIErrorModel = model.Definition{
	Name: "api_error",
	Fields: model.Fields{
		{Name: "code", Type: model.Text()},
		{Name: "message", Type: model.Text()},
	},
}
