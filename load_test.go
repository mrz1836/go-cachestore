package cachestore

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coocood/freecache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errLoadFailed is the error the failing loads return
var errLoadFailed = errors.New("load failed")

// loadedValue is what the tests load: it holds a slice, so each caller's copy can be told apart
type loadedValue struct {
	N     int      `json:"n"`
	Items []string `json:"items"`
}

// loadCounter counts the calls of the loads it makes
type loadCounter struct {
	calls atomic.Int32
}

// returning is a load that counts its call and returns value
func (l *loadCounter) returning(value loadedValue) func(context.Context) (loadedValue, error) {
	return func(context.Context) (loadedValue, error) {
		l.calls.Add(1)
		return value, nil
	}
}

// blockedUntil is a load that counts its call, then waits for release to close before it returns value
func (l *loadCounter) blockedUntil(release <-chan struct{}, value loadedValue) func(context.Context) (loadedValue, error) {
	return func(context.Context) (loadedValue, error) {
		l.calls.Add(1)
		<-release
		return value, nil
	}
}

// failingAfter is a load that counts its call, then waits for release to close before it fails
func (l *loadCounter) failingAfter(release <-chan struct{}) func(context.Context) (loadedValue, error) {
	return func(context.Context) (loadedValue, error) {
		l.calls.Add(1)
		<-release
		return loadedValue{}, errLoadFailed
	}
}

// waitUntilLoading waits until a load has started
func (l *loadCounter) waitUntilLoading(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool { return l.calls.Load() > 0 }, 5*time.Second, time.Millisecond)
}

// letCallersJoin gives the callers started before it time to reach the running load
func letCallersJoin() {
	time.Sleep(50 * time.Millisecond)
}

// loadResult is what one GetOrLoad call returned
type loadResult struct {
	value loadedValue
	err   error
}

// getOrLoadAsync loads testKey through GetOrLoadWithOptions in a new goroutine, and delivers its result
func getOrLoadAsync(ctx context.Context, c ClientInterface, load func(context.Context) (loadedValue, error),
	opts LoadOptions,
) <-chan loadResult {
	result := make(chan loadResult, 1)
	go func() {
		value, err := GetOrLoadWithOptions(ctx, c, testKey, time.Minute, load, opts)
		result <- loadResult{value: value, err: err}
	}()
	return result
}

// receive waits for a call's result, and fails the test when it takes over five seconds
func receive(t *testing.T, result <-chan loadResult) loadResult {
	t.Helper()
	select {
	case r := <-result:
		return r
	case <-time.After(5 * time.Second):
		require.FailNow(t, "GetOrLoad did not return")
		return loadResult{}
	}
}

// onEachEngine runs test once on a new FreeCache client and once on a new miniredis client
func onEachEngine(t *testing.T, test func(t *testing.T, c ClientInterface)) {
	t.Helper()
	t.Run("freecache", func(t *testing.T) {
		test(t, newFreeCacheClient(t))
	})
	t.Run("miniredis", func(t *testing.T) {
		c, _ := newMiniredisClient(t.Context(), t)
		test(t, c)
	})
}

// assertReturnsWhenCanceled runs call with a context canceled 50ms in, and
// checks that call returns the context's error well within 200ms of the cancel
func assertReturnsWhenCanceled(t *testing.T, call func(ctx context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	timer := time.AfterFunc(50*time.Millisecond, cancel)
	defer timer.Stop()

	start := time.Now()
	err := call(ctx)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, 250*time.Millisecond)
}

// TestGetOrLoadReturnsAStoredValueWithoutLoading proves a hit decodes the stored value and never loads
func TestGetOrLoadReturnsAStoredValueWithoutLoading(t *testing.T) {
	onEachEngine(t, func(t *testing.T, c ClientInterface) {
		ctx := t.Context()
		stored := loadedValue{N: 7, Items: []string{"stored"}}
		require.NoError(t, SetModelValue(ctx, c, testKey, stored, time.Minute))

		var loads loadCounter
		got, err := GetOrLoad(ctx, c, testKey, time.Minute, loads.returning(loadedValue{N: 42}))
		require.NoError(t, err)
		assert.Equal(t, stored, got)
		assert.Zero(t, loads.calls.Load())
	})
}

