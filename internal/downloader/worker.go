package downloader

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"random-traffic-consumer/internal/config"
	"random-traffic-consumer/internal/limiter"
	"random-traffic-consumer/internal/stats"
)

// Worker represents a single download worker
type Worker struct {
	id       int
	registry *URLRegistry
	client   *http.Client
	stats    *stats.Stats
	limiter  *limiter.TokenBucket
	config   *config.DownloadConfig
	ctx      context.Context
	rand     *rand.Rand // Thread-safe random generator
	mu       sync.Mutex
}

// NewWorker creates a new download worker
func NewWorker(
	id int,
	registry *URLRegistry,
	client *http.Client,
	statsCollector *stats.Stats,
	limiter *limiter.TokenBucket,
	cfg *config.DownloadConfig,
	ctx context.Context,
) *Worker {
	return &Worker{
		id:       id,
		registry: registry,
		client:   client,
		stats:    statsCollector,
		limiter:  limiter,
		config:   cfg,
		ctx:      ctx,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano() + int64(id))),
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

			// Check if URL pool is empty - worker should exit
			if w.registry.IsEmpty() {
				return
			}

			// Perform download
			w.download()
		}
	}()
}

// download performs a single download operation
func (w *Worker) download() {
	// Select random URL from registry
	url, err := w.registry.GetRandomURL()
	if err != nil {
		// URL pool is empty - worker should exit
		return
	}

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
				// Mark URL as failed in registry
				w.registry.MarkFailed(url)
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

	// Set realistic browser headers
	w.setBrowserHeaders(req, url)

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

	// Download body
	var bytesDownloaded int64
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// Consume tokens for each chunk as we download
			// This prevents deadlock when multiple workers wait for large amounts of tokens
			if w.limiter != nil && w.limiter.IsEnabled() {
				if err := w.limiter.Wait(w.ctx, int64(n)); err != nil {
					return bytesDownloaded, err
				}
			}
			bytesDownloaded += int64(n)
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

	// Set realistic browser headers
	w.setBrowserHeaders(req, url)

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
				if err := w.limiter.Wait(w.ctx, int64(n)); err != nil {
					return bytesDownloaded, err
				}
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

// recoverFromPanic handles panics in the worker goroutine
func (w *Worker) recoverFromPanic() {
	if r := recover(); r != nil {
		// Log panic - worker will exit cleanly
		fmt.Printf("Worker %d panic recovered: %v\n", w.id, r)
	}
}
