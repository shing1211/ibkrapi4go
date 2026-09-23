# Phase E — Docs/DX — Detailed Plan

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Purpose:** Bring documentation to truth: fix version/spec drift, wrong error
  docs, a broken design-check tool, a leaking generated type, and add missing
  DX material (gateway setup, permissions, examples, streaming docs, README
  architecture diagram).

**Note:** E8 depends on D7. E1–E7, E10 are independent and can run in Wave 2/3.

---

## E1 — Fix version/spec drift

- **Role:** docs · **Size:** S · **Deps:** — · **Wave:** 2

**Objective:**
- `pkg/ibkr/doc.go:47` says `v1.0.0` while `v1.0.1` is tagged.
- `README.md:56` and `docs/ROADMAP.md:255-256` still say v1.0.0.
- Spec version says `v2.39.0` in `README.md:6,64` (+ 5 translations),
  `docs/SPEC.md:4`, `docs/RELEASING.md:76`, but `scripts/check_spec_version.py:17`
  pins `v2.40.0` and `CHANGELOG.md:40` confirms v2.40.

**Files:** `pkg/ibkr/doc.go`, `README.md` + 5 translations, `docs/SPEC.md`,
`docs/ROADMAP.md`, `docs/RELEASING.md`, and the README `Last synced` banners.

**Approach:** Set `doc.go` Version to `v1.0.1`; regenerate `docs/SPEC.md`
(the repo has `make docs-spec` if present — verify); update all spec references
to v2.40.0; bump translation banners.

**Acceptance:** No stale version/spec references; `check_links.py` + `check_i18n.py`
pass.

**Verification:** `python scripts/check_links.py` · `python scripts/check_i18n.py`

---

## E2 — Fix wrong error docs

- **Role:** docs · **Size:** S · **Deps:** — · **Wave:** 2

**Objective:** `docs/ERRORS.md:46-47` says `ErrStreamDisconnected`/`ErrStreamReconnected`
are "still available but deprecated"; they were removed (`MIGRATION.md:120-127`,
`CHANGELOG.md:63`) and are absent from `pkg/ibkr/errors.go`.

**Files:** `docs/ERRORS.md`

**Approach:** Remove the stale entries; ensure the documented error list matches
`pkg/ibkr/errors.go`.

**Acceptance:** ERRORS.md matches code exactly.

**Verification:** manual diff vs `pkg/ibkr/errors.go`

---

## E3 — Fix `scripts/check_design/main.go`

- **Role:** backend · **Size:** S/M · **Deps:** — · **Wave:** 2

**Objective:** The tool is a **self-referential no-op**:
- Transport check parses `internal/transport.go` into `actualOrder`/`deduped`
  (lines 69-126) then never uses them; assertion only checks the doc against
  hard-coded keywords.
- Manager-count check: `prefix := managerType + ")"` (line 310) is compared to
  `ident.Name` which never contains `)`; `extractDocCount` (340) expects
  `|Manager|` but the doc uses backticks/spaces → returns 0 and `continue`s.

**Files:** `scripts/check_design/main.go`, possibly `docs/design/01-transport.md`,
`docs/design/03-managers.md`

**Approach:** Make the transport check compare the real middleware order parsed
from code vs the doc; fix manager counting (strip backticks, fix receiver
comparison). Keep it wired into `make docs-check`.

**Acceptance:** Introducing a deliberate drift (e.g. reorder a middleware in the
doc, or change a manager method count) makes the tool fail.

**Verification:** `go run ./scripts/check_design` (clean pass + injected-failure check)

---

## E4 — Remove leaking generated type `CreateSessionRaw`

- **Role:** backend · **Size:** S · **Deps:** — · **Wave:** 2

**Objective:** `pkg/ibkr/rest_sso.go:160` `CreateSessionRaw` returns generated
`*client.CreateSsoSessionsResponse`, violating
`docs/design/04-generated-wrapping.md:10-14`.

**Files:** `pkg/ibkr/rest_sso.go` (+ any callers), `docs/design/04-generated-wrapping.md`

**Approach:** Remove the method or return a public SDK type; if it must expose
raw, relocate to `internal/`.

**Acceptance:** No generated `client.*` type appears in a public `pkg/ibkr`
signature.

**Verification:** `go build ./...`; grep public signatures

---

