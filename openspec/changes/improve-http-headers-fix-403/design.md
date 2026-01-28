## Context

### Current State
The traffic consumer tool (`internal/downloader/worker.go`) currently sets basic HTTP request headers:

```go
req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
req.Header.Set("Accept-Encoding", "gzip, deflate, br")
req.Header.Set("Connection", "keep-alive")
req.Header.Set("Referer", url)  // Self-referencing
```

### Problem
These headers have several issues:
1. **Chrome 120 is outdated** - Latest version is 131+, old versions are suspicious
2. **Self-referencing Referer** - Using full URL as Referer is easily detected
3. **Wrong Accept header** - HTML content type for binary downloads is mismatched
4. **Missing modern browser headers** - No Sec-Fetch-* or Sec-Ch-Ua headers
5. **Easily detected as automated** - Pattern doesn't match real browser behavior

### Constraints
- Must use Go standard library only (no external dependencies)
- Must work with existing worker architecture
- Must maintain backward compatibility with config.yaml

## Goals / Non-Goals

**Goals:**
- Add realistic browser request headers to reduce 403 errors
- Update User-Agent to current Chrome version
- Implement proper Referer header generation (domain only)
- Add Sec-Fetch-* headers for modern browser behavior
- Add Sec-Ch-Ua client hint headers
- Fix Accept header for binary downloads

**Non-Goals:**
- Adding proxy support
- Implementing cookie persistence
- URL health checking (deferred to future change)
- Replacing existing URLs (deferred to future change)

## Decisions

### Decision 1: Chrome 131 as User-Agent baseline
**Choice:** Use Chrome 131.0.0.0 as the User-Agent version

**Rationale:**
- Chrome 131 is current stable version (as of 2026)
- Using a real current version avoids detection as outdated browser
- Format follows standard: `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36`

**Alternatives considered:**
- Edge/Firefox UA - Less common, might be more scrutinized
- Randomizing versions - Adds complexity without clear benefit
- Bot-specific UA - Defeats the purpose of appearing as browser

### Decision 2: Accept header set to "*/*"
**Choice:** Use `*/*` for Accept header instead of HTML content types

**Rationale:**
- Binary downloads (exe, zip) should use `*/*` to accept any content type
- Matches real browser behavior for direct file downloads
- More accurate than listing HTML types for binary resources

**Alternatives considered:**
- Keep HTML Accept types - Doesn't match binary download behavior
- Use specific MIME types - Too restrictive, might break on unexpected content types

### Decision 3: Referer from URL domain extraction
**Choice:** Extract domain from URL for Referer header

**Rationale:**
- Self-referencing (Referer = URL) is easily detected as automated
- Using domain (e.g., `https://example.com/`) is realistic
- Simple to implement using `url.Parse()` and scheme+host

**Implementation:**
```go
parsedURL, _ := url.Parse(url)
referer := fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host)
req.Header.Set("Referer", referer)
```

### Decision 4: Sec-Fetch header values for navigation
**Choice:** Use values indicating user-initiated navigation

| Header | Value | Rationale |
|--------|-------|-----------|
| Sec-Fetch-Site | "none" | Direct URL, not from another site |
| Sec-Fetch-Mode | "navigate" | Navigation-like request |
| Sec-Fetch-User | "?1" | User-initiated (not automated) |
| Sec-Fetch-Dest | "document" | Destination is document-like |

**Alternatives considered:**
- "cors" mode - More accurate for cross-origin, but "navigate" is more common
- "empty" mode - Some browsers use this, but less standard

### Decision 5: Sec-Ch-Ua headers format
**Choice:** Use standard Chrome Client Hints format

```
Sec-Ch-Ua: "Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"
Sec-Ch-Ua-Mobile: ?0
Sec-Ch-Ua-Platform: "Windows"
```

**Rationale:**
- Matches actual Chrome 131 behavior
- Standard format from [Client Hints specification](https://wicg.github.io/ua-client-hints/)
- The "Not_A Brand" with version 24 is a known Chrome quirk to include

### Decision 6: Helper function for header setting
**Choice:** Create `setBrowserHeaders(req *http.Request, url string)` helper function

**Rationale:**
- DRY principle - headers set in both `fetch()` and `fetchWithoutRange()`
- Centralized header management makes updates easier
- Clear separation of header-setting logic

**Implementation location:**
- Add as private function in `worker.go`
- Call from both `fetch()` and `fetchWithoutRange()` after creating request

## Implementation Structure

```go
// In worker.go

// setBrowserHeaders sets realistic browser headers on the request
func (w *Worker) setBrowserHeaders(req *http.Request, targetURL string) {
    // Parse URL for Referer
    parsedURL, _ := url.Parse(targetURL)
    referer := fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host)

    // Standard headers
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
    req.Header.Set("Accept", "*/*")
    req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
    req.Header.Set("Accept-Encoding", "gzip, deflate, br")
    req.Header.Set("Referer", referer)
    req.Header.Set("DNT", "1")

    // Sec-Fetch headers
    req.Header.Set("Sec-Fetch-Site", "none")
    req.Header.Set("Sec-Fetch-Mode", "navigate")
    req.Header.Set("Sec-Fetch-User", "?1")
    req.Header.Set("Sec-Fetch-Dest", "document")

    // Client Hints
    req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
    req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
    req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
}
```

### Code Changes Required

**File:** `internal/downloader/worker.go`

1. **Add helper function** `setBrowserHeaders()` after `selectRandomURL()`

2. **Replace header setting in `fetch()`** (lines ~146-152):
   - Remove existing `req.Header.Set()` calls
   - Add `w.setBrowserHeaders(req, url)` after creating request

3. **Replace header setting in `fetchWithoutRange()`** (lines ~221-227):
   - Remove existing `req.Header.Set()` calls
   - Add `w.setBrowserHeaders(req, url)` after creating request

## Risks / Trade-offs

### Risk 1: Some servers may still reject requests
**Risk:** Even with improved headers, some servers have additional protections (IP checks, rate limiting, requiring cookies)

**Mitigation:**
- This is a minimal improvement; future changes can add URL health checking
- Monitor error logs to identify persistently failing URLs
- Document which URLs work vs don't work

### Risk 2: Header values may become outdated
**Risk:** Chrome version and header formats change over time

**Mitigation:**
- Chrome 131 is current (2026); future updates can increment version
- Sec-Fetch and Sec-Ch-Ua are stable standards
- Consider making version a config constant for easy updates

### Trade-off: Increased request size
**Trade-off:** More headers = larger request overhead (~300 bytes per request)

**Acceptable because:**
- Negligible impact on large file downloads
- Benefit of reduced 403 errors far outweighs overhead
- Still minimal compared to actual data transferred

## Migration Plan

### Deployment Steps
1. Update `worker.go` with new helper function
2. Replace header-setting calls in `fetch()` and `fetchWithoutRange()`
3. Build and test: `go build -o random-traffic-consumer.exe .`
4. Run with existing config.yaml and monitor 403 error rate
5. Verify successful downloads increase

### Rollback Strategy
- If issues occur, revert `worker.go` to previous version
- No data migration needed (no config file changes)
- Simple binary redeployment

## Open Questions

None - design is straightforward with clear implementation path.

### Future Considerations (out of scope)
- Should Chrome version be configurable in config.yaml?
- Should we add randomization to headers?
- Should URL health checking be added?
- Should failed URLs be tracked and skipped?
