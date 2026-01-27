# Multi-thread Downloader Specification

## Purpose

Defines requirements for the concurrent download worker pool that manages HTTP connections, retries, and URL randomization for network traffic generation.

## Requirements

### Requirement: Worker pool management
The system SHALL maintain a fixed pool of 64 concurrent worker goroutines for downloading resources.

#### Scenario: Initialize worker pool
- **WHEN** the application starts
- **THEN** the system creates exactly 64 goroutine workers
- **AND** all workers are ready to accept download tasks
- **AND** worker startup timing follows staggered startup configuration if enabled

#### Scenario: Worker restart on error
- **WHEN** a worker encounters a fatal error
- **THEN** the worker restarts and continues processing tasks

### Requirement: HTTP connection reuse
The system SHALL reuse HTTP connections across multiple download requests to minimize TCP handshake overhead.

#### Scenario: Connection pooling
- **WHEN** multiple downloads target the same host
- **THEN** the system reuses existing connections when available
- **AND** maximum idle connections is configurable (default: 100)

#### Scenario: Idle connection timeout
- **WHEN** a connection remains idle for 90 seconds
- **THEN** the connection is closed and removed from the pool

### Requirement: Random URL selection
Each worker SHALL randomly select a target URL from the configured pool for each download operation.

#### Scenario: URL randomization
- **WHEN** a worker initiates a new download
- **THEN** it selects a URL from the pool using a random distribution
- **AND** the same URL may be selected by multiple workers concurrently

### Requirement: Download timeout handling
The system SHALL enforce a per-request timeout and handle timeout failures gracefully.

#### Scenario: Request timeout
- **WHEN** a download request exceeds the configured timeout (default: 30s)
- **THEN** the request is cancelled
- **AND** the error is recorded in statistics
- **AND** the worker continues with the next download

#### Scenario: Configurable timeout
- **WHEN** the user sets a custom timeout in config.yaml
- **THEN** the system uses the custom timeout value for all requests

### Requirement: Retry mechanism
The system SHALL retry failed downloads up to a configurable number of times before marking as failure.

#### Scenario: Retry on transient failure
- **WHEN** a download fails with a transient error (network timeout, connection refused)
- **THEN** the system retries the download up to 3 times
- **AND** each retry uses a new URL selection

#### Scenario: Retry limit reached
- **WHEN** a download fails after the maximum retry attempts
- **THEN** the download is marked as failed
- **AND** the error is counted in statistics
- **AND** the worker continues with the next download
