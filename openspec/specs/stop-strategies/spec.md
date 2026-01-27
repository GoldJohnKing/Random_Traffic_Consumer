# Stop Strategies Specification

## Purpose

Defines requirements for test termination conditions including duration-based, traffic-based, and combined stop modes.

## Requirements

### Requirement: Multiple stop modes
The system SHALL support three stop modes: duration-only, traffic-only, and both conditions combined.

#### Scenario: Duration mode
- **WHEN** stop.mode is set to "duration" and stop.duration is set to "1h"
- **THEN** the system runs for exactly 1 hour
- **AND** exits after the time elapses regardless of traffic consumed
- **AND** traffic_limit is ignored

#### Scenario: Traffic mode
- **WHEN** stop.mode is set to "traffic" and stop.traffic_limit is set to "100 GB"
- **THEN** the system runs until 100 GB is downloaded
- **AND** exits when traffic limit is reached regardless of time elapsed
- **AND** duration is ignored

#### Scenario: Both mode
- **WHEN** stop.mode is set to "both" with duration="2h" and traffic_limit="50 GB"
- **THEN** the system exits when EITHER condition is met first
- **AND** if 50 GB is reached in 30 minutes, the system exits
- **AND** if 2 hours elapses before reaching 50 GB, the system exits

### Requirement: Traffic limit parsing
The system SHALL parse traffic limit values using a custom TrafficSize type that implements yaml.Unmarshaler.

#### Scenario: Parse GB units via UnmarshalYAML
- **WHEN** stop.traffic_limit is set to "100 GB"
- **THEN** the TrafficSize.UnmarshalYAML method is invoked
- **AND** the limit is converted to 107374182400 bytes (100 * 1024^3)
- **AND** parsing happens automatically during YAML unmarshaling

#### Scenario: Parse MB units via UnmarshalYAML
- **WHEN** stop.traffic_limit is set to "500 MB"
- **THEN** the limit is converted to 524288000 bytes (500 * 1024^2)

#### Scenario: Parse TB units via UnmarshalYAML
- **WHEN** stop.traffic_limit is set to "1 TB"
- **THEN** the limit is converted to 1099511627776 bytes (1 * 1024^4)

#### Scenario: Invalid unit returns error
- **WHEN** stop.traffic_limit is set to "100 XB"
- **THEN** UnmarshalYAML returns an error
- **AND** the error message indicates the unknown unit
- **AND** system exits before starting downloads

### Requirement: Real-time progress display
The system SHALL display progress toward stop conditions in console output.

#### Scenario: Duration mode countdown
- **WHEN** stop.mode is "duration" and 45 minutes have elapsed of a 1-hour test
- **THEN** the console displays "剩余: 00:15:00" or "Remaining: 00:15:00"

#### Scenario: Traffic mode progress
- **WHEN** stop.mode is "traffic" with 100 GB limit and 12.5 GB has been downloaded
- **THEN** the console displays "总流量: 12.50 GB / 100 GB"
- **AND** a progress bar shows 12.5% completion

#### Scenario: Both mode dual progress
- **WHEN** stop.mode is "both" with 2-hour duration and 50 GB traffic limit
- **AND** current state is 45 minutes elapsed and 12.5 GB downloaded
- **THEN** the console displays both time progress (37.5%) and traffic progress (25%)
