# Proposal

## Why

The current `ibkrapi4go` SDK provides a functionally correct foundation for interacting with the IBKR API. However, to qualify as a reliable, production-grade tool for high-stakes financial applications (e.g., HFT, mission-critical trading), it must evolve beyond functional correctness to ensure **deterministic performance, absolute state consistency, and catastrophic failure prevention.** This change proposes the architectural roadmap to achieve these enterprise-level requirements.

## What Changes

This change will focus on hardening the SDK's core transport and interaction layers to meet industry benchmarks for financial services resilience. The work is focused purely on architectural definition in this phase.

*   **NEW CAPABILITIES**: New capabilities will be introduced to serve as durable contracts for the required system behavior:
    *   `execution-integrity`: Ensures that every order mutation is handled exactly once and that all financial states are auditable.
    *   `transport-resilience`: Implements advanced, state-aware network fault tolerance, moving beyond simple retries.
*   **MODIFIED CAPABILITIES**: Existing capabilities will be modified (via delta specs) to incorporate safety mechanisms:
    *   `client-gateway`: Modifications are required to support cryptographic signing of mutations and to manage a global safety state (Kill Switch).

## Capabilities

### New Capabilities
- `execution-integrity`: The framework for guaranteed, auditable transaction processing. This governs idempotency, sequence tracking, and state validation.
- `transport-resilience`: The framework for making the underlying connection (REST/WS) self-healing and predictable under extreme network and broker throttling.

### Modified Capabilities
- `client-gateway`: This central contract must evolve to support an explicit Kill Switch and be able to validate cryptographic signatures on orders/mutations sent from the client.

## Impact

The scope impact is significant and spans:
*   **Architecture**: Major overhaul of the `ClientTransport` and `Manager` composition.
*   **Dependencies**: Potential introduction of Cryptography/HMAC libraries to support signed transactions.
*   **APIs**: No direct exposure of new client-callable APIs in this proposal, but the internal service contracts must be updated.