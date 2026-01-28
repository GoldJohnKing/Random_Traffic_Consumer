## Context

### Current State
The application uses a multi-worker architecture where each worker receives a copy of the URL list at initialization. Each worker independently selects random URLs from its local slice and retries failures up to `config.retry` times. There is no shared state for URL health - when a URL fails, workers have no way to communicate this to each other.

### The Problem
When all configured URLs become unavailable (e.g., servers down, network issues), all workers enter an infinite retry loop. Each worker continuously:
1. Selects a random (failed) URL
2. Retries it N times
3. Loops back to step 1 forever

The program never terminates because workers only exit via external context cancellation.

### Constraints
- Must maintain thread-safety across multiple concurrent workers
- Must preserve existing retry semantics (respect `config.retry` per URL)
- Must log to stderr (consistent with current error handling)
- No external dependencies can be added

## Goals / Non-Goals

**Goals:**
- Eliminate worker deadlock when all URLs fail
- Automatically remove permanently failed URLs from the pool
- Terminate gracefully when URL pool is exhausted
- Maintain random URL selection across workers
- Preserve existing retry behavior per URL

**Non-Goals:**
- URL health recovery (once removed, URLs stay removed)
- Distinguishing transient vs permanent failures (all failures treated equally)
- Per-URL backoff strategies
- Persisting URL health across restarts

## Decisions

### 1. Centralized URLRegistry with Shared State

**Decision:** Create a single `URLRegistry` instance managed by the `Pool`, shared by all workers via reference.

**Rationale:**
- Single source of truth for URL availability
- Failure count shared across all workers (no redundant retries)
- Thread-safe via `sync.RWMutex`
- Enables pool-level exhaustion detection

**Alternatives considered:**
- **Distributed failure tracking via channels**: More complex, eventual consistency issues
- **Per-worker failure maps**: Doesn't solve redundant retries, no centralized exhaustion detection

### 2. Per-URL Failure Counter with Threshold Removal

**Decision:** Track failure count per URL using `map[string]int`. Remove URL when `failureCount >= maxRetries`.

**Rationale:**
- Simple, deterministic semantics
- Respects existing `config.retry` value
- No additional configuration needed
- Atomic check-and-remove under mutex

**Alternatives considered:**
- **Immediate removal on first failure**: Too aggressive, network blips would remove valid URLs
- **Exponential backoff with recovery**: More complex, out of scope per requirements

### 3. Workers Fetch New URL Each Download Cycle

**Decision:** Call `registry.GetRandomURL()` at the start of each `download()` call, not cache URLs per worker.

**Rationale:**
- Workers are not bound to specific URLs
- Fresh URL selection respects current pool state
- Automatically adapts as URLs are removed

**Alternatives considered:**
- **Worker binds to URL for lifetime**: Would require worker restart/reassignment when URL fails
- **Worker periodically refreshes**: Adds complexity with minimal benefit

### 4. Pool Monitor Goroutine for Exhaustion Detection

**Decision:** Launch a background goroutine in `Pool.Start()` that periodically checks `registry.IsEmpty()` and cancels pool context when true.

**Rationale:**
- Non-blocking for workers (no polling in hot path)
- Immediate shutdown when pool exhausted
- Clean termination via existing context cancellation mechanism

**Alternatives considered:**
- **Worker-initiated shutdown**: Race condition - which worker calls cancel?
- **Blocking GetRandomURL()**: Would deadlock all workers waiting on empty pool

### 5. stderr Logging for URL Removal

**Decision:** Log URL removal events to stderr with format: `"URL <url> removed after <n> failures"`

**Rationale:**
- Consistent with existing error logging (see `worker.go:106`)
- Allows users to redirect errors separately from stdout stats
- Provides visibility into URL degradation

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              URLRegistry                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  Fields:                                                                    │
│    - mu           sync.RWMutex                                             │
│    - urls         []string          (available URLs)                        │
│    - failureCount map[string]int     (per-URL retry counter)                │
│    - maxRetries   int               (from config.Download.Retry)            │
│    - poolCtx      context.Context    (for shutdown signalling)              │
│                                                                             │
│  Methods:                                                                   │
│    + GetRandomURL() (string, error)  - Returns random URL or error if empty │
│    + MarkFailed(url string)          - Increments failure count, removes    │
│                                        if threshold reached, logs to stderr │
│    + IsEmpty() bool                   - Returns true if no URLs available   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │                │
                                    │                │
                    ┌───────────────┴────────────────┴───────────────┐
                    │                                               │
                    ▼                                               ▼
        ┌───────────────────────┐                       ┌───────────────────────┐
        │      Worker           │                       │       Pool             │
        ├───────────────────────┤                       ├───────────────────────┤
        │  - No urls field      │                       │  - Creates registry    │
        │  - download() calls   │                       │  - Passes to workers   │
        │    GetRandomURL()     │                       │  - Starts monitor      │
        └───────────────────────┘                       └───────────────────────┘
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| **Mutex contention** - All workers compete for registry lock | Lock is only held during URL selection (fast operation), not during download |
| **Worker starvation** - If all URLs failing, workers may frequently hit empty pool | Workers exit immediately when `GetRandomURL()` returns error (no busy wait) |
| **False positive removal** - Transient network issues cause valid URLs to be removed | This is intentional per requirements; users can restart with fresh URL list |
| **No URL recovery** - Removed URLs never come back even if server recovers | Out of scope per requirements; can be added later if needed |

## Migration Plan

No special migration needed - this is an internal implementation change:
1. Add `internal/downloader/registry.go`
2. Modify `worker.go` - remove `urls` field, update `download()` method
3. Modify `pool.go` - add registry creation and monitor goroutine
4. Build and test

Rollback: Revert commits, restore previous worker/pool implementation.

## Open Questions

None - design is complete and ready for implementation.
