package cachestore

// Engine is the different engines that are supported for the cachestore
type Engine string

// Supported engines
const (
	Empty     Engine = "empty"     // No engine set
	FreeCache Engine = "freecache" // FreeCache (in-memory cache)
	Redis     Engine = "redis"     // Redis
)

// String is the string version of engine
func (e Engine) String() string {
	return string(e)
}

// IsEmpty will return true if the engine is not set
func (e Engine) IsEmpty() bool {
	return e == Empty
}
