package downloader

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"random-traffic-consumer/internal/config"
	"random-traffic-consumer/internal/limiter"
	"random-traffic-consumer/internal/stats"
)

// Worker represents a single download worker
type Worker struct {
	id      int
	urls    []string
	client  *http.Client
	stats   *stats.Stats
	limiter *limiter.TokenBucket
	config  *config.DownloadConfig
	ctx     context.Context
	rand    *rand.Rand // Thread-safe random generator
	mu      sync.Mutex
}

// NewWorker creates a new download worker
func NewWorker(
	id int,
	urls []string,
	client *http.Client,
	statsCollector *stats.Stats,
	limiter *limiter.TokenBucket,
	cfg *config.DownloadConfig,
	ctx context.Context,
) *Worker {
	return &Worker{
		id:      id,
		urls:    urls,
		client:  client,
		stats:   statsCollector,
		limiter: limiter,
		config:  cfg,
		ctx:     ctx,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano() + int64(id))),
	}
}

// Start begins the worker's download loop
func (w *Worker) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer w.recoverFromPanic()

		for {
			// Check for cancellation
			select {
			case <-w.ctx.Done():
				return
			default:
			}

			// Perform download
			w.download()
		}
	}()
}

// download performs a single download operation
func (w *Worker) download() {
	// Select random URL
	url := w.selectRandomURL()

	// Retry logic
	for attempt := 0; attempt <= w.config.Retry; attempt++ {
		if attempt > 0 {
			// Wait before retry
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}

		// Check for cancellation before download
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// Wait for bandwidth tokens if enabled
		if w.limiter != nil && w.limiter.IsEnabled() {
			// We'll wait for tokens after we know the content length
			// For now, proceed with the request
		}

		// Perform download
		bytesDownloaded, err := w.fetch(url)

		if err != nil {
			if attempt == w.config.Retry {
				// Final attempt failed - log to stderr for debugging
				fmt.Fprintf(os.Stderr, "Worker %d: Failed after %d attempts: %v (URL: %s)\n", w.id, attempt+1, err, url)
				w.stats.RecordError()
			}
			continue
		}

		// Success
		w.stats.RecordBytes(bytesDownloaded)
		w.stats.RecordSuccess()
		return
	}

	// All retries failed
	w.stats.RecordError()
}

// fetch performs the actual HTTP request and download
func (w *Worker) fetch(url string) (int64, error) {

	if url == "" {
		return 0, fmt.Errorf("empty URL")
	}

	var req *http.Request
	var err error

	// Create request with Range header if chunked
	if w.config.ChunkSize > 0 {
		offset := w.rand.Int63() // Random starting position
		req, err = http.NewRequestWithContext(w.ctx, "GET", url, nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+w.config.ChunkSize-1))
	} else {
		req, err = http.NewRequestWithContext(w.ctx, "GET", url, nil)
		if err != nil {
			return 0, err
		}
	}

	// Add headers to mimic a real browser and avoid bot detection
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", url)

	// Execute request
	resp, err := w.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		// If Range request failed with 416, try without Range
		if resp.StatusCode == 416 && w.config.ChunkSize > 0 {
			return w.fetchWithoutRange(url)
		}
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Get content length if available
	contentLength := resp.ContentLength

	// Wait for bandwidth tokens before downloading body
	if w.limiter != nil && w.limiter.IsEnabled() {
		if contentLength > 0 {
			w.limiter.Wait(contentLength)
		} else {
			// Unknown size, wait for a reasonable chunk
			w.limiter.Wait(1024 * 1024) // 1MB
		}
	}

	// Download body
	var bytesDownloaded int64
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			bytesDownloaded += int64(n)

			// If we didn't know content length upfront, consume tokens as we read
			if w.limiter != nil && w.limiter.IsEnabled() && contentLength <= 0 {
				w.limiter.Wait(int64(n))
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return bytesDownloaded, err
		}

		// Check for chunk size limit
		if w.config.ChunkSize > 0 && bytesDownloaded >= w.config.ChunkSize {
			break
		}
	}

	return bytesDownloaded, nil
}

// fetchWithoutRange performs a download without Range header (fallback)
func (w *Worker) fetchWithoutRange(url string) (int64, error) {
	req, err := http.NewRequestWithContext(w.ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	// Add headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", url)

	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Download with chunk limit
	var bytesDownloaded int64
	buffer := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			bytesDownloaded += int64(n)
			if w.limiter != nil && w.limiter.IsEnabled() {
				w.limiter.Wait(int64(n))
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return bytesDownloaded, err
		}

		if w.config.ChunkSize > 0 && bytesDownloaded >= w.config.ChunkSize {
			break
		}
	}

	return bytesDownloaded, nil
}

// selectRandomURL selects a random URL from the pool
func (w *Worker) selectRandomURL() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.urls) == 0 {
		return ""
	}
	return w.urls[w.rand.Intn(len(w.urls))]
}

// recoverFromPanic handles panics in the worker goroutine
func (w *Worker) recoverFromPanic() {
	if r := recover(); r != nil {
		// Log panic - worker will exit cleanly
		fmt.Printf("Worker %d panic recovered: %v\n", w.id, r)
	}
}
