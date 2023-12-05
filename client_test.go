package cachestore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient will test the method NewClient()
func TestNewClient(t *testing.T) {
	t.Parallel()

	testCases := getInMemoryTestCases(t)
	for _, testCase := range testCases {
		t.Run(testCase.name+" - basic client", func(t *testing.T) {
			c, err := NewClient(context.Background(), testCase.opts)
			assert.NotNil(t, c)
			require.NoError(t, err)
			assert.Equal(t, testCase.engine, c.Engine())
		})

		t.Run(testCase.name+" - basic client, debugging", func(t *testing.T) {
			c, err := NewClient(context.Background(), WithDebugging(), testCase.opts)
			assert.NotNil(t, c)
			require.NoError(t, err)
			assert.Equal(t, testCase.engine, c.Engine())
			assert.True(t, c.IsDebug())
		})

		t.Run(testCase.name+" - basic client, new relic", func(t *testing.T) {
			ctx := getNewRelicCtx(t, testAppName, testTxn)

			c, err := NewClient(ctx, WithNewRelic(), testCase.opts)
			assert.NotNil(t, c)
			require.NoError(t, err)
			assert.Equal(t, testCase.engine, c.Engine())
			assert.True(t, c.IsNewRelicEnabled())

			c.Close(ctx)
			assert.Equal(t, Empty, c.Engine())
		})
	}

	t.Run("empty client, no options, defaults to FreeCache", func(t *testing.T) {
		c, err := NewClient(context.Background())
		assert.NotNil(t, c)
		require.NoError(t, err)
		assert.Equal(t, FreeCache, c.Engine())
	})

	t.Run("["+Redis.String()+"] - redis connection is nil", func(t *testing.T) {
		c, err := NewClient(context.Background(),
			WithRedisConnection(nil),
			WithDebugging(),
		)
		assert.NotNil(t, c)
		require.NoError(t, err)
		assert.Equal(t, FreeCache, c.Engine())
	})

	t.Run("["+Redis.String()+"] - redis config is nil", func(t *testing.T) {
		c, err := NewClient(context.Background(),
			WithRedis(nil),
			WithDebugging(),
		)
		assert.NotNil(t, c)
		require.NoError(t, err)
		assert.Equal(t, FreeCache, c.Engine())
	})

	t.Run("["+Redis.String()+"] - bad redis connection", func(t *testing.T) {
		c, err := NewClient(context.Background(),
			WithRedis(&RedisConfig{
				URL: RedisPrefix + "localbadhost:1919",
			}),
			WithDebugging(),
		)
		assert.Nil(t, c)
		require.Error(t, err)
	})

	t.Run("["+Redis.String()+"] - load mocked redis connection", func(t *testing.T) {
		redisClient, _ := loadMockRedis(
			testIdleTimeout, testMaxConnLifetime, testMaxActiveConnections, testMaxIdleConnections,
		)
		assert.NotNil(t, redisClient)

		c, err := NewClient(context.Background(),
			WithRedisConnection(redisClient),
			WithDebugging(),
		)
		assert.NotNil(t, c)
		require.NoError(t, err)
		assert.Equal(t, Redis, c.Engine())
	})

	t.Run("["+Redis.String()+"] - good redis connection", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test: redis is required")
		}

		c, err := NewClient(context.Background(),
			WithRedis(&RedisConfig{
				URL: testLocalConnectionURL,
			}),
			WithDebugging(),
		)
		assert.NotNil(t, c)
		require.NoError(t, err)
	})
}

