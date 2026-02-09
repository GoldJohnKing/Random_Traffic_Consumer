package downloader

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"random-traffic-consumer/internal/config"
	"random-traffic-consumer/internal/limiter"
	"random-traffic-consumer/internal/stats"
)

// Pool manages a pool of download workers
type Pool struct {
	workers    []*Worker
	count      int
	registry   *URLRegistry
	client     *http.Client
	stats      *stats.Stats
	limiter    *limiter.TokenBucket
	dlConfig   *config.DownloadConfig
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	configPath string // Path to config file for URL persistence
}

// NewPool creates a new worker pool
func NewPool(
	cfg *config.Config,
	httpClient *http.Client,
	statsCollector *stats.Stats,
	bwLimiter *limiter.TokenBucket,
	configPath string,
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	// Create URLRegistry with config URLs, retry count, and config path
	registry := NewURLRegistry(cfg.URLs, cfg.Download.Retry, ctx, configPath)

	return &Pool{
		count:      cfg.Download.Workers,
		registry:   registry,
		client:     httpClient,
		stats:      statsCollector,
		limiter:    bwLimiter,
		dlConfig:   &cfg.Download,
		ctx:        ctx,
		cancel:     cancel,
		configPath: configPath,
	}
}

// Start starts all workers in the pool
func (p *Pool) Start() {
	p.workers = make([]*Worker, p.count)

	for i := 0; i < p.count; i++ {
		// Apply staggered startup delay if enabled
		if p.dlConfig.StaggerStart.Enabled {
			delay := p.calculateStaggerDelay(i)
			if delay > 0 {
				time.Sleep(delay)
			}
		}

		worker := NewWorker(
			i,
			p.registry,
			p.client,
			p.stats,
			p.limiter,
			p.dlConfig,
			p.ctx,
		)
		p.workers[i] = worker
		worker.Start(&p.wg)
	}

	fmt.Printf("Started %d workers\n", p.count)

	// Start pool monitor goroutine
	p.startPoolMonitor()
}

// startPoolMonitor monitors the URL pool for exhaustion and terminates when empty
func (p *Pool) startPoolMonitor() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				// Pool already shutting down
				return
			case <-ticker.C:
				if p.registry.IsEmpty() {
					fmt.Println("All URLs exhausted, terminating...")
					p.cancel()
					return
				}
			}
		}
	}()
}

// calculateStaggerDelay calculates the startup delay for a worker using the formula:
// delay = (worker_index × base_delay) + random(-jitter, +jitter)
func (p *Pool) calculateStaggerDelay(workerIndex int) time.Duration {
	baseDelay := p.dlConfig.StaggerStart.BaseDelay
	jitter := p.dlConfig.StaggerStart.Jitter

	// Calculate base delay: worker_index × base_delay
	baseDelayForWorker := time.Duration(workerIndex) * baseDelay

	// Calculate jitter: random value in [-jitter, +jitter]
	jitterOffset := time.Duration(rand.Int63n(2*int64(jitter)+1)) - jitter

	// Total delay = base + jitter
	totalDelay := baseDelayForWorker + jitterOffset

	// Ensure non-negative (in case jitter pushes it below zero)
	if totalDelay < 0 {
		totalDelay = 0
	}

	return totalDelay
}

// Stop stops all workers in the pool
func (p *Pool) Stop() {
	fmt.Println("Stopping workers...")
	p.cancel()
	p.wg.Wait()
	fmt.Println("All workers stopped")
}

// GetContext returns the pool's context for external cancellation
func (p *Pool) GetContext() context.Context {
	return p.ctx
}

// Cancel cancels the pool's context (for external shutdown)
func (p *Pool) Cancel() {
	p.cancel()
}
