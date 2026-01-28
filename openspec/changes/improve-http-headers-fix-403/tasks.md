## 1. Code Implementation

- [x] 1.1 Add `setBrowserHeaders()` helper function to `worker.go`
- [x] 1.2 Replace header setting in `fetch()` function with helper call
- [x] 1.3 Replace header setting in `fetchWithoutRange()` function with helper call

## 2. Build and Test

- [x] 2.1 Build the application: `go build -o random-traffic-consumer.exe .`
- [ ] 2.2 Run with existing config.yaml and monitor for 403 errors
- [ ] 2.3 Verify successful downloads have increased compared to baseline

## 3. Verification

- [x] 3.1 Check that User-Agent contains Chrome 131+ (verified: Chrome/131.0.0.0)
- [x] 3.2 Check that Accept header is "*/*" not HTML types (verified: "*/*")
- [x] 3.3 Check that Referer uses domain format (verified: scheme://host/ format)
- [x] 3.4 Check that Sec-Fetch-* headers are present (verified: Site, Mode, User, Dest)
- [x] 3.5 Check that Sec-Ch-Ua headers are present (verified: Ua, Mobile, Platform)
- [x] 3.6 Check that DNT header is set to "1" (verified: DNT "1")
