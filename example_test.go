package cachestore_test

import (
	"context"
	"fmt"
	"time"

	cachestore "github.com/mrz1836/go-cachestore"
)

// ExampleGetOrLoad reads one key three times: the first read misses and calls
// load, and the next two find the stored value.
func ExampleGetOrLoad() {
	ctx := context.Background()
	client, err := cachestore.NewClient(ctx, cachestore.WithFreeCache())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.Close(ctx)

	// load stands in for an expensive query
	loads := 0
	load := func(context.Context) (int, error) {
		loads++
		return 42, nil
	}

	var answer int
	for range 3 {
		if answer, err = cachestore.GetOrLoad(ctx, client, "answer", time.Minute, load); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(answer)
	}
	fmt.Println("loads:", loads)

	// Output:
	// 42
	// 42
	// 42
	// loads: 1
}
