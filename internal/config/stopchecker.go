package config

import (
	"time"

	"random-traffic-consumer/internal/stats"
)

// StopChecker manages stop condition checking
type StopChecker struct {
	mode         string
	startTime    time.Time
	duration     time.Duration
	trafficLimit int64
	stats        *stats.Stats
}

// NewStopChecker creates a new stop condition checker
func NewStopChecker(cfg *Config, statsCollector *stats.Stats) *StopChecker {
	return &StopChecker{
		mode:         cfg.Stop.Mode,
		startTime:    time.Now(),
		duration:     cfg.Stop.Duration,
		trafficLimit: int64(cfg.Stop.TrafficLimit),
		stats:        statsCollector,
	}
}

// ShouldStop returns true if the stop condition has been met
func (sc *StopChecker) ShouldStop() bool {
	switch sc.mode {
	case "duration":
		return sc.checkDuration()
	case "traffic":
		return sc.checkTraffic()
	case "both":
		return sc.checkDuration() || sc.checkTraffic()
	default:
		return false
	}
}

// checkDuration checks if time limit has been reached
func (sc *StopChecker) checkDuration() bool {
	elapsed := time.Since(sc.startTime)
	return elapsed >= sc.duration
}

// checkTraffic checks if traffic limit has been reached
func (sc *StopChecker) checkTraffic() bool {
	totalBytes := sc.stats.GetTotalBytes()
	return totalBytes >= sc.trafficLimit
}

// GetRemaining returns the remaining time (for duration mode)
func (sc *StopChecker) GetRemaining() time.Duration {
	elapsed := time.Since(sc.startTime)
	remaining := sc.duration - elapsed
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// GetTrafficProgress returns the traffic progress as a percentage
func (sc *StopChecker) GetTrafficProgress() float64 {
	if sc.trafficLimit == 0 {
		return 0
	}
	totalBytes := sc.stats.GetTotalBytes()
	return (float64(totalBytes) / float64(sc.trafficLimit)) * 100
}

// GetTimeProgress returns the time progress as a percentage
func (sc *StopChecker) GetTimeProgress() float64 {
	if sc.duration == 0 {
		return 0
	}
	elapsed := time.Since(sc.startTime)
	return (elapsed.Seconds() / sc.duration.Seconds()) * 100
}
