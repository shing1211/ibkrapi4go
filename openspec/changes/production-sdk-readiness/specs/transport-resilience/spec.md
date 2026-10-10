# Spec Delta

## Purpose

Lets the SDK maintain predictable, self-healing network connectivity under extreme load, broker throttling, and transient fault conditions, ensuring that financial operations are never blocked indefinitely and that system resources are never exhausted by retry storms.

## ADDED Requirements

### Requirement: Stateful Circuit Breaker
The system SHALL implement a circuit breaker that transitions between CLOSED, OPEN, and HALF-OPEN states based on the specific type of error, not just error count.

#### Scenario: Circuit opens on execution timeout
- **WHEN** three consecutive execution timeout errors occur within a 10-second window
- **THEN** the circuit breaker SHALL transition to an OPEN state, immediately failing all subsequent requests for a cooldown period

#### Scenario: Circuit transitions to HALF-OPEN after cooldown
- **WHEN** the circuit breaker is in an OPEN state and the cooldown period expires
- **THEN** the circuit breaker SHALL transition to a HALF-OPEN state, allowing a single probe request to pass

#### Scenario: Circuit closes after successful probe
- **WHEN** a probe request in HALF-OPEN state succeeds
- **THEN** the circuit breaker SHALL immediately transition to a CLOSED state and reset all error counters

### Requirement: Smart Retry with Exponential Backoff
The system SHALL implement a jittered exponential backoff for transient errors, with the retry strategy differing based on the error classification.

#### Scenario: Transient error triggers exponential backoff retry
- **WHEN** a request fails with a transient error (e.g., 503 Service Unavailable)
- **THEN** the system SHALL retry the request using a jittered exponential backoff starting at 100ms and capping at 30s

#### Scenario: Logic error does not trigger retry
- **WHEN** a request fails with a logic error (e.g., 400 Bad Request, insufficient margin)
- **THEN** the system SHALL NOT retry the request and SHALL return the error immediately to the caller

#### Scenario: Retry is halted on circuit open
- **WHEN** a retry is attempted while the circuit breaker is in an OPEN state
- **THEN** the system SHALL immediately fail the request without attempting to send it to the network

### Requirement: Adaptive Rate Limiting
The system SHALL implement adaptive throttling that automatically adjusts the outgoing request rate based on broker feedback.

#### Scenario: Rate limit exceeded triggers cooldown mode
- **WHEN** a request receives a 429 Too Many Requests response from the broker
- **THEN** the system SHALL immediately enter a cooldown mode, reducing the outgoing request rate by 50%

#### Scenario: Cooldown mode prevents request submission
- **WHEN** the system is in cooldown mode
- **THEN** the system SHALL queue outgoing requests and only submit them at the reduced rate

#### Scenario: Cooldown mode exits after successful request
- **WHEN** a request succeeds while in cooldown mode
- **THEN** the system SHALL gradually restore the request rate to normal over 5 successful requests