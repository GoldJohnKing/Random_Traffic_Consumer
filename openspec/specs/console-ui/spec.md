# Console UI Specification

## Purpose

Defines requirements for console output formatting and display in the Random Traffic Consumer tool, including multi-line refresh, section layout, and formatting standards for traffic, time, and request statistics.

## Requirements

### Requirement: Statistics data source
The console SHALL obtain statistics from the stats monitoring subsystem.

#### Scenario: Get statistics snapshot
- **WHEN** the console refreshes
- **THEN** the system requests a statistics snapshot from the stats collector
- **AND** the snapshot contains total bytes, success count, error count, current speed, and average speed

### Requirement: Multi-line screen refresh
The console output SHALL use ANSI escape sequences to clear and redraw the entire status display on each refresh interval.

#### Scenario: Refresh on interval
- **WHEN** the refresh interval timer triggers
- **THEN** the system clears the screen and redraws all status sections

#### Scenario: Clean display without stacking
- **WHEN** multiple refresh cycles occur
- **THEN** the display shall not show stacked or duplicated output

### Requirement: Section layout with separators
The console output SHALL display information in sections separated by ASCII lines.

#### Scenario: Display main sections
- **WHEN** the status display renders
- **THEN** the system shall display sections with clear separators using `-` and `=` characters

#### Scenario: Section headers
- **WHEN** a section is displayed
- **THEN** the section shall have a header name followed by a separator line

### Requirement: Traffic section display
The system SHALL display traffic information in a dedicated section.

#### Scenario: Traffic display with limit
- **WHEN** stop mode is "traffic" or "both" and a traffic limit is configured
- **THEN** the system shall display downloaded amount, limit, and percentage

#### Scenario: Traffic display without limit
- **WHEN** stop mode is "duration" (no traffic limit)
- **THEN** the system shall display only the downloaded amount without denominator or percentage

#### Scenario: Speed display
- **WHEN** the traffic section renders
- **THEN** the system shall display current speed in Mbps

### Requirement: Time section display
The system SHALL display time information in a dedicated section.

#### Scenario: Time display with limit
- **WHEN** stop mode is "duration" or "both" and a duration limit is configured
- **THEN** the system shall display elapsed time, remaining time, total time, and percentage

#### Scenario: Time display without limit
- **WHEN** stop mode is "traffic" (no duration limit)
- **THEN** the system shall display only the elapsed time without denominator or percentage

### Requirement: Requests section display
The system SHALL display request statistics in a dedicated section.

#### Scenario: Display request statistics
- **WHEN** the requests section renders
- **THEN** the system shall display worker count, success count, and failure count

### Requirement: Bandwidth limit section (optional)
The system SHALL display bandwidth limit information when limiting is enabled.

#### Scenario: Display limit information
- **WHEN** bandwidth limiting is enabled
- **THEN** the system shall display the limit in Mbps and current utilization percentage

#### Scenario: Hide when disabled
- **WHEN** bandwidth limiting is disabled
- **THEN** the bandwidth limit section shall not appear

### Requirement: Column alignment
Numerical values SHALL be right-aligned within fixed-width columns.

#### Scenario: Consistent alignment
- **WHEN** the display renders
- **THEN** all numerical values shall be right-aligned in columns of consistent width

### Requirement: Speed unit format
Speed SHALL be displayed in Mbps (megabits per second).

#### Scenario: Speed display format
- **WHEN** speed is displayed
- **THEN** the value shall be formatted as "X.XX Mbps"

### Requirement: Traffic unit format
Traffic SHALL be displayed using 1024-based units (KB, MB, GB, TB).

#### Scenario: Auto-scale traffic
- **WHEN** traffic amount is displayed
- **THEN** the system shall automatically select appropriate unit (KB/MB/GB/TB)

#### Scenario: Traffic display format
- **WHEN** traffic is displayed
- **THEN** the value shall be formatted as "X.XX XB" (e.g., "1.23 GB")

### Requirement: Time format
Time SHALL be displayed in HH:MM:SS format.

#### Scenario: Time display format
- **WHEN** time is displayed
- **THEN** the value shall be formatted as "HH:MM:SS"

### Requirement: No latency tracking
The system SHALL NOT collect or display latency statistics.

#### Scenario: No latency in stats
- **WHEN** statistics are collected
- **THEN** latency data shall not be recorded

#### Scenario: No latency in display
- **WHEN** the console renders
- **THEN** no latency information shall appear in any section

### Requirement: No progress bars
The system SHALL NOT display visual progress bars.

#### Scenario: Clean sections
- **WHEN** any section renders
- **THEN** no progress bar graphics shall appear

### Requirement: Windows 10 1607+ compatibility
The console output SHALL work correctly on Windows 10 version 1607 and later.

#### Scenario: ANSI sequence support
- **WHEN** running on Windows 10 1607+
- **THEN** ANSI escape sequences shall render correctly for screen clearing and redrawing
