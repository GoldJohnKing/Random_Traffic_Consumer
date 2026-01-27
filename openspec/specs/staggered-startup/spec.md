# Staggered Startup Specification

## Purpose

Defines requirements for configurable worker startup staggering to distribute connection establishment over time and prevent bandwidth spikes during initialization.

## Requirements

### Requirement: Staggered worker startup
When enabled, the system SHALL delay the start of each worker by a calculated amount to distribute initialization over time.

#### Scenario: Staggered startup with default configuration
- **WHEN** `stagger_start.enabled` is `true`
- **AND** `stagger_start.base_delay` is `200ms`
- **AND** `stagger_start.jitter` is `50ms`
- **THEN** Worker 0 starts immediately (delay ~0ms)
- **AND** Worker 1 starts after approximately 200ms (±50ms)
- **AND** Worker 2 starts after approximately 400ms (±50ms)
- **AND** Worker N starts after approximately N × 200ms (±50ms)

#### Scenario: Staggered startup disabled
- **WHEN** `stagger_start.enabled` is `false` or omitted
- **THEN** all workers start simultaneously
- **AND** no startup delays are applied

### Requirement: Delay calculation with jitter
The system SHALL calculate each worker's startup delay using the formula: `delay = (worker_index × base_delay) + random(-jitter, +jitter)`.

#### Scenario: Jitter randomization
- **WHEN** calculating delay for Worker 2 with `base_delay=200ms` and `jitter=50ms`
- **THEN** base delay is 400ms (2 × 200ms)
- **AND** final delay is between 350ms and 450ms
- **AND** the jitter value is randomly determined each time

#### Scenario: No jitter (jitter = 0)
- **WHEN** `jitter` is set to `0ms`
- **THEN** all workers start at exactly predictable intervals
- **AND** Worker N delay equals N × base_delay

### Requirement: Configuration validation
The system SHALL validate the staggered startup configuration at load time and reject invalid values.

#### Scenario: Jitter exceeds base delay
- **WHEN** `jitter` is greater than `base_delay`
- **THEN** configuration loading fails
- **AND** an error message indicates that jitter must not exceed base_delay

#### Scenario: Negative values
- **WHEN** `base_delay` or `jitter` is negative
- **THEN** configuration loading fails
- **AND** an error message indicates that values must be positive

#### Scenario: Valid configuration
- **WHEN** `base_delay` is `100ms`
- **AND** `jitter` is `50ms` (less than or equal to base_delay)
- **THEN** configuration loads successfully
- **AND** staggered startup is enabled

### Requirement: Configuration structure
The system SHALL support the following configuration structure under `download.stagger_start`.

#### Scenario: Full configuration
- **WHEN** config.yaml contains:
  ```yaml
  download:
    stagger_start:
      enabled: true
      base_delay: 200ms
      jitter: 50ms
  ```
- **THEN** staggered startup is enabled with specified parameters
- **AND** base_delay of 200ms is used for worker spacing
- **AND** jitter of ±50ms is applied to each worker's delay

#### Scenario: Minimal configuration
- **WHEN** config.yaml contains:
  ```yaml
  download:
    stagger_start:
      enabled: true
  ```
- **THEN** staggered startup is enabled with default values
- **AND** base_delay defaults to 200ms
- **AND** jitter defaults to 50ms

#### Scenario: Disabled configuration
- **WHEN** `stagger_start.enabled` is `false`
- **THEN** staggered startup is disabled
- **AND** workers start simultaneously regardless of other settings
