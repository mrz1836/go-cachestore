package cachestore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildTLSConfig tests the method buildTLSConfig()
func TestBuildTLSConfig(t *testing.T) {
	t.Parallel()

	t.Run("no tls requested returns nil", func(t *testing.T) {
		t.Parallel()
		cfg, err := buildTLSConfig(&RedisConfig{URL: "redis://localhost:6379"})
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("use tls flag enables tls", func(t *testing.T) {
		t.Parallel()
		cfg, err := buildTLSConfig(&RedisConfig{URL: "redis://localhost:6379", UseTLS: true})
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	})

	t.Run("rediss scheme enables tls", func(t *testing.T) {
		t.Parallel()
		cfg, err := buildTLSConfig(&RedisConfig{URL: "rediss://localhost:6379"})
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("server name and skip verify", func(t *testing.T) {
		t.Parallel()
		cfg, err := buildTLSConfig(&RedisConfig{
			URL:                   "redis://host:6379",
			TLSServerName:         "example.com",
			TLSInsecureSkipVerify: true,
		})
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "example.com", cfg.ServerName)
		assert.True(t, cfg.InsecureSkipVerify)
	})

	t.Run("explicit tls config wins", func(t *testing.T) {
		t.Parallel()
		custom := &tls.Config{ServerName: "custom", MinVersion: tls.VersionTLS13}
		cfg, err := buildTLSConfig(&RedisConfig{URL: "redis://host:6379", TLSConfig: custom})
		require.NoError(t, err)
		assert.Same(t, custom, cfg)
	})

	t.Run("missing ca file errors", func(t *testing.T) {
		t.Parallel()
		cfg, err := buildTLSConfig(&RedisConfig{
			URL:           "rediss://host:6379",
			TLSCACertPath: filepath.Join(t.TempDir(), "does-not-exist.pem"),
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidRedisCACert)
		assert.Nil(t, cfg)
	})

	t.Run("invalid ca content errors", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "ca.pem")
		require.NoError(t, os.WriteFile(path, []byte("not a pem"), 0o600))

		cfg, err := buildTLSConfig(&RedisConfig{URL: "rediss://host:6379", TLSCACertPath: path})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidRedisCACert)
		assert.Nil(t, cfg)
	})

	t.Run("valid ca loads into root pool", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "ca.pem")
		require.NoError(t, os.WriteFile(path, generateTestCACertPEM(t), 0o600))

		cfg, err := buildTLSConfig(&RedisConfig{URL: "rediss://host:6379", TLSCACertPath: path})
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.NotNil(t, cfg.RootCAs)
	})
}

// generateTestCACertPEM returns a self-signed CA certificate in PEM format.
func generateTestCACertPEM(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "go-cachestore-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
