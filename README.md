# tuu
<img src="docs/img/badges.svg">

`tuu` será el adaptador de **TUU** (las máquinas de cobro con tarjeta de la empresa chilena
Haulmer) para los puertos de cobro de [payment_manager](https://github.com/veltylabs/payment_manager),
igual que [transbank](https://github.com/veltylabs/transbank) lo es para Webpay. Un negocio que
cambie de procesador cambia de adaptador; los módulos de dominio no saben cuál está detrás.

## Comenzando

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

La máquina debe estar en Modo Integración (menú de la app Pago).

## Documentos

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — qué es TUU, sus APIs y qué falta preguntarle.
- [docs/diagrams/terminal.md](docs/diagrams/terminal.md) — diagrama de flujo de un cobro con máquina.
