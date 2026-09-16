# Architecture Decision Records

Short, dated records of decisions that shape the project. Each records the
context, the decision, and its consequences. Superseding a decision means adding
a new ADR, not editing the old one.

| ADR | Title | Status |
|-----|-------|--------|
| [0001](./0001-two-api-surfaces.md) | Treat the CPAPI and IB REST surfaces as distinct | Accepted |
| [0002](./0002-auth-models.md) | Two auth models: `ssoBearer` and `oauth2Bearer` | Accepted |
| [0003](./0003-openapi-codegen.md) | Generate types/client with oapi-codegen + spec patching | Accepted |
| [0004](./0004-minimal-dependencies.md) | Minimal dependency set | Accepted |
| [0005](./0005-v1-scope.md) | v1 targets the CPAPI surface only | Accepted |
| [0006](./0006-context-channel-api.md) | Context-aware, channel-based API; `coder/websocket` | Accepted |
| [0007](./0007-license-apache-2.0.md) | License under Apache-2.0 with DCO | Accepted |
| [0008](./0008-numeric-precision.md) | Money/quantity are never `float64` | Accepted |
| [0009](./0009-no-auto-retry-orders.md) | Never auto-retry order mutations | Accepted |
| [0010](./0010-naming.md) | Project naming and module path | Accepted |
| [0010](./0010-test-dependencies.md) | Test-only dependencies (`goleak`) | Accepted |
| [0011](./0011-oauth2-surface.md) | OAuth2 / IB REST surface | Accepted |

Template:

```markdown
# NNNN — Title
- Status: Proposed | Accepted | Superseded by NNNN
- Date: YYYY-MM-DD

## Context
## Decision
## Consequences
```
