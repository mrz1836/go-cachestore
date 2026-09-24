package cachestore

import (
	"context"
	"errors"
	"time"
)

// GetModelValue retrieves the value stored at key and unmarshals its JSON into a
// value of type T. It is the generic, value-returning counterpart to
// Client.GetModel: rather than taking a caller-allocated pointer, it returns the
// decoded value directly, so the caller writes
//
//	user, found, err := GetModelValue[User](ctx, c, key)
//
// instead of allocating a User and passing its address.
//
// found reports whether the key existed. On a miss it returns the zero value of T,
// false, and a nil error, so a missing key is distinguished from a decode or
// transport error without inspecting a sentinel. Any other error is returned as-is
// with found false and the zero value of T.
func GetModelValue[T any](ctx context.Context, c CacheService, key string) (T, bool, error) {
	var value T
	switch err := c.GetModel(ctx, key, &value); {
	case err == nil:
		return value, true, nil
	case errors.Is(err, ErrKeyNotFound):
		var zero T
		return zero, false, nil
	default:
		var zero T
		return zero, false, err
	}
}

// SetModelValue marshals value to JSON and stores it at key with the given TTL. It
// is the generic counterpart to Client.SetModel and the write side of
// GetModelValue.
//
// Redis only supports dependency keys at this time.
func SetModelValue[T any](ctx context.Context, c CacheService, key string, value T, ttl time.Duration, dependencies ...string) error {
	return c.SetModel(ctx, key, &value, ttl, dependencies...)
}
