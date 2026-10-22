package tuu

import "webtyp.com/fmt"

// Errores de TUU.
// Ejemplo: if err == tuu.ErrNoAPIKey { ... }
var (
	ErrNoAPIKey      = fmt.Err("tuu: Config.APIKey is required")
	ErrUnknownDevice = fmt.Err("tuu: the device is not in Config.Devices")
	ErrCurrency      = fmt.Err("tuu: only CLP is supported")
	ErrMethod        = fmt.Err("tuu: method must be debit or credit")
	ErrUnknownStatus = fmt.Err("tuu: unknown payment request status")
	ErrBadResponse   = fmt.Err("tuu: unreadable response")
	ErrNoPayoutPlan  = fmt.Err("tuu: Config.Payout is not set")
)
