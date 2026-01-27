package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"random-traffic-consumer/internal/config"
	"random-traffic-consumer/internal/downloader"
	"random-traffic-consumer/internal/limiter"
	"random-traffic-consumer/internal/stats"
	"random-traffic-consumer/internal/ui"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize components
	statsCollector := stats.NewStats()

	var bwLimiter *limiter.TokenBucket
	if cfg.BandwidthLimit.Enabled {
		bwLimiter = limiter.NewTokenBucket(int64(cfg.BandwidthLimit.Limit), true)
	}

	httpClient := downloader.NewHTTPClient(&cfg.HTTP, cfg.Download.Timeout)

	// Create worker pool
	pool := downloader.NewPool(cfg, httpClient, statsCollector, bwLimiter)

	// Create console formatter
	formatter := ui.NewFormatter(statsCollector, cfg, bwLimiter)

	// Create stop checker
	stopChecker := config.NewStopChecker(cfg, statsCollector)

	// Print startup banner
	formatter.PrintStartupBanner()

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to signal when stop condition is met
	stopChan := make(chan struct{}, 1)

	// Start workers
	pool.Start()

	// Start console refresh loop in background
	go func() {
		ticker := time.NewTicker(cfg.Output.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				statsCollector.UpdateSpeed()

				var remaining time.Duration
				var trafficProgress float64

				if cfg.Stop.Mode == "duration" || cfg.Stop.Mode == "both" {
					remaining = stopChecker.GetRemaining()
				}
				if cfg.Stop.Mode == "traffic" || cfg.Stop.Mode == "both" {
					trafficProgress = stopChecker.GetTrafficProgress()
				}

				formatter.Render(remaining, trafficProgress)
			}
		}
	}()

	// Start stop condition checker in background
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if stopChecker.ShouldStop() {
					stopChan <- struct{}{}
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Wait for signal or stop condition
	select {
	case <-sigChan:
		fmt.Println("\n\nReceived interrupt signal, shutting down...")
	case <-stopChan:
		fmt.Println("\n\nStop condition reached, shutting down...")
	}

	// Cancel context to stop all goroutines
	cancel()

	// Stop workers (waits for them to finish)
	pool.Stop()

	// Print final statistics
	formatter.PrintFinalStats()
}
