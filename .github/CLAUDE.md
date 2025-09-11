# CLAUDE.md - go-cachestore

## Overview

**go-cachestore** is a Go caching abstraction layer supporting Redis and FreeCache (in-memory) engines. It provides a unified interface for key-value storage with TTL support, model serialization, and distributed locking.

## Core Architecture

### Engines
- **Redis**: Production-grade distributed caching with dependency keys
- **FreeCache**: Local in-memory caching (100MB default, 20% GC trigger)
- **Empty**: Uninitialized state (defaults to FreeCache with warning)

### Main Components
- `Client`: Main client with engine abstraction
- `ClientInterface`: Unified interface combining CacheService and LockService
- `RedisConfig`: Redis connection configuration
- Engine-specific implementations in `redis.go` and `freecache.go`

## Key Features

### Cache Operations
- `Set(ctx, key, value, deps...)` - Set key-value with optional dependencies
- `SetTTL(ctx, key, value, ttl, deps...)` - Set with TTL expiration
- `Get(ctx, key)` - Retrieve string value
- `Delete(ctx, key)` - Remove key
- `SetModel/GetModel` - JSON serialization for structs

### Locking System
- `WriteLock(ctx, key, ttl)` - Create exclusive lock with auto-generated secret
- `WriteLockWithSecret(ctx, key, secret, ttl)` - Lock with custom secret
- `WaitWriteLock(ctx, key, ttl, ttw)` - Aggressive lock retry within TTW
- `ReleaseLock(ctx, key, secret)` - Release lock if secret matches

### Client Configuration
```go
// Basic usage
client, err := cachestore.NewClient(ctx, cachestore.WithFreeCache())

// Redis with config
client, err := cachestore.NewClient(ctx, cachestore.WithRedis(&RedisConfig{
    URL: "redis://localhost:6379",
    MaxIdleConnections: 10,
    UseTLS: false,
}))

// With existing connections
client, err := cachestore.NewClient(ctx,
    cachestore.WithRedisConnection(existingRedisClient),
    cachestore.WithNewRelic(),
    cachestore.WithDebugging(),
)
```

## Development Commands

### Testing (via MAGE-X)
```bash
magex test              # Unit tests only
magex test:race         # Tests with race detector
magex bench             # Benchmarks
magex test:coverage     # Coverage analysis
magex fuzz:run          # Fuzz testing
```

### Build & Quality
```bash
magex clean             # Clean build artifacts
magex lint              # Code linting
magex deps:update       # Update all dependencies
magex audit:report      # Security audit
```

### CI/CD
- Uses GitHub Actions with fortress.yml workflow
- CodeQL security analysis
- Dependabot auto-merge
- Multi-version Go testing (Go 1.24+)

## File Structure

### Core Files
- `client.go` - Main client implementation
- `cachestore.go` - Cache operations (Set, Get, Delete, Model ops)
- `lock.go` - Locking operations
- `interface.go` - Service interfaces
- `engine.go` - Engine type definitions
- `redis.go` / `freecache.go` - Engine-specific implementations

### Configuration
- `client_options.go` - Functional options pattern
- `definitions.go` - Constants and Redis config struct
- `errors.go` - Custom error types

### Testing
- Comprehensive test coverage with mock Redis (miniredis)
- Fuzz testing for all major components
- Race condition testing
- Engine-specific test suites

## Critical Notes

### Redis Features
- Dependency keys for cache invalidation
- TLS support for cloud Redis (DigitalOcean)
- Connection pooling and lifecycle management
- NewRelic integration for monitoring

### FreeCache Features
- 100MB default cache size
- Automatic garbage collection
- Thread-safe operations
- No dependency key support

### Error Handling
- `ErrKeyRequired` - Empty key validation
- `ErrKeyNotFound` - Cache miss
- `ErrLockCreateFailed` - Lock acquisition failure
- `ErrSecretRequired` - Lock secret validation

### Dependencies
- `github.com/coocood/freecache` - In-memory cache
- `github.com/mrz1836/go-cache` - Redis abstraction
- `github.com/gomodule/redigo` - Redis driver
- `github.com/newrelic/go-agent/v3` - APM integration

## Usage Examples

### Basic Caching
```go
client, _ := cachestore.NewClient(ctx, cachestore.WithFreeCache())
defer client.Close(ctx)

// String operations
client.Set(ctx, "key", "value")
value, _ := client.Get(ctx, "key")

// Model operations
client.SetModel(ctx, "user:123", &user, time.Hour)
client.GetModel(ctx, "user:123", &user)
```

### Distributed Locking
```go
secret, err := client.WriteLock(ctx, "resource", 60) // 60 second TTL
if err != nil {
    // Handle lock failure
}
defer client.ReleaseLock(ctx, "resource", secret)

// Critical section
performExclusiveOperation()
```

## When to Use

- **Redis**: Multi-instance deployments, persistence needs, dependency invalidation
- **FreeCache**: Single-instance apps, development, high-performance local caching
- **Locking**: Resource coordination, prevent duplicate processing, critical sections

The abstraction allows switching engines without code changes, making it ideal for development→production transitions.