## E5 — Add `docs/GATEWAY-SETUP.md` + `docs/PERMISSIONS.md`

- **Role:** docs · **Size:** S · **Deps:** — · **Wave:** 3

**Objective:** No gateway-setup doc (README links externally only) and no
permissions/entitlements doc (only a passing mention in `docs/STREAMING.md:110`).

**Files:** Create `docs/GATEWAY-SETUP.md`, `docs/PERMISSIONS.md`; link from
`README.md` Repository Docs table (+ translations).

**Approach:**
- GATEWAY-SETUP: install/run the Client Portal Gateway, browser auth/2FA,
  session lifecycle, `WithGatewayURL`/`WithInsecureSkipVerify` config, TLS notes.
- PERMISSIONS: trading permissions, market-data entitlements, delayed data,
  read-only accounts, subscription limits.

**Acceptance:** Docs exist, are linked, and `check_links.py` passes.

**Verification:** `python scripts/check_links.py`

---

## E6 — Fix misleading live examples

- **Role:** docs · **Size:** S · **Deps:** — · **Wave:** 3

**Objective:**
- Live examples require `IBKR_USERNAME`/`IBKR_PASSWORD` but never pass them to
  `NewClient` (`examples/live-portfolio/main.go:35`,
  `examples/options-chain/main.go:41`, `examples/multi-account/main.go:46`).
- `examples/live/oauth2-flow.go:197-203` `forceRefreshToken` doesn't force a refresh.
- `examples/options-chain/main.go:70-155` bypasses SDK managers (raw HTTP).

**Files:** the affected example files + `examples/README.md`

**Approach:** Remove unused credential requirements (the gateway is
pre-authenticated), make `forceRefreshToken` actually force a refresh or rename
it, and use SDK managers in options-chain where possible (or document why raw HTTP).

**Acceptance:** Examples run against the mock/live as documented; no dead env vars.

**Verification:** `go build ./...`; manual run vs mock

---

## E7 — Add cancellation + reconciliation examples

- **Role:** docs · **Size:** S · **Deps:** — · **Wave:** 3

**Objective:** `TradeManager.Cancel` exists (`pkg/ibkr/trade.go:252`) and the CLI
exposes it (`cmd/ibkr/orders.go:205`), but no `examples/` demo; reconciliation is
documented only (`docs/design/09:67-75`).

**Files:** Create `examples/mock/cancel-order/main.go` (or similar) +
`examples/mock/reconcile-open-orders/main.go`; update `examples/README.md`.

**Approach:** Mock-gateway examples (no creds) demonstrating cancel and
open-order reconciliation.

**Acceptance:** Both compile and run against the mock; README index updated.

**Verification:** `go build ./...`; `go run ./examples/...`

---

## E8 — Document streaming (incl. account/portfolio from D7)

- **Role:** docs · **Size:** S · **Deps:** D7 · **Wave:** 4

**Objective:** `docs/STREAMING.md` must cover the new account/portfolio
streaming capability shipped in D7 (subscription API, typed events, limits,
reconnect/gap behavior).

**Files:** `docs/STREAMING.md`, `examples/README.md` (link a demo if added)

**Approach:** Document the new subscribe API, event types, and operational notes;
update the "scope" section to reflect that account/portfolio streaming is now
supported.

**Acceptance:** STREAMING.md matches the shipped API; `check_links.py` passes.

**Verification:** `python scripts/check_links.py`

---

## E10 — README ASCII architecture diagram

- **Role:** docs · **Size:** S · **Deps:** — · **Wave:** 3

**Objective:** `README.md:140-150` has only a directory tree; real diagrams live
in `docs/ARCHITECTURE.md`.

**Files:** `README.md` (+ 5 translations if the switcher/banner rules require)

**Approach:** Add a top-level ASCII flow (NewClient → Session → Managers →
transport chain) mirroring `docs/ARCHITECTURE.md:110-141`. Follow the
translation lockstep rule (AGENTS.md hard rule 8, TRANSLATING.md).

**Acceptance:** Diagram present; translations' "Last synced" bumped;
`check_i18n.py` passes.

**Verification:** `python scripts/check_i18n.py`

---

## Phase E Gate

`check_links.py` · `check_i18n.py` · `go run ./scripts/check_design` all pass;
no stale version/spec/error references; new docs linked from README.
