package downloader

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

// URLRegistry manages a pool of URLs with health tracking.
// It is thread-safe and shared across all workers.
type URLRegistry struct {
	mu            sync.RWMutex
	urls          []string       // Available URLs
	failureCount  map[string]int // Per-URL failure counter
	maxRetries    int            // Remove URL after this many failures
	poolCtx       context.Context
	rand          *rand.Rand // Thread-safe random generator
	originalURLs  []string   // Original URL list for recovery
	resetInterval time.Duration
	lastResetTime time.Time
	configPath    string     // Path to config file for persistence
	fileMu        sync.Mutex // Mutex for file operations
}

// NewURLRegistry creates a new URLRegistry with the given URLs and configuration.
func NewURLRegistry(urls []string, maxRetries int, poolCtx context.Context, configPath string) *URLRegistry {
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
		configPath:   configPath,
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

		// Persist to config file
		if r.configPath != "" {
			r.persistExcludedURL(url)
		}
	}
}

// persistExcludedURL comments out the excluded URL in the config file.
// This operation is thread-safe and preserves file formatting.
func (r *URLRegistry) persistExcludedURL(url string) {
	r.fileMu.Lock()
	defer r.fileMu.Unlock()

	// Read config file
	file, err := os.Open(r.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open config file for persistence: %v\n", err)
		return
	}
	defer file.Close()

	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		// Check if this line contains the URL (not already commented)
		if !found && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Look for pattern: - "url" or - 'url'
			if strings.Contains(line, "- \""+url+"\"") || strings.Contains(line, "- '"+url+"'") {
				// Comment out this line
				lines = append(lines, "# "+line)
				found = true
				continue
			}
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read config file: %v\n", err)
		return
	}

	if !found {
		fmt.Fprintf(os.Stderr, "Warning: URL not found in config file for persistence: %s\n", url)
		return
	}

	// Write back to file
	output, err := os.Create(r.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create config file for persistence: %v\n", err)
		return
	}
	defer output.Close()

	writer := bufio.NewWriter(output)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}
	if err := writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write config file: %v\n", err)
	}
}

// IsEmpty returns true if there are no URLs available in the pool.
func (r *URLRegistry) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.urls) == 0
}
