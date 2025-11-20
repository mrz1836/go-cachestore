// Package main shows how to create a new client
package main

import (
	"context"

	"github.com/mrz1836/go-logger"

	"github.com/mrz1836/go-cachestore"
)

func main() {
	ctx := context.Background()

	// Create a new client (default is FreeCache)
	client, err := cachestore.NewClient(ctx)
	if err != nil {
		logger.Fatalln(err.Error())
	}
	defer client.Close(ctx)

	// Success!
	logger.Data(2, logger.DEBUG, "Engine loaded: "+client.Engine().String())
}
