// Package main shows how to use redis with the caching library
package main

import (
	"context"
	"time"

	"github.com/mrz1836/go-logger"

	"github.com/mrz1836/go-cachestore"
)

func main() {
	ctx := context.Background()

	// Create a new client
	client, err := cachestore.NewClient(ctx, cachestore.WithRedis(&cachestore.RedisConfig{
		DependencyMode:        false,
		MaxActiveConnections:  2,
		MaxConnectionLifetime: 240 * time.Second,
		MaxIdleConnections:    10,
		MaxIdleTimeout:        240 * time.Second,
		URL:                   "localhost:" + cachestore.DefaultRedisPort,
		UseTLS:                false,
	}))
	if err != nil {
		logger.Fatalln(err.Error())
	}
	defer client.Close(ctx)

	// Success!
	logger.Data(2, logger.DEBUG, "Engine loaded: "+client.Engine().String())
}
