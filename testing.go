package cachestore

import (
	"context"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mrz1836/go-cache"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/require"
)

// loadMockRedis will load a mocked redis connection
func loadMockRedis(
	idleTimeout time.Duration, //nolint:unparam // same param but for testing
	maxConnTime time.Duration, //nolint:unparam // same param but for testing
	maxActive int, //nolint:unparam // same param but for testing
	maxIdle int, //nolint:unparam // same param but for testing
) (client *cache.Client, conn *redigomock.Conn) {
	conn = redigomock.NewConn()
	client = &cache.Client{
		DependencyScriptSha: "",
		Pool: &redis.Pool{
			Dial:            func() (redis.Conn, error) { return conn, nil },
			IdleTimeout:     idleTimeout,
			MaxActive:       maxActive,
			MaxConnLifetime: maxConnTime,
			MaxIdle:         maxIdle,
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				if time.Since(t) < time.Minute {
					return nil
				}
				_, doErr := c.Do(cache.PingCommand)
				return doErr
			},
		},
		ScriptsLoaded: nil,
	}
	return client, conn
}

// loadRealRedis will load a real redis connection
func loadRealRedis(
	ctx context.Context,
	connectionURL string,
	idleTimeout time.Duration,
	maxConnTime time.Duration,
	maxActive int,
	maxIdle int,
	dependency bool,
	newRelic bool,
) (client *cache.Client, conn redis.Conn, err error) {
	if client, err = cache.Connect(
		ctx,
		connectionURL,
		maxActive,
		maxIdle,
		maxConnTime,
		idleTimeout,
		dependency,
		newRelic,
	); err != nil {
		return nil, nil, err
	}

	conn, err = client.GetConnectionWithContext(ctx)
	return client, conn, err
}

// getNewRelicApp will return a dummy new relic app
func getNewRelicApp(appName string) (*newrelic.Application, error) {
	if len(appName) == 0 {
		return nil, ErrAppNameRequired
	}
	return newrelic.NewApplication(
		func(config *newrelic.Config) {
			config.AppName = appName
			config.DistributedTracer.Enabled = true
			config.Enabled = false
		},
	)
}

// getNewRelicCtx will return a dummy ctx
func getNewRelicCtx(t *testing.T, appName, txnName string) context.Context {
	// Load new relic (dummy)
	newRelic, err := getNewRelicApp(appName)
	require.NoError(t, err)
	require.NotNil(t, newRelic)

	// Create new relic tx
	return newrelic.NewContext(
		context.Background(), newRelic.StartTransaction(txnName),
	)
}
