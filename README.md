# tuu

`tuu` será el adaptador de **TUU** (las máquinas de cobro con tarjeta de la empresa chilena
Haulmer) para los puertos de cobro de [payment_manager](https://github.com/veltylabs/payment_manager),
igual que [transbank](https://github.com/veltylabs/transbank) lo es para Webpay. Un negocio que
cambie de procesador cambia de adaptador; los módulos de dominio no saben cuál está detrás.

> STATUS (remove this note when the first adapter release lands): este repo hoy solo
> documenta. Todavía no hay código: las decisiones de diseño están abiertas en
> `webtyp/docs/MONEY_FLOWS_MASTER_PLAN.md` (P21–P23: boleta en efectivo y en línea, procesador en línea).

## Getting Started

1. **Entender qué ofrece TUU a un programador.** Lee [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md):
   qué es Haulmer, las tres APIs de TUU (cobro en línea por enlace, cobro remoto a una máquina,
   integración entre apps Android), sus límites y lo que la documentación oficial no responde.
2. **Verlo funcionando localmente.** Aún no se puede: no hay código.
3. **Desarrollar el adaptador.** Espera a que cierren las preguntas del master; el contrato
   lo define `payment_manager`, nunca este repo.

## Documentos

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — qué es TUU, sus APIs y qué falta preguntarle.