// TestClient_Close will test the method Close()
func TestClient_Close(t *testing.T) {
	// t.Parallel()

	t.Run("["+FreeCache.String()+"] - load connection and close", func(t *testing.T) {
		c, err := NewClient(context.Background(),
			WithFreeCache(), WithDebugging(),
		)
		require.NotNil(t, c)
		require.NoError(t, err)

		c.Close(context.Background())

		assert.Equal(t, Empty, c.Engine())
		assert.Nil(t, c.FreeCache())
	})

	t.Run("["+Redis.String()+"] - load mocked connection and close", func(t *testing.T) {
		c, _ := newMockRedisClient(t)
		c.Close(context.Background())

		assert.Equal(t, Empty, c.Engine())
		assert.Nil(t, c.Redis())
	})

	t.Run("["+Redis.String()+"] - load connection and close", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test: redis is required")
		}

		redisClient, _, err := loadRealRedis(
			context.Background(), testLocalConnectionURL, testIdleTimeout, testMaxConnLifetime,
			testMaxActiveConnections, testMaxIdleConnections, true, false,
		)
		require.NotNil(t, redisClient)
		require.NoError(t, err)

		var c ClientInterface
		c, err = NewClient(context.Background(),
			WithRedisConnection(redisClient),
			WithDebugging(),
		)
		require.NotNil(t, c)
		require.NoError(t, err)

		c.Close(context.Background())

		assert.Equal(t, Empty, c.Engine())
		assert.Nil(t, c.Redis())
	})
}

// TestClient_Debug will test the method Debug()
func TestClient_Debug(t *testing.T) {
	t.Parallel()

	t.Run("["+FreeCache.String()+"] - turn debug on", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithFreeCache())
		require.NotNil(t, c)
		require.NoError(t, err)

		assert.False(t, c.IsDebug())

		c.Debug(true)

		assert.True(t, c.IsDebug())
	})

	t.Run("["+Redis.String()+"] - turn debug on", func(t *testing.T) {
		redisClient, _ := loadMockRedis(
			testIdleTimeout, testMaxConnLifetime, testMaxActiveConnections, testMaxIdleConnections,
		)
		require.NotNil(t, redisClient)

		c, err := NewClient(context.Background(),
			WithRedisConnection(redisClient),
		)
		require.NotNil(t, c)
		require.NoError(t, err)

		assert.False(t, c.IsDebug())

		c.Debug(true)

		assert.True(t, c.IsDebug())
	})

	t.Run("["+FreeCache.String()+"] - turn debug off", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithFreeCache(), WithDebugging())
		require.NotNil(t, c)
		require.NoError(t, err)

		assert.True(t, c.IsDebug())

		c.Debug(false)

		assert.False(t, c.IsDebug())
	})
}

// TestClient_IsDebug will test the method IsDebug()
func TestClient_IsDebug(t *testing.T) {
	t.Parallel()

	t.Run("["+FreeCache.String()+"] - check debug", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithFreeCache())
		require.NotNil(t, c)
		require.NoError(t, err)

		assert.False(t, c.IsDebug())

		c.Debug(true)

		assert.True(t, c.IsDebug())
	})

	t.Run("["+Redis.String()+"] - check debug", func(t *testing.T) {
		redisClient, _ := loadMockRedis(
			testIdleTimeout, testMaxConnLifetime, testMaxActiveConnections, testMaxIdleConnections,
		)
		require.NotNil(t, redisClient)

		c, err := NewClient(context.Background(),
			WithRedisConnection(redisClient),
		)
		require.NotNil(t, c)
		require.NoError(t, err)

		assert.False(t, c.IsDebug())

		c.Debug(true)

		assert.True(t, c.IsDebug())
	})
}

// TestClient_Engine will test the method Engine()
func TestClient_Engine(t *testing.T) {
	t.Parallel()

	t.Run("["+FreeCache.String()+"] - get engine", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithFreeCache())
		require.NotNil(t, c)
		require.NoError(t, err)
		assert.Equal(t, FreeCache, c.Engine())
	})

	t.Run("["+Redis.String()+"] - get engine", func(t *testing.T) {
		c, _ := newMockRedisClient(t)
		assert.Equal(t, Redis, c.Engine())
	})
}

// BenchmarkClient_Engine will benchmark the method Engine()
func BenchmarkClient_Engine(b *testing.B) {
	c, _ := NewClient(context.Background(), WithFreeCache())
	for i := 0; i < b.N; i++ {
		_ = c.Engine()
	}
}

// BenchmarkClient_Engine will benchmark the method Engine()
func BenchmarkClient_IsDebug(b *testing.B) {
	c, _ := NewClient(context.Background(), WithDebugging())
	for i := 0; i < b.N; i++ {
		_ = c.IsDebug()
	}
}
