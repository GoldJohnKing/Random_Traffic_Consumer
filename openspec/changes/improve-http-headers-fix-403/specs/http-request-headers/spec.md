## ADDED Requirements

### Requirement: Browser-like request headers
The HTTP client SHALL include request headers that mimic a modern browser to avoid detection and reduce 403 Forbidden errors.

#### Scenario: Include all required browser headers
- **WHEN** worker makes an HTTP GET request
- **THEN** request includes User-Agent with latest Chrome version
- **AND** request includes Sec-Fetch-Site header
- **AND** request includes Sec-Fetch-Mode header
- **AND** request includes Sec-Fetch-User header
- **AND** request includes Sec-Fetch-Dest header
- **AND** request includes Sec-Ch-Ua header with platform information
- **AND** request includes Sec-Ch-Ua-Mobile header
- **AND** request includes Sec-Ch-Ua-Platform header
- **AND** request includes DNT header

### Requirement: Updated User-Agent version
The User-Agent header SHALL use current browser version (Chrome 131+) to appear as a legitimate browser.

#### Scenario: User-Agent matches current browser
- **WHEN** worker makes an HTTP request
- **THEN** User-Agent header contains Chrome version 131 or higher
- **AND** User-Agent matches format: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{version}.0.0.0 Safari/537.36"

### Requirement: Proper Accept header for downloads
The Accept header SHALL match the expected content type for binary downloads rather than HTML.

#### Scenario: Accept header for binary downloads
- **WHEN** worker requests a binary resource (exe, zip, etc.)
- **THEN** Accept header is set to "*/*"
- **AND** Accept header does not include HTML-related content types

### Requirement: Correct Referer header format
The Referer header SHALL use the target URL's domain rather than the full URL to avoid self-referencing detection.

#### Scenario: Referer uses domain instead of full URL
- **WHEN** worker makes a request to "https://example.com/path/file.zip"
- **THEN** Referer header is set to "https://example.com/"
- **AND** Referer does NOT equal the full request URL

### Requirement: Sec-Fetch header values
Sec-Fetch headers SHALL use values appropriate for navigation-initiated downloads.

#### Scenario: Sec-Fetch headers for direct download
- **WHEN** worker makes a request for a file download
- **THEN** Sec-Fetch-Site is "none" (for direct URLs)
- **AND** Sec-Fetch-Mode is "navigate" or "cors"
- **AND** Sec-Fetch-User is "?1" (indicates user-initiated)
- **AND** Sec-Fetch-Dest is "document"

### Requirement: Sec-Ch-Ua headers
Sec-Ch-Ua headers SHALL provide accurate client hints for modern browser identification.

#### Scenario: Client hints headers present
- **WHEN** worker makes an HTTP request
- **THEN** Sec-Ch-Ua contains browser brand and version list
- **AND** Sec-Ch-Ua-Mobile is "?0" (not a mobile device)
- **AND** Sec-Ch-Ua-Platform is "Windows"

### Requirement: DNT header
The Do Not Track header SHALL be set to signal privacy preferences like a real browser.

#### Scenario: DNT header included
- **WHEN** worker makes an HTTP request
- **THEN** DNT header is set to "1"
