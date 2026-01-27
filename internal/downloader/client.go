package downloader

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"random-traffic-consumer/internal/config"
)

// NewHTTPClient creates a new HTTP client with configured transport settings
func NewHTTPClient(cfg *config.HTTPConfig, timeout time.Duration) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: 10, // Reasonable default per host
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableKeepAlives:   cfg.DisableKeepAlives,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Secure by default
		},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxConnsPerHost:       0, // Unlimited
		ResponseHeaderTimeout: timeout,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}
