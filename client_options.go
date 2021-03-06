package cachestore

import (
	"context"
	"strings"

	"github.com/coocood/freecache"
	"github.com/mrz1836/go-cache"
	zLogger "github.com/mrz1836/go-logger"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// ClientOps allow functional options to be supplied
// that overwrite default client options.
type ClientOps func(c *clientOptions)

// defaultClientOptions will return an clientOptions struct with the default settings
//
// Useful for starting with the default and then modifying as needed
func defaultClientOptions() *clientOptions {

	// Set the default options
	return &clientOptions{
		debug:           false,
		engine:          Empty,
		freeCache:       nil,
		newRelicEnabled: false,
		redisConfig:     &RedisConfig{},
	}
}

// getTxnCtx will check for an existing transaction
func (c *clientOptions) getTxnCtx(ctx context.Context) context.Context {
	if c.newRelicEnabled {
		txn := newrelic.FromContext(ctx)
		if txn != nil {
			ctx = newrelic.NewContext(ctx, txn)
		}
	}
	return ctx
}

// WithNewRelic will enable the NewRelic wrapper
func WithNewRelic() ClientOps {
	return func(c *clientOptions) {
		c.newRelicEnabled = true
	}
}

// WithDebugging will enable debugging mode
func WithDebugging() ClientOps {
	return func(c *clientOptions) {
		c.debug = true
	}
}

// WithRedis will set the redis configuration
func WithRedis(redisConfig *RedisConfig) ClientOps {
	return func(c *clientOptions) {

		// Don't panic if nil is passed
		if redisConfig == nil {
			return
		}

		// Add prefix if missing
		if !strings.Contains(redisConfig.URL, RedisPrefix) {
			redisConfig.URL = RedisPrefix + redisConfig.URL
		}

		// Set the config and engine
		c.redisConfig = redisConfig
		c.engine = Redis
		c.redis = nil // If you load via config, remove the connection

		// Set any defaults
		if c.redisConfig.MaxIdleTimeout.String() == emptyTimeDuration {
			c.redisConfig.MaxIdleTimeout = DefaultRedisMaxIdleTimeout
		}
	}
}

// WithRedisConnection will set an existing redis connection (read & write)
func WithRedisConnection(redisClient *cache.Client) ClientOps {
	return func(c *clientOptions) {
		if redisClient != nil {
			c.redis = redisClient
			c.engine = Redis
			c.redisConfig = nil // If you load an existing connection, config is not needed
		}
	}
}

// WithFreeCache will set the cache to local memory using FreeCache
func WithFreeCache() ClientOps {
	return func(c *clientOptions) {
		c.engine = FreeCache
	}
}

// WithFreeCacheConnection will set the cache to use an existing FreeCache connection
func WithFreeCacheConnection(client *freecache.Cache) ClientOps {
	return func(c *clientOptions) {
		if client != nil {
			c.engine = FreeCache
			c.freeCache = client
		}
	}
}

// WithLogger will set the custom logger interface
func WithLogger(customLogger zLogger.GormLoggerInterface) ClientOps {
	return func(c *clientOptions) {
		if customLogger != nil {
			c.logger = customLogger
		}
	}
}
