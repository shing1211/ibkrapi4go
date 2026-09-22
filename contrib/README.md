# contrib — Community Contributions

This directory contains supplementary modules that extend the core SDK without
adding runtime dependencies to `github.com/shing1211/ibkrapi4go`. They are
versioned in lock-step with the core module.

## Stability: Experimental

Contrib modules are **experimental**. Their APIs may change in any release.
They are not covered by the stability contract in `docs/STABILITY.md`. A
module's stability is noted in its own documentation.

## Modules

### `otel` — OpenTelemetry Bridge

Exports all SDK metrics to OpenTelemetry instruments. See
[`otel/go.mod`](./otel/go.mod) for the OpenTelemetry dependency.

**Module**: `github.com/shing1211/ibkrapi4go/contrib/otel`

```go
import (
    "go.opentelemetry.io/otel"
    otelbridge "github.com/shing1211/ibkrapi4go/contrib/otel"
)

meter := otel.Meter("my-app")
metrics := otelbridge.New(meter)

cli, _ := ibkr.NewClient(
    ibkr.WithGatewayURL("https://localhost:5000"),
    ibkr.WithMetrics(metrics),
)
```

See [`otel/otel.go`](./otel/otel.go) for the full API.

## Adding a contrib module

New contrib modules require an ADR under `docs/adr/` describing:
- What problem the module solves
- Why it is not included in the core SDK (e.g., extra dependencies, niche use case)
- The versioning policy (experimental, must track core module)

Each module must have its own `go.mod` and maintain compatibility with the
corresponding core SDK version.
