# AGENTS.md — veltylabs/tuu

<!-- webtyp:base — copied from devskills skill agents-md; do not edit here -->
## Ecosystem rules — read before any change

`docs/PLAN.md` is the spec: no design judgment; anything unclear is a question in the PR, not a guess.

- Public API: typed (no `any`/`interface{}`), explicit, minimal; never export a symbol only a test uses.
- One intent, one path: the old path is deleted in the same change. No silent fallback, `TODO`, commented code or stub.
- Fail at compile time, else a loud diagnostic — never silently. Zero values deny.
- A defect in a dependency is reported, never wrapped or patched here.
- No string literals in logic: named constants. The same logic never lives in two repos: import the package the plan names; if you find an existing one the plan does not name that already does what you are about to write, say so in the PR instead of writing it.
- `cmd/*/main.go`: parse flags, inject, print/exit only. stdout = data, stderr = logs, exit codes are the contract.
- WASM code: `webtyp/fmt`, `webtyp/time`, `webtyp/json` instead of stdlib; no `map`; `dom.Element` embedded by value.
- `css.go`, `svg.go`, `js.go`, `html.go` carry `//go:build !wasm`; no `ssr.go`, no `front.go`.
- Build tags: `wasm`/`!wasm` only when the API does not exist on that platform; `browser`/`!browser` when the file belongs to another binary One main per folder, no destination tag: `web/browser/main.go` → `web/public/browser.wasm`, `web/server/main.go` → `web/server/server`, `web/edge/main.go` → `web/edge/edge.wasm`.
- Code and keys in English; UI text is `lang.Text`, translations live in `lang.json`. Comments and docs: English in `webtyp/*`; in `veltylabs/*` **every README, `docs/*` file, diagram and code comment is in Spanish**, written for a reader with little technical background, and each exported symbol's comment shows a short usage example.
- Tests cover behaviour and domain rules, never the look (CSS, colours, markup order, class strings). In `tests/` (external package), real stack with fakes only at the edge; run `gotest`.
- Docs (`README.md`, `docs/*`) updated with the API; permanent docs never link `docs/PLAN.md`.
- Before the PR: `git diff --stat main` lists every file of the plan's stages, and nothing the plan did not order is deleted.
<!-- /webtyp:base -->

Notas de trabajo para agentes IA. Qué es TUU y sus APIs: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Este repo

Adaptador de **TUU** (máquinas de cobro de Haulmer) para los puertos de
[`github.com/veltylabs/payment_manager`](https://github.com/veltylabs/payment_manager): hoy `Terminal`
(cobro remoto a una máquina); después `Gateway` (pago en línea). Implementa el contrato; nunca lo redefine.
Si al contrato le falta algo, la corrección es un plan contra `payment_manager`. Solo lo importa la raíz
de composición de una app (`config/`), nunca un módulo `veltylabs/modules/*`.

Mismas reglas que [`github.com/veltylabs/transbank`](https://github.com/veltylabs/transbank) (su modelo):

- **Prohibido** `net/http`, `encoding/json`, `strings`, `strconv`, `errors`, `fmt` y `time` de la stdlib en
  código que no sea de prueba. HTTP → `webtyp.com/fetch`; JSON → `webtyp.com/json` sobre `model.Definition`
  generados con `ormc` (nunca editar `model_orm.go`); texto y errores → `webtyp.com/fmt`.
- **Sin `map`** ni `reflect`. Literales repetidos (rutas, cabeceras, estados de TUU) solo en constantes exportadas.
- **Sin secretos en el código ni en los logs**: la llave de API llega por `Config`.
- Todo en español salvo el código: README, `docs/*`, diagramas y comentarios, con ejemplo de uso en cada
  símbolo exportado.
- Pruebas en `tests/` contra un servidor falso (`httptest`), nunca contra TUU real.
