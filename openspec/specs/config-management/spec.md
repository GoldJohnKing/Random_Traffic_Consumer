# Config Management Specification

## Purpose

Defines requirements for YAML-based configuration loading, validation, and all configurable system parameters.

## Requirements

### Requirement: YAML configuration file
The system SHALL load all configuration from a config.yaml file in the current working directory.

#### Scenario: Default config loading
- **WHEN** the application starts
- **THEN** it attempts to load config.yaml from the current directory
- **AND** if the file is missing, the system exits with an error message

#### Scenario: Custom config path
- **WHEN** the user provides a custom config path via command-line flag
- **THEN** the system loads configuration from the specified path
- **AND** the flag format is `-config /path/to/config.yaml`

### Requirement: URL pool configuration
The system SHALL load download target URLs from the configuration file.

#### Scenario: URL list loading
- **WHEN** config.yaml contains a urls array with 3 entries
- **THEN** the system loads all 3 URLs into the download pool
- **AND** workers randomly select from these 3 URLs

#### Scenario: Empty URL pool validation
- **WHEN** config.yaml contains an empty urls array
- **THEN** the system exits with an error: "URL pool cannot be empty"
- **AND** no downloads are initiated

### Requirement: Download configuration section
The system SHALL support configurable download parameters.

#### Scenario: Worker count configuration
- **WHEN** download.workers is set to 64
- **THEN** the system creates exactly 64 concurrent worker goroutines

#### Scenario: Timeout configuration
- **WHEN** download.timeout is set to "30s"
- **THEN** all HTTP requests timeout after 30 seconds
- **AND** valid time units include: s (seconds), m (minutes), h (hours)

#### Scenario: Retry configuration
- **WHEN** download.retry is set to 3
- **THEN** failed downloads are retried up to 3 times before marking as failure

#### Scenario: Chunk size configuration
- **WHEN** download.chunk_size is set to 0
- **THEN** workers download complete files
- **AND** when set to 10485760 (10MB), workers download in 10MB chunks

### Requirement: Stop condition configuration
The system SHALL load test termination conditions from the configuration.

#### Scenario: Mode configuration
- **WHEN** stop.mode is set to "duration"
- **THEN** the system uses duration-based termination
- **AND** valid modes are: "duration", "traffic", "both"

#### Scenario: Duration parsing
- **WHEN** stop.duration is set to "1h"
- **THEN** the duration is parsed as 1 hour (3600 seconds)
- **AND** valid formats include: "30m", "1h", "1h30m"

#### Scenario: Traffic limit parsing
- **WHEN** stop.traffic_limit is set to "100 GB"
- **THEN** the limit is converted to bytes (107374182400)
- **AND** the TrafficSize type handles parsing via UnmarshalYAML
- **AND** valid units include: MB, GB, TB

#### Scenario: Traffic limit with MB unit
- **WHEN** stop.traffic_limit is set to "1024 MB"
- **THEN** the limit is converted to bytes (1073741824)

#### Scenario: Invalid traffic limit format
- **WHEN** stop.traffic_limit is set to an invalid format like "invalid"
- **THEN** the system exits with an error indicating the unknown unit
- **AND** no downloads are initiated

### Requirement: Bandwidth limit configuration
The system SHALL load bandwidth limiting settings using a single field with custom type parsing.

#### Scenario: Enable bandwidth limit
- **WHEN** bandwidth_limit.enabled is true
- **AND** bandwidth_limit.max is "200 Mbps"
- **THEN** the BandwidthSize type handles parsing via UnmarshalYAML
- **AND** the token bucket limiter is active with the parsed limit

#### Scenario: Disable bandwidth limit
- **WHEN** bandwidth_limit.enabled is false or the section is omitted
- **THEN** no bandwidth limiting is applied
- **AND** bandwidth_limit.max is not parsed

#### Scenario: Invalid bandwidth limit format
- **WHEN** bandwidth_limit.max is set to an invalid format
- **THEN** the system exits with an error indicating the expected format

### Requirement: HTTP client configuration
The system SHALL support HTTP client tuning parameters.

#### Scenario: Connection pool configuration
- **WHEN** http.max_idle_conns is set to 100
- **THEN** the HTTP client maintains up to 100 idle connections

#### Scenario: Idle timeout configuration
- **WHEN** http.idle_conn_timeout is set to "90s"
- **THEN** idle connections are closed after 90 seconds

#### Scenario: Keep-alive control
- **WHEN** http.disable_keepalive is false
- **THEN** HTTP keep-alive is enabled for connection reuse

### Requirement: Output configuration
The system SHALL support configurable console refresh interval.

#### Scenario: Refresh interval configuration
- **WHEN** output.refresh_interval is set to "500ms"
- **THEN** the console statistics update every 500 milliseconds
- **AND** valid units include: ms (milliseconds), s (seconds)

### Requirement: Configuration validation
The system SHALL validate all configuration values on startup.

#### Scenario: Invalid worker count
- **WHEN** download.workers is set to 0 or a negative number
- **THEN** the system exits with an error: "Worker count must be positive"

#### Scenario: Invalid stop mode
- **WHEN** stop.mode is set to an invalid value like "invalid"
- **THEN** the system exits with an error listing valid modes: "duration", "traffic", "both"

#### Scenario: Missing required duration
- **WHEN** stop.mode is "duration" but stop.duration is not specified
- **THEN** the system exits with an error: "Duration is required for duration mode"

#### Scenario: Missing required traffic limit
- **WHEN** stop.mode is "traffic" but stop.traffic_limit is not specified
- **THEN** the system exits with an error: "Traffic limit is required for traffic mode"
