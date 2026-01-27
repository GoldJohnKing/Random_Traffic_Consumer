package ui

import (
	"fmt"
	"strings"
	"time"

	"random-traffic-consumer/internal/config"
	"random-traffic-consumer/internal/limiter"
	"random-traffic-consumer/internal/stats"
)

// Formatter handles console output formatting
type Formatter struct {
	stats   *stats.Stats
	cfg     *config.Config
	limiter *limiter.TokenBucket
}

// NewFormatter creates a new console formatter
func NewFormatter(statsCollector *stats.Stats, cfg *config.Config, bwLimiter *limiter.TokenBucket) *Formatter {
	return &Formatter{
		stats:   statsCollector,
		cfg:     cfg,
		limiter: bwLimiter,
	}
}

// Render displays the current statistics on the console
func (f *Formatter) Render(remaining time.Duration, trafficProgress float64) {
	snapshot := f.stats.GetSnapshot()

	// Clear screen and move cursor to home position
	fmt.Print("\x1b[2J\x1b[H")

	// Build output
	var output []string

	// Title section
	output = append(output, f.drawTitle())

	// Traffic section
	output = append(output, f.drawTrafficSection(snapshot))

	// Time section
	output = append(output, f.drawTimeSection(remaining))

	// Requests section
	output = append(output, f.drawRequestsSection(snapshot))

	// Bandwidth limit section (optional)
	if f.limiter != nil && f.limiter.IsEnabled() {
		output = append(output, f.drawBandwidthLimitSection(snapshot))
	}

	// Print all sections
	fmt.Print(strings.Join(output, "\n"))
}

// drawTitle creates the title section
func (f *Formatter) drawTitle() string {
	const width = 60
	border := strings.Repeat("=", width)
	title := "Random Traffic Consumer - Running"
	padding := (width - len(title)) / 2
	if padding < 0 {
		padding = 0
	}

	var sb strings.Builder
	sb.WriteString(border + "\n")
	sb.WriteString(fmt.Sprintf("%s%s%s\n", strings.Repeat(" ", padding), title, strings.Repeat(" ", padding)))
	sb.WriteString(border + "\n")
	return sb.String()
}

// drawTrafficSection creates the traffic section
func (f *Formatter) drawTrafficSection(snapshot stats.Snapshot) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Traffic\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	// Downloaded row
	if f.cfg.Stop.Mode == "duration" {
		// No traffic limit - show only downloaded
		sb.WriteString(fmt.Sprintf("Downloaded:   %s\n", formatBytes(snapshot.TotalBytes)))
	} else {
		// Has traffic limit - show with percentage
		trafficPercent := (float64(snapshot.TotalBytes) / float64(int64(f.cfg.Stop.TrafficLimit))) * 100
		sb.WriteString(fmt.Sprintf("Downloaded:   %s / %s  (%.2f%%)\n",
			formatBytes(snapshot.TotalBytes),
			formatBytes(int64(f.cfg.Stop.TrafficLimit)),
			trafficPercent))
	}

	// Speed row
	sb.WriteString(fmt.Sprintf("Speed:        %s\n", formatMbps(snapshot.CurrentSpeed)))

	return sb.String()
}

// drawTimeSection creates the time section
func (f *Formatter) drawTimeSection(remaining time.Duration) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Time\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	elapsed := f.stats.GetElapsedTime()
	sb.WriteString(fmt.Sprintf("Elapsed:     %s\n", formatDuration(elapsed)))

	if f.cfg.Stop.Mode == "duration" || f.cfg.Stop.Mode == "both" {
		// Has time limit - show remaining and percentage
		timePercent := (1 - remaining.Seconds()/f.cfg.Stop.Duration.Seconds()) * 100
		sb.WriteString(fmt.Sprintf("Remaining:   %s / %s  (%.2f%%)\n",
			formatDuration(remaining),
			formatDuration(f.cfg.Stop.Duration),
			timePercent))
	}

	return sb.String()
}