// TestGetOrLoadCallsLoadOnceForConcurrentCallers proves the callers of one client share one load
func TestGetOrLoadCallsLoadOnceForConcurrentCallers(t *testing.T) {
	onEachEngine(t, func(t *testing.T, c ClientInterface) {
		const callers = 20
		ctx := t.Context()
		want := loadedValue{N: 42, Items: []string{"a", "b"}}
		release := make(chan struct{})
		var loads loadCounter
		load := loads.blockedUntil(release, want)

		results := make([]<-chan loadResult, 0, callers)
		for range callers {
			results = append(results, getOrLoadAsync(ctx, c, load, LoadOptions{}))
		}

		// Release the load once it runs and every caller has reached it
		loads.waitUntilLoading(t)
		letCallersJoin()
		close(release)

		values := make([]loadedValue, 0, callers)
		for _, result := range results {
			r := receive(t, result)
			require.NoError(t, r.err)
			values = append(values, r.value)
		}
		for _, value := range values {
			assert.Equal(t, want, value)
		}
		assert.Equal(t, int32(1), loads.calls.Load())

		// Each caller decoded its own copy
		assert.NotSame(t, &values[0].Items[0], &values[1].Items[0])

		stored, err := c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.JSONEq(t, `{"n":42,"items":["a","b"]}`, stored)
	})
}

// TestGetOrLoadSharesOneLoadAcrossClientsOnOneRedis proves the fill lock makes two clients share one load
func TestGetOrLoadSharesOneLoadAcrossClientsOnOneRedis(t *testing.T) {
	ctx := t.Context()
	first, s := newMiniredisClient(ctx, t)
	second, err := NewClient(ctx, WithRedis(&RedisConfig{
		MaxConnectionLifetime: DefaultRedisMaxIdleTimeout,
		MaxIdleTimeout:        DefaultRedisMaxIdleTimeout,
		URL:                   s.Addr(),
	}))
	require.NoError(t, err)

	want := loadedValue{N: 42, Items: []string{"shared"}}
	release := make(chan struct{})
	var loads loadCounter
	firstResult := getOrLoadAsync(ctx, first, loads.blockedUntil(release, want), LoadOptions{})
	loads.waitUntilLoading(t)

	// The second client finds the fill lock held, and polls for the value
	secondResult := getOrLoadAsync(ctx, second, loads.returning(loadedValue{N: 7}),
		LoadOptions{PollInterval: 10 * time.Millisecond})
	letCallersJoin()
	close(release)

	a := receive(t, firstResult)
	b := receive(t, secondResult)
	require.NoError(t, a.err)
	require.NoError(t, b.err)
	assert.Equal(t, want, a.value)
	assert.Equal(t, want, b.value)
	assert.Equal(t, int32(1), loads.calls.Load())
}

// TestGetOrLoadWaiterGivesUpAfterLockWait proves a held fill lock ends the wait with ErrLoadInProgress
func TestGetOrLoadWaiterGivesUpAfterLockWait(t *testing.T) {
	ctx := t.Context()
	c, _ := newMiniredisClient(ctx, t)
	_, err := c.WriteLock(ctx, testKey+":fill-lock", 30)
	require.NoError(t, err)

	var loads loadCounter
	start := time.Now()
	_, err = GetOrLoadWithOptions(ctx, c, testKey, time.Minute, loads.returning(loadedValue{N: 42}),
		LoadOptions{LockWait: 200 * time.Millisecond, PollInterval: 20 * time.Millisecond})
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrLoadInProgress)
	assert.GreaterOrEqual(t, elapsed, 200*time.Millisecond)
	assert.Less(t, elapsed, 2*time.Second)
	assert.Zero(t, loads.calls.Load())
}

// TestGetOrLoadWaiterReturnsItsContextError proves a waiting caller returns as soon as its context is done
func TestGetOrLoadWaiterReturnsItsContextError(t *testing.T) {
	t.Run("miniredis, the fill lock held elsewhere", func(t *testing.T) {
		c, _ := newMiniredisClient(t.Context(), t)
		_, err := c.WriteLock(t.Context(), testKey+":fill-lock", 30)
		require.NoError(t, err)

		var loads loadCounter
		assertReturnsWhenCanceled(t, func(ctx context.Context) error {
			_, loadErr := GetOrLoadWithOptions(ctx, c, testKey, time.Minute, loads.returning(loadedValue{N: 42}),
				LoadOptions{LockWait: time.Second})
			return loadErr
		})
		assert.Zero(t, loads.calls.Load())
	})

	t.Run("freecache, waiting on another caller's load", func(t *testing.T) {
		c := newFreeCacheClient(t)
		release := make(chan struct{})
		var loads loadCounter
		leader := getOrLoadAsync(t.Context(), c, loads.blockedUntil(release, loadedValue{N: 42}), LoadOptions{})
		loads.waitUntilLoading(t)

		assertReturnsWhenCanceled(t, func(ctx context.Context) error {
			_, loadErr := GetOrLoad(ctx, c, testKey, time.Minute, loads.returning(loadedValue{N: 7}))
			return loadErr
		})

		close(release)
		require.NoError(t, receive(t, leader).err)
		assert.Equal(t, int32(1), loads.calls.Load())
	})
}

