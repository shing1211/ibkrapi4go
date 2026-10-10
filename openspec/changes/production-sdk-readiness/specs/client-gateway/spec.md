# Spec Delta

## Purpose

Lets the SDK serve as the central, stateful orchestrator for all communication with the IBKR gateway, providing a unified, safe entry point for all trading operations with deterministic failure handling and global safety controls.

## ADDED Requirements

### Requirement: Global Safety State Management
The system SHALL maintain a single, authoritative global safety state (Kill Switch) that is queried before any trading operation is executed.

#### Scenario: Kill Switch is engaged before order submission
- **WHEN** a client attempts to submit an order while the Kill Switch is engaged
- **THEN** the system SHALL reject the order immediately with a `ErrSafetyEngaged` error and SHALL NOT transmit the order to the gateway

#### Scenario: Kill Switch can be engaged programmatically
- **WHEN** a client calls `Client.EngageKillSwitch()`
- **THEN** the system SHALL immediately set the global safety state to engaged, causing all subsequent trading operations to be rejected

#### Scenario: Kill Switch disengagement re-enables operations
- **WHEN** a client calls `Client.DisengageKillSwitch()`
- **THEN** the system SHALL set the global safety state to disengaged and SHALL resume normal trading operations

### Requirement: Mutation Request Signing
The system SHALL apply a cryptographic signature to all critical mutation requests (orders above a configurable size threshold, withdrawals) before transmitting them to the gateway.

#### Scenario: Large order is signed before transmission
- **WHEN** a client submits an order with a value exceeding the configured signing threshold
- **THEN** the system SHALL generate an HMAC signature over the request payload and attach it to the request headers before sending

#### Scenario: Request without valid signature is rejected by gateway
- **WHEN** the gateway receives a mutation request without a valid signature
- **THEN** the system SHALL log the rejection and return `ErrSignatureInvalid` to the client

### Requirement: Deterministic Failure on Critical Errors
The system SHALL halt all further operations and engage the Kill Switch immediately upon detecting a critical integrity error (e.g., fill for an unknown order, price data outside hard bounds).

#### Scenario: Unknown fill triggers safety halt
- **WHEN** the gateway reports a fill for an order ID that does not exist in the local order tracking map
- **THEN** the system SHALL immediately engage the Kill Switch and log a `CriticalIntegrityError`

#### Scenario: Price outside hard bounds triggers safety halt
- **WHEN** incoming market data contains a price that exceeds the configured hard bounds for a tracked instrument
- **THEN** the system SHALL immediately engage the Kill Switch and log a `PriceBoundViolationError`