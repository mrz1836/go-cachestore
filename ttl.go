package cachestore

import (
	"math"
	"time"
)

// maxTTLSeconds is the longest TTL, in whole seconds, that converts back to a time.Duration
const maxTTLSeconds = int64(math.MaxInt64 / time.Second)

// ttlSeconds is ttl in whole seconds, rounded up, so a fraction never shortens
// an entry's life or turns into no expiry: 500ms is 1s, and 1.5s is 2s
//
// Zero is no expiry. A negative ttl is ErrInvalidTTL. Only the last fraction of
// a second below the longest time.Duration (about 292 years) rounds down, so the
// result still converts back to a time.Duration
func ttlSeconds(ttl time.Duration) (int64, error) {
	if ttl < 0 {
		return 0, ErrInvalidTTL
	}

	// Divide, then add one for a remainder (ttl + time.Second - 1 would overflow)
	secs := int64(ttl / time.Second)
	if ttl%time.Second != 0 && secs < maxTTLSeconds {
		secs++
	}
	return secs, nil
}
