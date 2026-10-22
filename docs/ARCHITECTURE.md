# Arquitectura de `tuu`

Este documento explica qué es TUU, qué puede hacer un programa con sus máquinas y qué no
está documentado. Existe porque la clínica Monjitas cobra hoy con dos máquinas TUU (una para
las consultas, otra para la farmacia) y queremos que el sistema cobre sin papelitos ni montos
digitados a mano. Fuente: el portal oficial [developers.tuu.cl](https://developers.tuu.cl),
leído el 2026-10-22.

## Qué es Haulmer y qué es TUU

**Haulmer** es una empresa chilena (Curicó) que vende servicios para pymes: facturación
electrónica (**OpenFactura**), firma digital (Chilefirmas), hosting y servidores. **TUU** es su
línea de **máquinas de pago** (terminales POS: el aparato donde el cliente pasa o acerca la
tarjeta). Son equipos Android (Sunmi y Kozen) con la app de pago de TUU instalada.

Lo que importa para nosotros: la máquina TUU no solo cobra, también **emite el documento
tributario**. En un pago con tarjeta entrega un *comprobante de pago electrónico* (código SII
48), que el SII acepta como boleta; Haulmer lo envía al SII por el comercio. En efectivo, la
máquina emite una boleta electrónica normal.

## Las tres formas de hablar con TUU

Ninguna es Bluetooth ni red local. Las dos primeras son HTTP contra la nube de Haulmer; la
máquina recibe las órdenes por su propia conexión a internet (WiFi o 4G).

### 1. Payment Intent — cobro en línea por enlace

El cliente paga **en su propio teléfono** (tarjeta escrita, con 3-D Secure). Es el que sirve
para mostrar un **código QR** con el enlace de pago.

- Integración: `https://frontend-api.payment.haulmer.dev/v1/payment` · Producción:
  `https://core.payment.haulmer.com/api/v1/payment`.
- Autenticación: cada solicitud lleva `x_signature`, un HMAC-SHA256 de todos los campos `x_*`
  ordenados (la llave secreta nunca sale del servidor).
- Campos obligatorios: cuenta, monto (solo CLP), referencia única, nombre, apellido, correo y
  teléfono del cliente, nombre de la tienda y tres URL: `x_url_callback` (aviso servidor a
  servidor, **la fuente de verdad**), `x_url_complete` y `x_url_cancel` (vuelta del navegador).
- Resultado (`x_result`): `completed`, `failed`, `pending`. El aviso se reintenta hasta 10 veces
  si nuestro servidor no responde 200 → **el servidor debe ser alcanzable desde internet**.
- Hay ambiente de pruebas con tarjetas de prueba.

Se parece a Webpay (crear → redirigir → confirmar), así que cabe en el puerto `Gateway` de
`payment_manager`, salvo que la confirmación llega por aviso firmado y no por un `Commit`.

### 2. Pago Remoto — mandar el monto a una máquina

Nuestro servidor le dice a **una máquina concreta** (por su número de serie) «cobra $X»; la
máquina se despierta y el cliente pasa su tarjeta **en esa máquina**.

- Base `https://integrations.payment.haulmer.com`, cabecera `X-API-Key` (una por comercio,
  se saca en Espacio de Trabajo → Pagos → Configuración → API).
- `POST /RemotePayment/v2/Create` y `GET /RemotePayment/v2/GetPaymentRequest/:idempotency-key`.
- Medio: solo crédito (1) o débito (2) — **no efectivo**. Campo `dteType` elige el documento y
  `extradata.exemptAmount` el monto exento: igual al monto en un documento exento, menor en uno afecto.
  En la clínica Monjitas la máquina del consultorio es **exenta** y la de la farmacia **afecta**: el
  adaptador guarda ese tratamiento por número de serie.
- Estados: 0 pendiente, 1 enviado, 2 cancelado, 3 procesando, 4 fallido, 5 completado. Sin
  aviso documentado: se consulta (polling).
- Límites: 1 solicitud por minuto por máquina (si no, HTTP 429), máximo 5 pendientes.
- La máquina debe estar en «Modo Integración» (menú de la app Pago).

### 3. Integración entre aplicaciones — app propia dentro de la máquina

Una app Android instalada **en la misma máquina** abre la app de TUU con un `Intent`
(`com.haulmer.paymentapp`) y recibe el resultado. Exige registrarse como partner y publicar
la app en la tienda de Sunmi o Kozen. No nos sirve: el sistema es web.

## Lo que la documentación no responde (preguntar a TUU)

1. ¿El comprobante de un pago por **Payment Intent** llega al SII como el de la máquina? Por
   ley (art. 54 Ley IVA, Res. SII 176/2020) el comprobante de cualquier pago electrónico vale
   como boleta, así que no se emite otra; falta confirmar que Haulmer lo informe igual.
2. ¿Payment Intent usa la misma cuenta y comisión que la máquina, o es un producto aparte?
3. ¿Con la cuenta TUU se puede emitir la **boleta de un pago en efectivo** por API (OpenFactura), y con qué plan?
4. ¿Payment Intent devuelve la URL de pago en la respuesta? (la página no lo muestra).

## Adaptador de máquina (Terminal)

La respuesta de estado de TUU se mapea así al resultado de `payment_manager`:

- `Pending`, `Sent`, `Processing` → `pending`
- `Completed` → `authorized`
- `Failed` → `rejected`
- `Canceled` → `cancelled`

El `dteType` se envía como `48` (afecto) o `99` (exento) según la configuración de la máquina (`Device.Exempt`). Si es exento, el `exemptAmount` es igual al monto total; si es afecto, es 0.

## Abono y comisión

TUU deposita los cobros con tarjeta restando su comisión y el IVA de la comisión. Un plan
de 1,49 %, IVA 19 % y depósito en 2 días significa que de $30.000, la comisión es $447,
el IVA de la comisión es $85, y el abono llega en 2 días hábiles.
