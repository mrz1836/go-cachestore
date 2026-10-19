package cachestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	// fillLockSuffix names the lock that makes processes share one load of a key
	fillLockSuffix = ":fill-lock"

	// defaultLoadLockTTL is LoadOptions.LockTTL's default
	defaultLoadLockTTL = 10 * time.Second

	// defaultLoadLockWait is LoadOptions.LockWait's default
	defaultLoadLockWait = 5 * time.Second

	// defaultLoadPollInterval is LoadOptions.PollInterval's default
	defaultLoadPollInterval = 50 * time.Millisecond
)

// LoadOptions tune GetOrLoadWithOptions. A field that is zero or negative takes
// its default.
type LoadOptions struct {
	// LockTTL is the fill lock's TTL, rounded up to whole seconds, and the longest
	// one load may run. It defaults to 10 seconds.
	LockTTL time.Duration

	// LockWait is how long a caller waits for another process's load before it
	// returns ErrLoadInProgress. It defaults to 5 seconds.
	LockWait time.Duration

	// PollInterval is how often a waiting caller re-reads the key. It defaults to
	// 50 milliseconds.
	PollInterval time.Duration
}

// withDefaults gives each zero or negative field its default, and rounds LockTTL
// up to whole seconds, as the fill lock counts them
func (o LoadOptions) withDefaults() LoadOptions {
	if o.LockTTL <= 0 {
		o.LockTTL = defaultLoadLockTTL
	}
	if secs, err := ttlSeconds(o.LockTTL); err == nil {
		o.LockTTL = time.Duration(secs) * time.Second
	}
	if o.LockWait <= 0 {
		o.LockWait = defaultLoadLockWait
	}
	if o.PollInterval <= 0 {
		o.PollInterval = defaultLoadPollInterval
	}
	return o
}

// GetOrLoad returns the value stored at key, decoded from JSON into a T. On a
// miss it calls load once, stores its result with ttl, and returns it.
//
// It is GetOrLoadWithOptions with the default LoadOptions:
//
//	report, err := cachestore.GetOrLoad(ctx, client, "report:daily", time.Minute, buildReport)
func GetOrLoad[T any](ctx context.Context, c ClientInterface, key string, ttl time.Duration,
	load func(context.Context) (T, error),
) (T, error) {
	return GetOrLoadWithOptions(ctx, c, key, ttl, load, LoadOptions{})
}

