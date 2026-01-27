package stats

import (
	"sync/atomic"
	"time"
)

// Stats represents thread-safe statistics collection
type Stats struct {
	totalBytes   int64 // Atomic counter
	successCount int64 // Atomic counter
	errorCount   int64 // Atomic counter

	startTime    time.Time
	lastBytes    int64
	lastTime     time.Time
	speed        float64 // Current speed in Mbps
	avgSpeed     float64 // Average speed in Mbps
}

// Snapshot represents a point-in-time view of statistics
type Snapshot struct {
	TotalBytes   int64
	SuccessCount int64
	ErrorCount   int64
	CurrentSpeed float64 // Mbps
	AverageSpeed float64 // Mbps
}

// NewStats creates a new statistics collector
func NewStats() *Stats {
	return &Stats{
		startTime: time.Now(),
		lastTime:  time.Now(),
	}
}

// RecordBytes records n bytes downloaded
func (s *Stats) RecordBytes(n int64) {
	atomic.AddInt64(&s.totalBytes, n)
}

// RecordSuccess records a successful download
func (s *Stats) RecordSuccess() {
	atomic.AddInt64(&s.successCount, 1)
}

// RecordError records a failed download
func (s *Stats) RecordError() {
	atomic.AddInt64(&s.errorCount, 1)
}

// GetSnapshot returns a snapshot of current statistics
func (s *Stats) GetSnapshot() Snapshot {
	totalBytes := atomic.LoadInt64(&s.totalBytes)
	successCount := atomic.LoadInt64(&s.successCount)
	errorCount := atomic.LoadInt64(&s.errorCount)

	return Snapshot{
		TotalBytes:   totalBytes,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		CurrentSpeed: s.speed,
		AverageSpeed: s.avgSpeed,
	}
}

// UpdateSpeed calculates and updates current and average speeds
func (s *Stats) UpdateSpeed() {
	now := time.Now()
	totalBytes := atomic.LoadInt64(&s.totalBytes)

	// Calculate current speed (bytes since last update)
	bytesSinceUpdate := totalBytes - s.lastBytes
	elapsed := now.Sub(s.lastTime).Seconds()

	if elapsed > 0 {
		// Convert to Mbps: (bytes * 8) / 1,000,000 / seconds
		s.speed = (float64(bytesSinceUpdate) * 8) / (1_000_000 * elapsed)
	}

	s.lastBytes = totalBytes
	s.lastTime = now

	// Calculate average speed
	totalElapsed := now.Sub(s.startTime).Seconds()
	if totalElapsed > 0 {
		s.avgSpeed = (float64(totalBytes) * 8) / (1_000_000 * totalElapsed)
	}
}

// GetTotalBytes returns the total bytes downloaded
func (s *Stats) GetTotalBytes() int64 {
	return atomic.LoadInt64(&s.totalBytes)
}

// GetSuccessCount returns the success count
func (s *Stats) GetSuccessCount() int64 {
	return atomic.LoadInt64(&s.successCount)
}

// GetErrorCount returns the error count
func (s *Stats) GetErrorCount() int64 {
	return atomic.LoadInt64(&s.errorCount)
}

// GetElapsedTime returns the elapsed time since start
func (s *Stats) GetElapsedTime() time.Duration {
	return time.Since(s.startTime)
}
