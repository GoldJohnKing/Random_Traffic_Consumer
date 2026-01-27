# Stats Monitoring Specification

## Purpose

Defines requirements for thread-safe statistics collection and real-time speed calculation. Provides the data source for console UI display.

## Requirements

### Requirement: Thread-safe statistics collection
The system SHALL collect download statistics using atomic operations to ensure thread safety without locks.

#### Scenario: Byte counter atomicity
- **WHEN** multiple workers simultaneously update the total bytes downloaded
- **THEN** all updates are correctly counted without race conditions
- **AND** sync/atomic operations are used for all counter updates

#### Scenario: Success/error counter atomicity
- **WHEN** workers complete downloads with success or failure
- **THEN** success_count and error_count are updated atomically
- **AND** no counts are lost due to concurrent access

### Requirement: Real-time speed calculation
The system SHALL calculate current download speed based on bytes downloaded in the last refresh interval.

#### Scenario: Current speed calculation
- **WHEN** the console refreshes every 500ms
- **AND** 50 MB was downloaded in the last 500ms interval
- **THEN** the current speed is calculated as 50 MB / 0.5s = 100 MB/s = 800 Mbps

#### Scenario: Average speed calculation
- **WHEN** the system has been running for 60 seconds
- **AND** total bytes downloaded is 600 MB
- **THEN** the average speed is calculated as 600 MB / 60s = 10 MB/s

### Requirement: Statistics snapshot
The system SHALL provide a point-in-time snapshot of all collected statistics for display.

#### Scenario: Snapshot generation
- **WHEN** the console requests current statistics
- **THEN** the system returns a snapshot containing total bytes, success count, error count, current speed, and average speed
- **AND** the snapshot represents an atomic point-in-time view
