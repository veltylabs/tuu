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
