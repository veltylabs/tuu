---
PLAN: "feat: tuu Push uses TerminalPush.WaitSeconds as the wait of that call"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 13185759063387103056
PR: https://github.com/veltylabs/tuu/pull/3
REVIEWED_COMMIT: eac4857f98a59569580cfbd93ed67f6835f5259d
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Publishing:** you publish only by OPENING THE PULL REQUEST (a `git push` from your sandbox does not reach GitHub). Submit after Stage 2 and keep committing.
> **Never ask a yes/no question about progress**; ask only when a signature in this plan is wrong, and name the symbol.
> **Before you submit:** `git status` must be clean and `git diff --stat main` must list the files of the stages. A pull request that carries only go.mod, go.sum or docs/PLAN.md is a failed delivery: `git add -A`, commit and submit again.
> No python, sed or regex over the source: edit file by file.

# Plan: `tuu` — the configurable wait of the card machine

## Why (read once)

`payment_manager` (main) added `WaitSeconds int64` to `TerminalPush`: how long the module lets the machine take to accept a charge (default 5 s, configurable by the app). The adapter must apply it as the timeout of that call; today every call uses `Config.TimeoutMs` (default 30 s), so a machine that is off makes the cashier wait 30 s instead of 5. `Status` keeps `Config.TimeoutMs`. `tuu` is a `Terminal`, not a `Gateway`: nothing else changes.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get github.com/veltylabs/payment_manager@main
go mod tidy
```
(`@main` because the tag is not cut yet; `gopush` rewrites it.) No `replace`.

## APIs used

| symbol | exact signature | where |
|---|---|---|
| `paymentmanager.TerminalPush.WaitSeconds` | `int64` (0 = use the adapter's default) | payment_manager `ports.go` |
| `(*Terminal).call` | `func (t *Terminal) call(method, path string, in model.Encodable, out model.Decodable) error` (uses `Timeout(t.cfg.TimeoutMs)`) | `api.go:20` |
| `tuu.Config.TimeoutMs` | `int` (0 = `DefaultTimeoutMs` = 30000) | `config.go` |

## Stage 1 — `api.go` and `terminal.go`

In `api.go` give `call` the timeout as a parameter and use it:

```go
func (t *Terminal) call(method, path string, timeoutMs int, in model.Encodable, out model.Decodable) error {
```
and `Timeout(t.cfg.TimeoutMs)` becomes `Timeout(timeoutMs)`; update the doc example to `err := t.call(methodPost, CreatePath, t.cfg.TimeoutMs, body, &out)`.

In `terminal.go`: `Push` computes and passes the wait:

```go
	waitMs := t.cfg.TimeoutMs
	if req.WaitSeconds > 0 {
		waitMs = int(req.WaitSeconds * 1000)
	}
	var out PaymentRequest
	if err := t.call(methodPost, CreatePath, waitMs, body, &out); err != nil {
```
and `Status` (and any other caller: `grep -n "t.call(" *.go`) passes `t.cfg.TimeoutMs`.

## Stage 2 — test (`tests/fake_tuu_test.go`; row in `docs/TESTS.md`)

Make the fake able to answer slowly: add `delay time.Duration` to `fakeTuu` (import `time`), and in the handler, first line: `time.Sleep(f.delay)`. Then add:

```go
func TestPush_UsesWaitSeconds(t *testing.T) {
	f := newFake(t)
	f.respond(201, `{"idempotencyKey":"p1","dteType":99,"status":"Pending","amount":30000,"deviceId":752,"description":"desc"}`)
	f.delay = 1500 * time.Millisecond
	term := f.terminal()
	req := paymentmanager.TerminalPush{RequestId: "p1", Device: "EXEMPT-DEVICE", Amount: 30000, Currency: paymentmanager.CurrencyCLP, Method: paymentmanager.MethodDebit, WaitSeconds: 1}
	if _, err := term.Push(req); err == nil {
		t.Fatal("a machine that answers after WaitSeconds must make Push fail")
	}
	req.WaitSeconds = 0 // the adapter default (30 s) waits for it
	if _, err := term.Push(req); err != nil {
		t.Fatalf("with the default wait the slow answer is accepted: %v", err)
	}
}
```
(the file imports `github.com/veltylabs/payment_manager` as `paymentmanager`: use that name.)

## Stage 3 — documentation (Spanish, veltylabs)

`README.md` and `docs/ARCHITECTURE.md`: one paragraph: «`Push` usa `WaitSeconds` de `payment_manager` como tiempo máximo de esa llamada; `Config.TimeoutMs` queda para el resto».

## Delivery check

`gotest` passes on every destination; `grep -n "t.cfg.TimeoutMs" api.go` → empty; the pull request is open. A tree that does not compile or whose tests did not run is a failed delivery.
