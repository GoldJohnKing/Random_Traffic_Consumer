## Why

When all configured URLs become unavailable, workers enter an infinite retry loop - each worker continuously retries the same dead URLs independently with no mechanism to remove failed URLs or terminate the program. This causes a deadlock where the application never exits despite having no working URLs.

## What Changes

- **BREAKING**: Remove per-worker URL slice (`Worker.urls []string`)
- **NEW**: Create centralized `URLRegistry` with thread-safe URL health tracking
- **NEW**: Workers fetch random URLs from registry on each download cycle (not bound to specific URLs)
- **NEW**: Failed URLs are permanently removed from pool after `config.retry` consecutive failures
- **NEW**: Program terminates gracefully when URL pool becomes empty
- **NEW**: URL removal events logged to stderr for visibility
- **MODIFY**: `Pool` to create and manage the `URLRegistry`
- **MODIFY**: `Worker.download()` to use `URLRegistry.GetRandomURL()` instead of local URL selection

## Capabilities

### New Capabilities
- `url-health-management`: Centralized URL availability tracking and management across all workers, with automatic removal of failed URLs and pool exhaustion detection

### Modified Capabilities
- None (implementation changes only, no external behavior changes at spec level)

## Impact

- **Code affected**:
  - `internal/downloader/worker.go`: Remove `urls` field, modify `download()` and `selectRandomURL()`
  - `internal/downloader/pool.go`: Add `URLRegistry` creation and management
  - `internal/downloader/registry.go`: **NEW FILE** - URL health tracking implementation
- **Dependencies**: No new external dependencies
- **APIs**: Internal worker/pool APIs modified, no external API changes
