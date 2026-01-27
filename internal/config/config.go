package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// BandwidthSize represents a bandwidth value in bytes per second.
// It implements yaml.Unmarshaler to parse strings like "200 Mbps".
type BandwidthSize int64

// UnmarshalYAML parses a bandwidth string like "200 Mbps" into bytes per second.
func (b *BandwidthSize) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	parsed, err := parseBandwidth(s)
	if err != nil {
		return err
	}

	*b = BandwidthSize(parsed)
	return nil
}

// TrafficSize represents a data size in bytes.
// It implements yaml.Unmarshaler to parse strings like "100 GB".
type TrafficSize int64

// UnmarshalYAML parses a traffic size string like "100 GB" into bytes.
func (t *TrafficSize) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	parsed, err := parseTrafficSizeString(s)
	if err != nil {
		return err
	}

	*t = TrafficSize(parsed)
	return nil
}

// Config represents the application configuration
type Config struct {
	URLs          []string        `yaml:"urls"`
	Download      DownloadConfig  `yaml:"download"`
	Stop          StopConfig      `yaml:"stop"`
	BandwidthLimit BandwidthConfig `yaml:"bandwidth_limit"`
	HTTP          HTTPConfig      `yaml:"http"`
	Output        OutputConfig    `yaml:"output"`
}

// DownloadConfig contains download-related settings
type DownloadConfig struct {
	Workers      int                `yaml:"workers"`
	Timeout      time.Duration      `yaml:"timeout"`
	Retry        int                `yaml:"retry"`
	ChunkSize    int64              `yaml:"chunk_size"`
	StaggerStart StaggerStartConfig `yaml:"stagger_start"`
}

// StaggerStartConfig contains staggered worker startup settings
type StaggerStartConfig struct {
	Enabled    bool          `yaml:"enabled"`
	BaseDelay  time.Duration `yaml:"base_delay"`
	Jitter     time.Duration `yaml:"jitter"`
}

// StopConfig contains stop condition settings
type StopConfig struct {
	Mode         string      `yaml:"mode"` // duration, traffic, both
	Duration     time.Duration `yaml:"duration"`
	TrafficLimit TrafficSize `yaml:"traffic_limit"` // e.g., "500 GB"
}

// BandwidthConfig contains bandwidth limiting settings
type BandwidthConfig struct {
	Enabled bool         `yaml:"enabled"`
	Limit   BandwidthSize `yaml:"limit"` // e.g., "200 Mbps"
}

// HTTPConfig contains HTTP client settings
type HTTPConfig struct {
	MaxIdleConns        int           `yaml:"max_idle_conns"`
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout"`
	DisableKeepAlives   bool          `yaml:"disable_keepalive"`
}

// OutputConfig contains output settings
type OutputConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
}

// LoadConfig loads and validates configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	// Validate URLs
	if len(c.URLs) == 0 {
		return fmt.Errorf("urls cannot be empty")
	}
	for i, url := range c.URLs {
		if url == "" || !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return fmt.Errorf("url[%d] is invalid: must start with http:// or https://", i)
		}
	}

	// Validate download settings
	if c.Download.Workers <= 0 {
		return fmt.Errorf("download.workers must be positive")
	}
	if c.Download.Timeout <= 0 {
		return fmt.Errorf("download.timeout must be positive")
	}
	if c.Download.Retry < 0 {
		return fmt.Errorf("download.retry cannot be negative")
	}
	if c.Download.ChunkSize < 0 {
		return fmt.Errorf("download.chunk_size cannot be negative")
	}

	// Validate stop mode
	validModes := map[string]bool{"duration": true, "traffic": true, "both": true}
	if !validModes[c.Stop.Mode] {
		return fmt.Errorf("stop.mode must be one of: duration, traffic, both")
	}

	// Validate duration requirements
	if c.Stop.Mode == "duration" || c.Stop.Mode == "both" {
		if c.Stop.Duration <= 0 {
			return fmt.Errorf("stop.duration is required for mode '%s'", c.Stop.Mode)
		}
	}

	// Validate traffic limit requirements
	if c.Stop.Mode == "traffic" || c.Stop.Mode == "both" {
		if c.Stop.TrafficLimit <= 0 {
			return fmt.Errorf("stop.traffic_limit is required for mode '%s'", c.Stop.Mode)
		}
	}

	// Validate bandwidth limit
	if c.BandwidthLimit.Enabled && int64(c.BandwidthLimit.Limit) <= 0 {
		return fmt.Errorf("bandwidth_limit.limit must be positive when enabled")
	}

	// Validate HTTP settings
	if c.HTTP.MaxIdleConns < 0 {
		return fmt.Errorf("http.max_idle_conns cannot be negative")
	}
	if c.HTTP.IdleConnTimeout < 0 {
		return fmt.Errorf("http.idle_conn_timeout cannot be negative")
	}

	// Validate output settings
	if c.Output.RefreshInterval <= 0 {
		return fmt.Errorf("output.refresh_interval must be positive")
	}

	// Set default values for staggered startup if enabled but not specified
	if c.Download.StaggerStart.Enabled {
		if c.Download.StaggerStart.BaseDelay == 0 {
			c.Download.StaggerStart.BaseDelay = 200 * time.Millisecond
		}
		if c.Download.StaggerStart.Jitter == 0 {
			c.Download.StaggerStart.Jitter = 50 * time.Millisecond
		}
		// Validate staggered startup settings
		if c.Download.StaggerStart.BaseDelay < 0 {
			return fmt.Errorf("download.stagger_start.base_delay cannot be negative")
		}
		if c.Download.StaggerStart.Jitter < 0 {
			return fmt.Errorf("download.stagger_start.jitter cannot be negative")
		}
		if c.Download.StaggerStart.Jitter > c.Download.StaggerStart.BaseDelay {
			return fmt.Errorf("download.stagger_start.jitter must not exceed base_delay")
		}
	}

	return nil
}

// parseBandwidth parses bandwidth string like "200 Mbps" into bytes per second.
// Supports bps units (Kbps, Mbps, Gbps) using decimal (1000-based) prefixes.
// bps units are divided by 8 to convert bits to bytes.
func parseBandwidth(s string) (int64, error) {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid format: expected '<value> <unit>', got '%s'", s)
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %w", err)
	}

	if value <= 0 {
		return 0, fmt.Errorf("value must be positive")
	}

	unit := strings.ToUpper(parts[1])
	var multiplier float64

	switch unit {
	case "KBPS", "Kbps":
		multiplier = 1000 / 8     // Kbps to bytes/s (decimal)
	case "MBPS", "Mbps":
		multiplier = 1000 * 1000 / 8  // Mbps to bytes/s (decimal)
	case "GBPS", "Gbps":
		multiplier = 1000 * 1000 * 1000 / 8  // Gbps to bytes/s (decimal)
	default:
		return 0, fmt.Errorf("unknown unit: %s (supported: Kbps, Mbps, Gbps)", unit)
	}

	return int64(value * multiplier), nil
}

// parseTrafficSizeString parses a traffic size string like "100 GB" into bytes
func parseTrafficSizeString(s string) (int64, error) {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid format: expected '<value> <unit>', got '%s'", s)
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %w", err)
	}

	if value <= 0 {
		return 0, fmt.Errorf("value must be positive")
	}

	unit := strings.ToUpper(parts[1])
	var multiplier float64

	switch unit {
	case "B", "BYTE":
		multiplier = 1
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit: %s (supported: B, KB, MB, GB, TB)", unit)
	}

	return int64(value * multiplier), nil
}

// ParseDurationString parses a duration string like "1h30m" into time.Duration
func ParseDurationString(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
