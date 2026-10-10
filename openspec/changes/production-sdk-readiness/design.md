# Design

## Context

This design is driven by the need to elevate the `ibkrapi4go` SDK from a functionally correct foundation to an enterprise-grade, production-hardened tool suitable for high-stakes financial applications. The initial proposal defined the goal: achieving **deterministic performance, absolute state consistency, and catastrophic failure prevention**. (See `proposal.md` for motivation.)

## Goals / Non-Goals

**Goals:**
*   **Guarantee Transaction Semantics:** Ensure all mutations (orders, transfers) are processed exactly once via system-level idempotency and sequence tracking (`execution-integrity`).
*   **Ensure Operational Safety:** Implement a global, deterministic "Kill Switch" capable of halting all trading activity instantly at the transport layer.
*   **Predictable Performance:** Minimize latency jitter by architecting high-throughput, lock-free data paths (addressing the `transport-resilience` goal).
*   **Auditability:** Provide wire-level timestamping and full immutable logging of all market data and transaction requests.

**Non-Goals:**
*   This planning phase does **not** include the implementation of the cryptographic signing logic; it includes only the **interface and architectural requirement** for it.
*   This design does **not** dictate the exact choice of an external, high-performance concurrency library; it only dictates the *pattern* (lock-free, ring buffer) required.

## Decisions

The core architectural decisions needed to meet the Tier 1 and Tier 2 production standards are:

1.  **Client Architecture (The Core)**: The `Client`'s responsibility must expand beyond mere dependency aggregation. It must become the **State Orchestrator**, holding the authoritative `SafetyState` (Kill Switch status, sequence counters) that all Manager components query before executing any request.
    *   *Rationale*: This centralizes control, ensuring a 'Kill Switch' check runs before *any* trade operation, enforcing global consistency.
    *   *Alternative Considered*: Encapsulating the Kill Switch within the lowest-level `ClientTransport`. This was rejected because it would allow a high-level Manager (e.g., `TradeManager`) to bypass the safety check if its internal retry loop wasn't programmed perfectly. The State Orchestrator pattern guarantees a single point of entry for safety enforcement.
2.  **Protocol Handling (Resilience)**: The Transport layer must adopt a **layered, middleware-style chain**. All requests must pass sequentially through: **Rate Limiter $\rightarrow$ Retry Mechanism $\rightarrow$ Circuit Breaker $\rightarrow$ Signature Verifier $\rightarrow$ Network Sender.**
    *   *Rationale*: This allows for predictable, ordered failure handling. For example, Rate Limiting must happen *before* engaging the Retry mechanism to avoid fruitless attempts.
    *   *Alternative Considered*: Using a single function that wraps retries and circuit breaking. This was rejected because it tightly couples the concerns, making maintenance of the strict ordering complex and brittle.

## Risks / Trade-offs

*   **State Synchronization Risk**: Introducing a single, global `SafetyState` via the `Client` increases coupling. Any bug in the `Client`'s state management becomes a critical bug for the entire SDK.
    *   *Mitigation*: Strict application of immutable state objects and heavy testing (TDD) on the `Client`'s core logic.
*   **Performance Overhead**: The introduction of logging, sequencing, and signature verification adds non-negligible latency overhead.
    *   *Mitigation*: Design the `Auditing/Logging` and `Signing` components to run asynchronously or on specialized, non-critical worker threads to avoid blocking the primary execution path.

## Open Questions

*   **Protocol Framing**: While we decided on the middleware pattern, we still need to formally define the exact structure of the "Kill Switch" message (e.g., a specific packet type, or a REST/WS control endpoint). This is a detail for `specs/client-gateway/spec.md`, not the design.
*   **External Dependency for Signature**: What specific, certified cryptographic library will be used? (e.g., Go's `crypto/rsa` vs. a specialized library). This is critical for implementation but can be resolved in the `tasks.md` / `specs/execution-integrity/spec.md`.