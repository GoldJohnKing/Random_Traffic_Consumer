## ADDED Requirements

### Requirement: Centralized URL pool management
The system SHALL maintain a centralized URL pool that tracks availability of all configured URLs across all worker instances.

#### Scenario: URL pool initialization
- **WHEN** the application starts
- **THEN** all URLs from configuration are added to the available URL pool
- **AND** each URL has a failure counter initialized to 0

#### Scenario: URL pool is shared across workers
- **WHEN** multiple workers are created
- **THEN** all workers reference the same URL pool instance
- **AND** URL availability state is shared across all workers

### Requirement: Random URL selection from pool
The system SHALL provide a method for workers to randomly select a URL from the available pool on each download attempt.

#### Scenario: Worker fetches random URL
- **WHEN** a worker calls GetRandomURL()
- **THEN** a URL is randomly selected from the available URL pool
- **AND** the URL is returned to the worker
- **AND** the operation is thread-safe

#### Scenario: Empty pool returns error
- **WHEN** a worker calls GetRandomURL() and the URL pool is empty
- **THEN** an error is returned
- **AND** no URL is provided to the worker

### Requirement: URL failure tracking and removal
The system SHALL track failure count per URL and permanently remove URLs from the pool when the failure count reaches the configured retry threshold.

#### Scenario: First URL failure increments counter
- **WHEN** a URL fails to download
- **THEN** the failure counter for that URL is incremented by 1
- **AND** the URL remains in the available pool

#### Scenario: URL removed after reaching retry threshold
- **WHEN** a URL's failure count reaches the configured max retry value
- **THEN** the URL is permanently removed from the available pool
- **AND** a removal message is logged to stderr
- **AND** the message includes the URL and failure count

#### Scenario: Removed URL no longer selected
- **WHEN** a URL has been removed from the pool
- **THEN** subsequent GetRandomURL() calls SHALL never return that URL

### Requirement: Pool exhaustion detection
The system SHALL provide a method to check if the URL pool is empty (no available URLs).

#### Scenario: Pool is empty when all URLs removed
- **WHEN** all URLs have been removed due to failures
- **THEN** IsEmpty() returns true

#### Scenario: Pool is not empty when URLs remain
- **WHEN** at least one URL remains in the available pool
- **THEN** IsEmpty() returns false

### Requirement: Program termination on pool exhaustion
The system SHALL terminate the entire application when the URL pool becomes empty.

#### Scenario: Pool monitor detects exhaustion
- **WHEN** the pool monitor detects that the URL pool is empty
- **THEN** the pool context is cancelled
- **AND** a termination message is logged
- **AND** all workers receive the cancellation signal

#### Scenario: Workers exit on context cancellation
- **WHEN** a worker receives the pool context cancellation
- **THEN** the worker exits its download loop
- **AND** the worker goroutine terminates

### Requirement: Thread-safe URL pool operations
All URL pool operations SHALL be thread-safe to support concurrent access from multiple workers.

#### Scenario: Concurrent GetRandomURL calls
- **WHEN** multiple workers call GetRandomURL() simultaneously
- **THEN** each call returns a valid URL (if pool not empty)
- **AND** no race conditions occur
- **AND** no URLs are duplicated or lost

#### Scenario: Concurrent MarkFailed calls
- **WHEN** multiple workers call MarkFailed() for different URLs simultaneously
- **THEN** all failure counts are correctly incremented
- **AND** URLs reaching threshold are properly removed

### Requirement: Workers fetch new URL each download cycle
Workers SHALL fetch a new URL from the pool on each download attempt, not cache URLs between attempts.

#### Scenario: Worker calls download multiple times
- **WHEN** a worker completes a download and calls download() again
- **THEN** GetRandomURL() is called again
- **AND** a (potentially different) URL is selected from the current pool state