// TestGetOrLoadNeverCachesAnError proves a failed load reaches its callers, stores nothing, and loads again next time
func TestGetOrLoadNeverCachesAnError(t *testing.T) {
	onEachEngine(t, func(t *testing.T, c ClientInterface) {
		const callers = 3
		ctx := t.Context()
		release := make(chan struct{})
		var loads loadCounter
		load := loads.failingAfter(release)

		results := make([]<-chan loadResult, 0, callers)
		for range callers {
			results = append(results, getOrLoadAsync(ctx, c, load, LoadOptions{}))
		}
		loads.waitUntilLoading(t)
		letCallersJoin()
		close(release)

		// Each caller gets the load's own error, as it was returned
		for _, result := range results {
			r := receive(t, result)
			require.ErrorIs(t, r.err, errLoadFailed)
			require.EqualError(t, r.err, errLoadFailed.Error())
		}
		assert.Equal(t, int32(1), loads.calls.Load())

		stored, err := c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Empty(t, stored)

		// The next call loads again
		want := loadedValue{N: 42}
		got, err := GetOrLoad(ctx, c, testKey, time.Minute, loads.returning(want))
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int32(2), loads.calls.Load())
	})
}

// TestGetOrLoadReplacesAValueThatNoLongerDecodes proves a stored value that doesn't decode is a miss
func TestGetOrLoadReplacesAValueThatNoLongerDecodes(t *testing.T) {
	onEachEngine(t, func(t *testing.T, c ClientInterface) {
		ctx := t.Context()
		require.NoError(t, c.SetTTL(ctx, testKey, "not json", time.Minute))

		want := loadedValue{N: 42, Items: []string{"new"}}
		var loads loadCounter
		got, err := GetOrLoad(ctx, c, testKey, time.Minute, loads.returning(want))
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int32(1), loads.calls.Load())

		stored, err := c.Get(ctx, testKey)
		require.NoError(t, err)
		assert.JSONEq(t, `{"n":42,"items":["new"]}`, stored)
	})
}

// TestGetOrLoadRefusesBadInputs proves a bad TTL, an empty key, or a done context fails before any load
func TestGetOrLoadRefusesBadInputs(t *testing.T) {
	onEachEngine(t, func(t *testing.T, c ClientInterface) {
		ctx := t.Context()
		var loads loadCounter
		load := loads.returning(loadedValue{N: 42})

		_, err := GetOrLoad(ctx, c, testKey, 0, load)
		require.ErrorIs(t, err, ErrInvalidTTL)

		_, err = GetOrLoad(ctx, c, testKey, -time.Second, load)
		require.ErrorIs(t, err, ErrInvalidTTL)

		_, err = GetOrLoad(ctx, c, "  ", time.Minute, load)
		require.ErrorIs(t, err, ErrKeyRequired)

		done, cancel := context.WithCancel(ctx)
		cancel()
		_, err = GetOrLoad(done, c, testKey, time.Minute, load)
		require.ErrorIs(t, err, context.Canceled)

		assert.Zero(t, loads.calls.Load())
	})
}

// TestGetOrLoadOutlivesACallerThatLeaves proves one caller leaving neither cancels the load nor fails the others
func TestGetOrLoadOutlivesACallerThatLeaves(t *testing.T) {
	c := newFreeCacheClient(t)
	want := loadedValue{N: 42}
	opts := LoadOptions{LockTTL: 2 * time.Second}

	// The load records its context and when it started, then blocks
	release := make(chan struct{})
	started := make(chan struct{})
	var loadCtx context.Context
	var loadStart time.Time
	var loads loadCounter
	load := func(ctx context.Context) (loadedValue, error) {
		loads.calls.Add(1)
		loadCtx, loadStart = ctx, time.Now()
		close(started)
		<-release
		return want, nil
	}

	firstCtx, cancelFirst := context.WithCancel(t.Context())
	defer cancelFirst()
	first := getOrLoadAsync(firstCtx, c, load, opts)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the load did not start")
	}
	second := getOrLoadAsync(t.Context(), c, load, opts)
	letCallersJoin()

	// The first caller leaves at once
	cancelFirst()
	require.ErrorIs(t, receive(t, first).err, context.Canceled)

	// The load runs on: no cancellation from the first caller, and a deadline of LockTTL
	require.NoError(t, loadCtx.Err())
	deadline, ok := loadCtx.Deadline()
	require.True(t, ok)
	assert.False(t, deadline.After(loadStart.Add(opts.LockTTL)))
	assert.WithinDuration(t, loadStart.Add(opts.LockTTL), deadline, time.Second)

	// The second caller gets the value
	close(release)
	r := receive(t, second)
	require.NoError(t, r.err)
	assert.Equal(t, want, r.value)
	assert.Equal(t, int32(1), loads.calls.Load())
}

