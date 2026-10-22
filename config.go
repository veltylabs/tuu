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

// PayoutPlan es el plan de abono contratado con TUU para una cuenta. Los valores salen del Espacio de Trabajo de TUU.
// Ejemplo: tuu.PayoutPlan{PercentBasisPoints: 149, FeeTaxBasisPoints: 1900, BusinessDays: 2} // porcentual 1,49 %, abono a 2 días
// Ejemplo: tuu.PayoutPlan{PercentBasisPoints: 79, FixedFee: 65, FeeTaxBasisPoints: 1900, BusinessDays: 2} // mixta 0,79 % + $65
type PayoutPlan struct {
	PercentBasisPoints int64 // comisión porcentual en puntos base (149 = 1,49 %)
	FixedFee           int64 // comisión fija por venta en pesos (mixta); 0 en la porcentual
	FeeTaxBasisPoints  int64 // IVA sobre la comisión (1900 = 19 %)
	BusinessDays       int   // días hábiles bancarios hasta el depósito (0 = al instante)
}

// Config es la configuración de una cuenta TUU.
// Ejemplo: tuu.New(tuu.Config{APIKey: llave, Devices: []tuu.Device{{Serial: "TJ44245N20440", Exempt: true}}})
type Config struct {
	APIKey    string      // llave del Espacio de Trabajo (Pagos → Configuración → API); nunca en el código
	BaseURL   string      // vacío = BaseURL; otra solo en pruebas
	TimeoutMs int         // 0 = DefaultTimeoutMs
	Devices   []Device    // las máquinas de esta cuenta
	Payout    *PayoutPlan // nil: el adaptador no declara abono (payment_manager registra el cobro sin fecha ni comisión)
}
