# Spec Delta

## Purpose

Lets the SDK guarantee exactly-once processing for critical financial mutations and provides an immutable audit trail for all operations, ensuring state consistency and regulatory compliance.

## ADDED Requirements

### Requirement: Idempotency Key Generation
The system SHALL generate a unique, client-side idempotency key for every mutation request to guarantee exactly-once processing.

#### Scenario: Idempotency key is attached to order request
- **WHEN** a client submits an order request
- **THEN** the system SHALL attach a unique, time-ordered idempotency key to the request header

#### Scenario: Retry uses the same idempotency key
- **WHEN** a request is retried due to a transient network error
- **THEN** the system SHALL use the same idempotency key to prevent duplicate execution

### Requirement: Sequence Tracking
The system SHALL maintain a local sequence number for every message received from the market data feed to detect gaps immediately.

#### Scenario: Gap detected in market data feed
- **WHEN** a message with a non-sequential local sequence number is received
- **THEN** the system SHALL trigger an immediate reconnection to the market data feed and log the incident

#### Scenario: Heartbeat failure detected
- **WHEN** the expected heartbeat interval is exceeded without a valid message
- **THEN** the system SHALL trigger an immediate reconnection and log the incident

### Requirement: Audit Logging
The system SHALL log all requests and responses with microsecond-precision timestamps for regulatory auditability.

#### Scenario: Request is logged with high-precision timestamp
- **WHEN** a mutation request is submitted
- **THEN** the system SHALL record the exact timestamp of submission in the audit log

#### Scenario: Response is logged with high-precision timestamp
- **WHEN** a response is received from the gateway
- **THEN** the system SHALL record the exact timestamp of receipt in the audit log