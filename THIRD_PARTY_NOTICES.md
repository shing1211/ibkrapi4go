# Third-Party Notices

ibkrapi4go is licensed under the [Apache License 2.0](./LICENSE). It depends on
the following third-party Go modules. These are normal Go module dependencies
(not vendored); their licenses apply to their respective code.

| Module | License | Role |
|--------|---------|------|
| `github.com/coder/websocket` | ISC | WebSocket client (runtime) |
| `golang.org/x/time/rate` | BSD-3-Clause | Token-bucket rate limiting (runtime) |
| `github.com/oapi-codegen/runtime` | Apache-2.0 | Runtime helpers used by generated client (runtime, generated) |
| `github.com/stretchr/testify` | MIT | Test assertions (test-only) |
| `github.com/oapi-codegen/oapi-codegen/v2` | Apache-2.0 | Code generator (build-time only) |

## Generated code

Files matching `client/*.gen.go` are **generated** by `oapi-codegen` from the
Interactive Brokers OpenAPI specification and are marked `DO NOT EDIT`.

The Interactive Brokers OpenAPI specification is **not redistributed** in this
repository. It is fetched at build time and cached under `specs/`, which is
excluded via `.gitignore`. Generated code is a transformation of that
specification for interoperability; it is not a redistribution of Interactive
Brokers source code.

## Trademarks

"Interactive Brokers", "IBKR", and related marks are the property of their
respective owners. See [DISCLAIMER.md](./DISCLAIMER.md).
