package main

import (
	"context"

	"github.com/mrz1836/go-cachestore"
	"github.com/mrz1836/go-logger"
)

func main() {
	ctx := context.Background()

	// Create a new client
	client, err := cachestore.NewClient(ctx, cachestore.WithFreeCache())
	if err != nil {
		logger.Fatalln(err.Error())
	}
	defer client.Close(ctx)

	// Success!
	logger.Data(2, logger.DEBUG, "Engine loaded: "+client.Engine().String())
}
