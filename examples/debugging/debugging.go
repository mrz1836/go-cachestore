package main

import (
	"context"
	"fmt"

	"github.com/mrz1836/go-cachestore"
	"github.com/mrz1836/go-logger"
)

func main() {
	ctx := context.Background()

	// Create a new client (default is FreeCache)
	client, err := cachestore.NewClient(ctx, cachestore.WithDebugging())
	if err != nil {
		logger.Fatalln(err.Error())
	}
	defer client.Close(ctx)

	// Success!
	logger.Data(2, logger.DEBUG, "Engine loaded: "+client.Engine().String())
	logger.Data(2, logger.DEBUG, fmt.Sprintf("Debugging: %v", client.IsDebug()))
}