// TestGetOrLoadRaisesALoadPanicInEachWaitingCaller proves a panic in load reaches every caller, with the load's stack
func TestGetOrLoadRaisesALoadPanicInEachWaitingCaller(t *testing.T) {
	const callers = 2
	ctx := t.Context()
	c := newFreeCacheClient(t)
	release := make(chan struct{})
	var loads loadCounter
	load := func(context.Context) (loadedValue, error) {
		loads.calls.Add(1)
		<-release
		panic("boom")
	}

	recovered := make(chan any, callers)
	for range callers {
		go func() {
			defer func() { recovered <- recover() }()
			_, _ = GetOrLoad(ctx, c, testKey, time.Minute, load)
		}()
	}
	loads.waitUntilLoading(t)
	letCallersJoin()
	close(release)

	for range callers {
		var r any
		select {
		case r = <-recovered:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "a caller did not return")
		}
		require.NotNil(t, r)
		text := fmt.Sprint(r)
		assert.Contains(t, text, "boom")
		assert.Contains(t, text, "goroutine")
	}
	assert.Equal(t, int32(1), loads.calls.Load())

	stored, err := c.Get(ctx, testKey)
	require.NoError(t, err)
	assert.Empty(t, stored)
}

// TestGetOrLoadStoresTheResultWithItsTTLAndReleasesTheFillLock proves the result follows the TTL rule and the fill lock is freed
func TestGetOrLoadStoresTheResultWithItsTTLAndReleasesTheFillLock(t *testing.T) {
	ctx := t.Context()
	c, s := newMiniredisClient(ctx, t)
	want := loadedValue{N: 42}

	var loads loadCounter
	got, err := GetOrLoad(ctx, c, testKey, 1500*time.Millisecond, loads.returning(want))
	require.NoError(t, err)
	assert.Equal(t, want, got)

	assert.Equal(t, 2*time.Second, s.TTL(testKey))
	assert.False(t, s.Exists(testKey+":fill-lock"))
}

// BenchmarkGetOrLoadHit measures a read of a stored value on each engine
func BenchmarkGetOrLoadHit(b *testing.B) {
	load := func(context.Context) (loadedValue, error) {
		return loadedValue{N: 42, Items: []string{"a", "b"}}, nil
	}

	b.Run("freecache", func(b *testing.B) {
		c, err := NewClient(b.Context(), WithFreeCacheConnection(freecache.NewCache(4*1024*1024)))
		require.NoError(b, err)
		benchmarkGetOrLoadHit(b, c, load)
	})

	// Idle connections are kept, as a service's pool keeps them, so each read doesn't dial
	b.Run("miniredis", func(b *testing.B) {
		s := miniredis.RunT(b)
		c, err := NewClient(b.Context(), WithRedis(&RedisConfig{
			MaxConnectionLifetime: DefaultRedisMaxIdleTimeout,
			MaxIdleConnections:    10,
			MaxIdleTimeout:        DefaultRedisMaxIdleTimeout,
			URL:                   s.Addr(),
		}))
		require.NoError(b, err)
		benchmarkGetOrLoadHit(b, c, load)
	})
}

// benchmarkGetOrLoadHit stores testKey through GetOrLoad, then times reading it back
func benchmarkGetOrLoadHit(b *testing.B, c ClientInterface, load func(context.Context) (loadedValue, error)) {
	b.Helper()
	ctx := b.Context()
	_, err := GetOrLoad(ctx, c, testKey, time.Hour, load)
	require.NoError(b, err)

	b.ReportAllocs()
	for b.Loop() {
		if _, err = GetOrLoad(ctx, c, testKey, time.Hour, load); err != nil {
			b.Fatal(err)
		}
	}
}
