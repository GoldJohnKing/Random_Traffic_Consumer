package downloader

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

// URLRegistry manages a pool of URLs with health tracking.
// It is thread-safe and shared across all workers.
type URLRegistry struct {
	mu              sync.RWMutex
	urls            []string          // Available URLs
	failureCount    map[string]int    // Per-URL failure counter
	maxRetries      int               // Remove URL after this many failures
	poolCtx         context.Context   // Context for shutdown signalling
	rand            *rand.Rand        // Thread-safe random generator
	originalURLs    []string          // Original URL list for recovery
	resetInterval   time.Duration     // Interval to reset failure counters
	lastResetTime   time.Time         // Last time failure counters were reset
}

// NewURLRegistry creates a new URLRegistry with the given URLs and configuration.
func NewURLRegistry(urls []string, maxRetries int, poolCtx context.Context) *URLRegistry {
	// Create a copy of the URLs slice to avoid external modification
	urlsCopy := make([]string, len(urls))
	copy(urlsCopy, urls)

	// Initialize failure counters for all URLs
	failureCount := make(map[string]int)
	for _, url := range urls {
		failureCount[url] = 0
	}

	return &URLRegistry{
		urls:         urlsCopy,
		failureCount: failureCount,
		maxRetries:   maxRetries,
		poolCtx:      poolCtx,
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetRandomURL returns a random URL from the available pool.
// Returns an error if the pool is empty.
func (r *URLRegistry) GetRandomURL() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if pool is empty
	if len(r.urls) == 0 {
		return "", fmt.Errorf("URL pool is empty")
	}

	// Select random URL
	idx := r.rand.Intn(len(r.urls))
	return r.urls[idx], nil
}

// MarkFailed increments the failure counter for a URL.
// If the failure count reaches maxRetries, the URL is permanently removed from the pool.
func (r *URLRegistry) MarkFailed(url string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Increment failure counter
	r.failureCount[url]++

	// Check if URL should be removed
	if r.failureCount[url] >= r.maxRetries {
		// Remove URL from pool
		for i, u := range r.urls {
			if u == url {
				r.urls = append(r.urls[:i], r.urls[i+1:]...)
				break
			}
		}
		// Log removal to stderr
		fmt.Fprintf(os.Stderr, "URL %s removed after %d failures\n", url, r.failureCount[url])
	}
}

// IsEmpty returns true if there are no URLs available in the pool.
func (r *URLRegistry) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.urls) == 0
}