// GetOrLoadWithOptions returns the value stored at key, decoded from JSON into a
// T. On a miss it calls load once, stores its result as JSON with ttl (whole
// seconds, rounded up, as SetModel stores it), and returns it.
//
// Callers on the same *Client and key share one call to load; the others wait
// for its result. On Redis and Valkey, that call runs only while it holds the
// key's fill lock, "<key>:fill-lock", so processes share it too: a caller that
// finds the lock held re-reads the key every PollInterval, takes the lock itself
// once it is free and the key is still empty, and returns ErrLoadInProgress when
// LockWait passes without a value. FreeCache has no fill lock, so there the
// shared call is the whole guarantee. Another ClientInterface implementation
// shares nothing in one process, though the fill lock still applies.
//
// An error from load reaches every caller of that call, as load returned it, and
// nothing is stored: errors are never cached. A stored value that no longer
// decodes into a T counts as a miss, and the load replaces it. A cache error is
// returned as is.
//
// load runs with the first caller's context values but not its cancellation,
// under a deadline of LockTTL, so one caller leaving fails none of the others and
// a load never outlives its fill lock. Each caller returns its own context's
// error as soon as its context is done. A panic in load is raised again in each
// waiting caller, with the load's stack.
//
// T must round-trip through encoding/json, and each caller decodes its own copy.
// An empty key (after trimming) returns ErrKeyRequired, a ttl of zero or less
// ErrInvalidTTL, and a done ctx its own error, before any cache call.
func GetOrLoadWithOptions[T any](ctx context.Context, c ClientInterface, key string, ttl time.Duration,
	load func(context.Context) (T, error), opts LoadOptions,
) (T, error) {
	var zero T

	// Sanitize the key (trailing or leading spaces)
	key = strings.TrimSpace(key)

	// Require a key, a positive TTL, and a live context
	if len(key) == 0 {
		return zero, ErrKeyRequired
	}
	if ttl <= 0 {
		return zero, ErrInvalidTTL
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	// A hit: the stored JSON decodes into a T
	value, stored, err := readStored[T](ctx, c, key)
	if err != nil {
		return zero, err
	} else if stored != nil {
		return value, nil
	}

	// A miss: one fill per key at a time on this client; any other ClientInterface fills inline
	opts = opts.withDefaults()
	fill := func(fillCtx context.Context) ([]byte, error) {
		return runFill(fillCtx, c, key, ttl, load, opts)
	}
	if client, ok := c.(*Client); ok {
		stored, err = client.loads.do(ctx, key, fill)
	} else {
		stored, err = fill(ctx)
	}
	if err != nil {
		return zero, err
	}

	// Every caller decodes its own value from the stored JSON
	if err = json.Unmarshal(stored, &value); err != nil {
		return zero, err
	}
	return value, nil
}

// readStored reads key: when it holds JSON that decodes into a T, it returns the
// value and the JSON. A missing key, or a value that no longer decodes, returns
// neither
func readStored[T any](ctx context.Context, c ClientInterface, key string) (T, []byte, error) {
	var value T
	s, err := c.Get(ctx, key)
	if err != nil || len(s) == 0 {
		return value, nil, err
	}

	b := []byte(s)
	if json.Unmarshal(b, &value) != nil {
		var zero T
		return zero, nil, nil //nolint:nilerr // a value that no longer decodes is a miss, which the next load replaces
	}
	return value, b, nil
}

// runFill is one fill of key: it re-reads the key, takes the fill lock on Redis
// and Valkey, loads, and stores the result, returning the stored JSON
func runFill[T any](ctx context.Context, c ClientInterface, key string, ttl time.Duration,
	load func(context.Context) (T, error), opts LoadOptions,
) ([]byte, error) {
	// FreeCache is one process, so the shared call is the whole guarantee
	if c.Engine() != Redis {
		if _, stored, err := readStored[T](ctx, c, key); err != nil || stored != nil {
			return stored, err
		}
		return loadAndStore(ctx, c, key, ttl, load, opts)
	}

	// Redis and Valkey: load only while holding the fill lock
	lockKey := key + fillLockSuffix
	waitUntil := time.Now().Add(opts.LockWait)
	for {
		// Another caller, in this process or another, may have stored it
		if _, stored, err := readStored[T](ctx, c, key); err != nil || stored != nil {
			return stored, err
		}

		secret, err := c.WriteLock(ctx, lockKey, int64(opts.LockTTL/time.Second))
		if err == nil {
			// A failed release isn't returned: the value is stored, and the lock expires after LockTTL
			defer func() { _, _ = c.ReleaseLock(context.WithoutCancel(ctx), lockKey, secret) }()

			// Another process may have stored it before this one took the lock
			if _, stored, readErr := readStored[T](ctx, c, key); readErr != nil || stored != nil {
				return stored, readErr
			}
			return loadAndStore(ctx, c, key, ttl, load, opts)
		}
		if !errors.Is(err, ErrLockExists) {
			return nil, err
		}

		// Another process holds the fill lock: wait for its value, up to LockWait
		if !time.Now().Before(waitUntil) {
			return nil, ErrLoadInProgress
		}
		if err = waitForRetry(ctx, opts.PollInterval); err != nil {
			return nil, err
		}
	}
}

// loadAndStore runs load, bounded by LockTTL, and stores its result at key with
// ttl, returning the stored JSON. A failed load stores nothing
func loadAndStore[T any](ctx context.Context, c ClientInterface, key string, ttl time.Duration,
	load func(context.Context) (T, error), opts LoadOptions,
) ([]byte, error) {
	// A load never outlives its fill lock, which another process may take after LockTTL
	loadCtx, cancel := context.WithTimeout(ctx, opts.LockTTL)
	defer cancel()

	value, err := load(loadCtx)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	// SetModel stores a json.RawMessage as given, under the TTL rule of every value
	if err = c.SetModel(ctx, key, json.RawMessage(b), ttl); err != nil {
		return nil, err
	}
	return b, nil
}

// loadGroup runs at most one fill per key at a time for one Client; callers
// that arrive while it runs wait for its result
type loadGroup struct {
	mu    sync.Mutex
	fills map[string]*fillCall
}

// fillCall is one running fill: done closes when it ends, and then exactly one
// of value (the stored JSON), err, or panicked is set
type fillCall struct {
	done     chan struct{}
	value    []byte
	err      error
	panicked *loadPanicError
}

// do joins the running fill for key, or starts one, detached from ctx's
// cancellation; then it waits for the fill's result or for ctx to be done
func (g *loadGroup) do(ctx context.Context, key string, fill func(context.Context) ([]byte, error)) ([]byte, error) {
	g.mu.Lock()
	if g.fills == nil {
		g.fills = make(map[string]*fillCall)
	}
	call, running := g.fills[key]
	if !running {
		call = &fillCall{done: make(chan struct{})}
		g.fills[key] = call
		go g.run(context.WithoutCancel(ctx), key, call, fill)
	}
	g.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.done:
	}

	// A panic in the load is raised again in every caller that waited for it
	if call.panicked != nil {
		panic(call.panicked)
	}
	return call.value, call.err
}

// run runs one fill and hands its result to every waiting caller; a panic is
// recovered with the fill's stack, to be raised again in each of them
func (g *loadGroup) run(ctx context.Context, key string, call *fillCall, fill func(context.Context) ([]byte, error)) {
	defer func() {
		if r := recover(); r != nil {
			call.panicked = &loadPanicError{value: r, stack: debug.Stack()}
		}

		g.mu.Lock()
		delete(g.fills, key)
		g.mu.Unlock()
		close(call.done)
	}()

	call.value, call.err = fill(ctx)
}

// loadPanicError carries a panic out of a load, with the load's stack, to be
// raised again in each caller that waited for it
type loadPanicError struct {
	value any
	stack []byte
}

// Error returns the panic's value, then the load's stack
func (e *loadPanicError) Error() string {
	return fmt.Sprintf("load panicked: %v\n\n%s", e.value, e.stack)
}
