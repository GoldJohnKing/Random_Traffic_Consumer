## Why

Configured download URLs are returning HTTP 403 Forbidden errors, causing worker failures and preventing effective traffic consumption. The current HTTP request headers are outdated and incomplete, making requests easily detectable as automated by server protections.

## What Changes

- Update User-Agent to latest browser version (Chrome 131+)
- Add comprehensive browser-like request headers (Sec-Fetch-* site, mode, user, dest)
- Fix Accept header to match actual resource types (application/octet-stream for downloads)
- Improve Referer header to use proper domain instead of self-referencing URL
- Add Sec-Ch-Ua headers for modern browser identification
- Add DNT header for privacy signaling

## Capabilities

### New Capabilities
- `http-request-headers`: Enhanced HTTP request header generation for browser-like behavior

### Modified Capabilities
- None (implementation-only change, no spec-level requirement changes)

## Impact

- **Affected Code**: `internal/downloader/worker.go` (fetch and fetchWithoutRange functions)
- **Dependencies**: None (uses only standard library)
- **Behavior Change**: Workers will send more realistic HTTP headers, reducing 403 errors from protected resources
