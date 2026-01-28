## 1. URLRegistry Implementation

- [x] 1.1 Create `internal/downloader/registry.go` with URLRegistry struct
- [x] 1.2 Implement `NewURLRegistry(urls []string, maxRetries int, poolCtx context.Context)` constructor
- [x] 1.3 Implement `GetRandomURL() (string, error)` method with thread-safe random selection
- [x] 1.4 Implement `MarkFailed(url string)` method with failure counting and removal logic
- [x] 1.5 Implement `IsEmpty() bool` method
- [x] 1.6 Add stderr logging for URL removal events (format: "URL <url> removed after <n> failures")

## 2. Worker Refactoring

- [x] 2.1 Remove `urls []string` field from Worker struct
- [x] 2.2 Add `registry *URLRegistry` field to Worker struct
- [x] 2.3 Update `NewWorker()` to accept registry instead of urls slice
- [x] 2.4 Remove `selectRandomURL()` method (no longer needed)
- [x] 2.5 Update `download()` to call `registry.GetRandomURL()` and handle empty pool error
- [x] 2.6 Update `download()` to call `registry.MarkFailed(url)` when all retries exhausted

## 3. Pool Integration

- [x] 3.1 Add `registry *URLRegistry` field to Pool struct
- [x] 3.2 Update `NewPool()` to create URLRegistry with config URLs and retry count
- [x] 3.3 Update `NewPool()` to pass registry to workers instead of urls slice
- [x] 3.4 Add `startPoolMonitor()` goroutine to check URL pool exhaustion
- [x] 3.5 Call `startPoolMonitor()` in `Start()` method
- [x] 3.6 Update monitor to log termination message and cancel pool context when pool empty

## 4. Testing

- [x] 4.1 Build application: `go build -o random-traffic-consumer.exe .`
- [x] 4.2 Test with valid URLs - workers should operate normally
- [x] 4.3 Test with invalid URLs - workers should fail, URLs should be removed after retry count
- [x] 4.4 Test pool exhaustion - application should terminate when all URLs removed
- [x] 4.5 Verify stderr logging for URL removal events
- [x] 4.6 Verify no deadlock occurs when all URLs fail
