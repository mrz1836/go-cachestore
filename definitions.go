package cachestore

import (
	"crypto/tls"
	"time"
)

const (
	// DefaultRedisMaxIdleTimeout is the default max timeout on an idle connection
	DefaultRedisMaxIdleTimeout = 240 * time.Second

	// DefaultRedisPort is the default Redis port
	DefaultRedisPort = "6379"

	// Empty time duration for comparison
	emptyTimeDuration = "0s"

	// lockRetrySleepTime is in milliseconds
	lockRetrySleepTime = 10 * time.Millisecond

	// RedisPrefix is the prefix for URL based connections
	RedisPrefix = "redis://"

	// RedissPrefix is the prefix for TLS URL based connections
	RedissPrefix = "rediss://"
)

// RedisConfig is the configuration for the cache client (redis or valkey)
//
// Redis and Valkey are wire-compatible, so the same configuration connects to
// either engine — point URL at your instance. For AWS ElastiCache with
// in-transit encryption use a rediss:// URL (or set UseTLS); for RBAC/ACL set
// Username and Password.
type RedisConfig struct {
	DependencyMode        bool          `json:"dependency_mode" mapstructure:"dependency_mode"`                 // false for digital ocean (not supported)
	MaxActiveConnections  int           `json:"max_active_connections" mapstructure:"max_active_connections"`   // 0
	MaxConnectionLifetime time.Duration `json:"max_connection_lifetime" mapstructure:"max_connection_lifetime"` // 0
	MaxIdleConnections    int           `json:"max_idle_connections" mapstructure:"max_idle_connections"`       // 10
	MaxIdleTimeout        time.Duration `json:"max_idle_timeout" mapstructure:"max_idle_timeout"`               // 240 * time.Second
	URL                   string        `json:"url" mapstructure:"url"`                                         // redis://localhost:6379

	// Authentication (ElastiCache RBAC / Redis ACL). When Username is set a
	// two-arg AUTH is issued; otherwise a single-arg AUTH is used. Supplying
	// credentials here keeps them out of the URL/logs.
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`

	// TLS configuration
	UseTLS bool `json:"use_tls" mapstructure:"use_tls"` // true for digital ocean (required); rediss:// also enables TLS

	// TLSConfig, when set, is used directly for the TLS handshake and takes
	// precedence over the TLS* fields below. Not serializable (json/mapstructure ignored).
	TLSConfig *tls.Config `json:"-" mapstructure:"-"`

	// TLSServerName overrides the SNI/ServerName used for certificate
	// validation. Leave empty to default to the endpoint host (correct for ElastiCache).
	TLSServerName string `json:"tls_server_name" mapstructure:"tls_server_name"`

	// TLSInsecureSkipVerify disables certificate verification (testing only).
	TLSInsecureSkipVerify bool `json:"tls_insecure_skip_verify" mapstructure:"tls_insecure_skip_verify"`

	// TLSCACertPath is an optional path to a PEM CA bundle used to verify the server.
	TLSCACertPath string `json:"tls_ca_cert_path" mapstructure:"tls_ca_cert_path"`
}
