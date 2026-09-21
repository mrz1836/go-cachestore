package cachestore

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"strings"

	"github.com/mrz1836/go-cache"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// loadRedisClient will load the cache client (redis or valkey)
func loadRedisClient(
	ctx context.Context,
	config *RedisConfig,
	newRelicEnabled bool,
) (*cache.Client, error) {
	// Check for a config
	if config == nil || config.URL == "" {
		return nil, ErrInvalidRedisConfig
	}

	// If NewRelic is enabled
	if newRelicEnabled {
		if txn := newrelic.FromContext(ctx); txn != nil {
			segment := txn.StartSegment("load_redis_client")
			segment.AddAttribute("url", config.URL)
			defer segment.End()
		}
	}

	// Build the TLS configuration (nil when TLS is not requested)
	tlsConfig, err := buildTLSConfig(config)
	if err != nil {
		return nil, err
	}

	// Attempt to create the client
	client, err := cache.ConnectWithOptions(ctx, cache.PoolOptions{
		URL:                  config.URL,
		MaxActiveConnections: config.MaxActiveConnections,
		IdleConnections:      config.MaxIdleConnections,
		MaxConnLifetime:      config.MaxConnectionLifetime,
		IdleTimeout:          config.MaxIdleTimeout,
		DependencyMode:       config.DependencyMode,
		NewRelicEnabled:      newRelicEnabled,
		TLSConfig:            tlsConfig,
		Username:             config.Username,
		Password:             config.Password,
	})
	if err != nil {
		return nil, err
	}

	// Test the connection if DependencyMode mode is off (no connection tested)
	if !config.DependencyMode { // Fire a ping to make sure it works!
		if err = cache.Ping(ctx, client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

// buildTLSConfig assembles a *tls.Config from the RedisConfig, or returns nil
// when TLS is not requested. An explicit config.TLSConfig always takes
// precedence over the individual TLS* fields.
func buildTLSConfig(config *RedisConfig) (*tls.Config, error) {
	// An explicit config wins
	if config.TLSConfig != nil {
		return config.TLSConfig, nil
	}

	// TLS is requested via the flag, a rediss:// URL, or any TLS field
	tlsRequested := config.UseTLS ||
		strings.HasPrefix(config.URL, RedissPrefix) ||
		config.TLSServerName != "" ||
		config.TLSInsecureSkipVerify ||
		config.TLSCACertPath != ""
	if !tlsRequested {
		return nil, nil //nolint:nilnil // nil config means "no TLS", not an error
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         config.TLSServerName,         // empty defaults to the endpoint host (SNI)
		InsecureSkipVerify: config.TLSInsecureSkipVerify, //nolint:gosec // opt-in, testing only
	}

	// Load a custom CA bundle when provided
	if config.TLSCACertPath != "" {
		pem, err := os.ReadFile(config.TLSCACertPath)
		if err != nil {
			return nil, errors.Join(ErrInvalidRedisCACert, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, ErrInvalidRedisCACert
		}
		tlsConfig.RootCAs = pool
	}

	return tlsConfig, nil
}