// drawRequestsSection creates the requests section
func (f *Formatter) drawRequestsSection(snapshot stats.Snapshot) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Requests\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")
	sb.WriteString(fmt.Sprintf("Workers:      %d\n", f.cfg.Download.Workers))
	sb.WriteString(fmt.Sprintf("Success:      %d\n", snapshot.SuccessCount))
	sb.WriteString(fmt.Sprintf("Failed:       %d\n", snapshot.ErrorCount))
	return sb.String()
}

// drawBandwidthLimitSection creates the bandwidth limit section
func (f *Formatter) drawBandwidthLimitSection(snapshot stats.Snapshot) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Bandwidth Limit\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	limitMbps := float64(f.limiter.GetRate()) * 8 / 1_000_000
	utilization := (snapshot.CurrentSpeed / limitMbps) * 100
	if utilization > 100 {
		utilization = 100
	}

	sb.WriteString(fmt.Sprintf("Limit:        %s\n", formatMbps(limitMbps)))
	sb.WriteString(fmt.Sprintf("Usage:        %s\n", formatPercentage(utilization)))
	return sb.String()
}

// PrintStartupBanner prints the startup configuration
func (f *Formatter) PrintStartupBanner() {
	fmt.Println("===========================================")
	fmt.Println("   Random Traffic Consumer")
	fmt.Println("   Network Bandwidth Stability Test Tool")
	fmt.Println("===========================================")
	fmt.Printf("Workers: %d\n", f.cfg.Download.Workers)
	fmt.Printf("URLs in pool: %d\n", len(f.cfg.URLs))
	fmt.Printf("Stop mode: %s\n", f.cfg.Stop.Mode)

	if f.cfg.Stop.Mode == "duration" || f.cfg.Stop.Mode == "both" {
		fmt.Printf("Duration: %s\n", f.cfg.Stop.Duration)
	}
	if f.cfg.Stop.Mode == "traffic" || f.cfg.Stop.Mode == "both" {
		fmt.Printf("Traffic limit: %s\n", formatBytes(int64(f.cfg.Stop.TrafficLimit)))
	}

	if f.limiter != nil && f.limiter.IsEnabled() {
		fmt.Printf("Bandwidth limit: %.1f Mbps\n", float64(f.limiter.GetRate())*8/1_000_000)
	}

	fmt.Printf("Chunk size: ")
	if f.cfg.Download.ChunkSize == 0 {
		fmt.Println("Complete file")
	} else {
		fmt.Printf("%s\n", formatBytes(f.cfg.Download.ChunkSize))
	}

	fmt.Println("===========================================")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()
}

// PrintFinalStats prints final statistics on shutdown
func (f *Formatter) PrintFinalStats() {
	snapshot := f.stats.GetSnapshot()
	fmt.Println()
	fmt.Println("===========================================")
	fmt.Println("               Final Statistics")
	fmt.Println("===========================================")
	fmt.Printf("Total traffic: %s\n", formatBytes(snapshot.TotalBytes))
	fmt.Printf("Duration: %s\n", formatDuration(f.stats.GetElapsedTime()))
	fmt.Printf("Average speed: %.1f Mbps\n", snapshot.AverageSpeed)
	fmt.Printf("Successful downloads: %d\n", snapshot.SuccessCount)
	fmt.Printf("Failed downloads: %d\n", snapshot.ErrorCount)
	fmt.Println("===========================================")
}

// formatBytes formats a byte count into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGT"[exp])
}

// formatMbps formats a speed value in Mbps
func formatMbps(mbps float64) string {
	return fmt.Sprintf("%.2f Mbps", mbps)
}

// formatPercentage formats a percentage value
func formatPercentage(percent float64) string {
	return fmt.Sprintf("%.2f%%", percent)
}

// formatDuration formats a duration into HH:MM:SS
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
