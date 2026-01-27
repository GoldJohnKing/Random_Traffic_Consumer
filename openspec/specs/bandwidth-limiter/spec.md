# Bandwidth Limiter Specification

## Purpose

Defines requirements for bandwidth limiting using the token bucket algorithm to control download speeds.

## Requirements

### Requirement: Token bucket rate limiting
The system SHALL implement bandwidth limiting using the token bucket algorithm to control download speed.

#### Scenario: Token bucket initialization
- **WHEN** bandwidth limiting is enabled with a max rate of 200 Mbps
- **THEN** the token bucket is initialized with capacity equal to 25,000,000 bytes (200 × 1000 × 1000 / 8)
- **AND** tokens are replenished at a rate of 25,000,000 bytes per second

#### Scenario: Consume tokens before download
- **WHEN** a worker initiates a download of N bytes
- **THEN** the worker waits until N tokens are available
- **AND** tokens are consumed before the download proceeds
- **AND** the actual download speed does not exceed the configured limit

### Requirement: Bandwidth limit configuration
The system SHALL use a BandwidthSize type implementing yaml.Unmarshaler for bandwidth limit parsing, using decimal (1000-based) prefixes for Mbps/Kbps/Gbps units.

#### Scenario: Bandwidth limit parsing via UnmarshalYAML
- **WHEN** bandwidth_limit.limit is set to "200 Mbps"
- **THEN** the BandwidthSize.UnmarshalYAML method is invoked
- **AND** the value is converted to bytes per second (25,000,000 = 200 × 1000 × 1000 / 8)
- **AND** no separate MaxBPS field is needed

#### Scenario: Bandwidth limit with Mbps unit
- **WHEN** bandwidth_limit.limit is set to "200 Mbps"
- **THEN** the limit is correctly converted to bytes per second (25,000,000)
- **AND** the limit is displayed as "200.00 Mbps" in the console

#### Scenario: Bandwidth limit with Kbps unit
- **WHEN** bandwidth_limit.limit is set to "500 Kbps"
- **THEN** the limit is correctly converted to bytes per second (62,500)
- **AND** the limit is displayed as "0.50 Mbps" in the console

#### Scenario: Bandwidth limit with Gbps unit
- **WHEN** bandwidth_limit.limit is set to "1 Gbps"
- **THEN** the limit is correctly converted to bytes per second (125,000,000)
- **AND** the limit is displayed as "1000.00 Mbps" in the console

#### Scenario: Enable bandwidth limit
- **WHEN** bandwidth_limit.enabled is set to true in config.yaml
- **THEN** the token bucket limiter is active for all workers
- **AND** download speed is capped at bandwidth_limit.limit

#### Scenario: Disable bandwidth limit
- **WHEN** bandwidth_limit.enabled is set to false or omitted
- **THEN** the system downloads at maximum available speed
- **AND** no token bucket throttling occurs

#### Scenario: Invalid bandwidth format returns error
- **WHEN** bandwidth_limit.limit is set to "invalid"
- **THEN** UnmarshalYAML returns an error
- **AND** the system exits with a clear error message

### Requirement: Fair token distribution
The system SHALL distribute available tokens fairly among all waiting workers.

#### Scenario: Multiple workers waiting for tokens
- **WHEN** multiple workers are waiting for tokens
- **THEN** tokens are allocated to workers in FIFO order
- **AND** no worker is starved indefinitely

### Requirement: Bandwidth utilization monitoring
The system SHALL track and display current bandwidth utilization relative to the limit.

#### Scenario: Utilization calculation
- **WHEN** bandwidth limiting is enabled
- **THEN** the system calculates utilization as (current_speed_mbps / limit_mbps) * 100%
- **AND** utilization is displayed in console output

#### Scenario: Display bandwidth limit at startup
- **WHEN** the application starts with bandwidth_limit.limit set to "200 Mbps"
- **THEN** the startup banner displays "Bandwidth limit: 200.0 Mbps"
- **AND** the runtime display shows "Limit: 200.00 Mbps"
