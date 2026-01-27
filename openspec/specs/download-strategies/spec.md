# Download Strategies Specification

## Purpose

Defines requirements for configurable download strategies including complete file downloads and chunked downloads with HTTP Range header support.

## Requirements

### Requirement: Configurable chunk size strategy
The system SHALL support both complete download and chunked download modes controlled by configuration.

#### Scenario: Complete download mode (chunk_size = 0)
- **WHEN** download.chunk_size is set to 0 in config.yaml
- **THEN** each worker downloads the entire target file
- **AND** the connection is maintained until download completes
- **AND** the connection may be reused for subsequent downloads

#### Scenario: Chunked download mode (chunk_size > 0)
- **WHEN** download.chunk_size is set to a non-zero value (e.g., 10485760 for 10MB)
- **THEN** each worker downloads only the specified number of bytes
- **AND** the worker closes the connection after downloading the chunk
- **AND** the worker reconnects to download the next chunk

### Requirement: HTTP Range header support
The system SHALL use HTTP Range headers for chunked downloads to resume from previous position.

#### Scenario: Range request for chunk
- **WHEN** downloading with chunk_size = 10MB and current offset is 20MB
- **THEN** the system sends Range: bytes=20971520-31457280
- **AND** the server responds with 206 Partial Content
- **AND** only the requested 10MB chunk is downloaded

#### Scenario: Server does not support Range
- **WHEN** the server does not support HTTP Range requests
- **THEN** the system falls back to downloading from the beginning
- **AND** downloads only up to chunk_size bytes before disconnecting

### Requirement: Strategy flexibility
The system SHALL allow the user to choose the appropriate download strategy for their testing scenario.

#### Scenario: Sustained bandwidth testing
- **WHEN** the user wants to test sustained bandwidth under continuous load
- **THEN** chunk_size = 0 is recommended for full connection reuse
- **AND** this simulates real-world large file downloads

#### Scenario: Connection stability testing
- **WHEN** the user wants to test connection establishment under frequent reconnects
- **THEN** chunk_size > 0 is recommended
- **AND** this tests NAT/firewall handling of frequent connections
