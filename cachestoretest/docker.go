// Package cachestoretest provides helpers for running real Redis and Valkey
// servers in tests using the local Docker CLI.
//
// It deliberately shells out to the `docker` binary rather than depending on a
// container library, so importing it adds no third-party dependencies to your
// module. Consuming projects can use it to spin up a throwaway, ready-to-use
// cachestore client backed by a real engine:
//
//	func TestThing(t *testing.T) {
//	    client, _ := cachestoretest.StartValkey(t)
//	    _ = client.Set(context.Background(), "k", "v")
//	}
//
// The helpers skip the test when Docker is unavailable, so suites stay green on
// machines without Docker.
package cachestoretest

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	cachestore "github.com/mrz1836/go-cachestore"
)

// Redis and Valkey are wire-compatible, so the same client works against both.
const (
	// RedisImage is the default Redis engine image.
	RedisImage = "redis:7-alpine"
	// ValkeyImage is the default Valkey engine image.
	ValkeyImage = "valkey/valkey:8-alpine"
)

const (
	// readyTimeout bounds how long we wait for a container to accept connections.
	readyTimeout = 30 * time.Second

	// startTimeout bounds the `docker run` invocation (image pull included).
	startTimeout = 120 * time.Second
)

// StartRedis starts a throwaway Redis container and returns a ready cachestore
// client plus its connection URL. The container and client are removed via
// t.Cleanup. The test is skipped when Docker is unavailable.
func StartRedis(tb testing.TB) (cachestore.ClientInterface, string) {
	tb.Helper()
	return start(tb, RedisImage)
}

// StartValkey starts a throwaway Valkey container and returns a ready cachestore
// client plus its connection URL. It behaves exactly like StartRedis.
func StartValkey(tb testing.TB) (cachestore.ClientInterface, string) {
	tb.Helper()
	return start(tb, ValkeyImage)
}

// start launches the given engine image and wires up a cachestore client.
func start(tb testing.TB, image string) (cachestore.ClientInterface, string) {
	tb.Helper()
	skipIfNoDocker(tb)

	url := startContainer(tb, image)

	ctx := context.Background()
	client := newClientWithRetry(ctx, tb, url)
	tb.Cleanup(func() { client.Close(ctx) })
	return client, url
}

// skipIfNoDocker skips the test when the Docker CLI or daemon is unavailable.
func skipIfNoDocker(tb testing.TB) {
	tb.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		tb.Skip("docker CLI not found; skipping container-based test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}").CombinedOutput(); err != nil {
		tb.Skipf("docker daemon not available; skipping container-based test: %v: %s", err, out)
	}
}

// startContainer runs the image detached, publishing container port 6379 to a
// pre-selected free loopback host port, and returns the connection URL. Binding
// an explicit port (rather than querying `docker port` afterwards) avoids a
// race where the mapping is not yet reported right after `docker run`. The
// container is force-removed via t.Cleanup.
func startContainer(tb testing.TB, image string) string {
	tb.Helper()

	hostPort := freePort(tb)
	publish := fmt.Sprintf("127.0.0.1:%d:6379", hostPort)

	runCtx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()
	//nolint:gosec // G204: test helper intentionally runs the local docker CLI
	out, err := exec.CommandContext(runCtx, "docker", "run", "-d", "--rm",
		"-p", publish, image).CombinedOutput()
	if err != nil {
		tb.Fatalf("docker run failed: %v: %s", err, out)
	}

	id := strings.TrimSpace(string(out))
	tb.Cleanup(func() {
		rmCtx, rmCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer rmCancel()
		//nolint:gosec // G204: test helper intentionally runs the local docker CLI
		_ = exec.CommandContext(rmCtx, "docker", "rm", "-f", id).Run()
	})

	return fmt.Sprintf("redis://127.0.0.1:%d", hostPort)
}

// freePort asks the OS for an available loopback TCP port.
func freePort(tb testing.TB) int {
	tb.Helper()
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("could not find a free port: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

// newClientWithRetry builds a cachestore client, retrying until the server
// accepts connections (NewClient pings on connect) or the deadline passes.
func newClientWithRetry(ctx context.Context, tb testing.TB, url string) cachestore.ClientInterface {
	tb.Helper()

	deadline := time.Now().Add(readyTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		client, err := cachestore.NewClient(ctx, cachestore.WithRedis(&cachestore.RedisConfig{
			URL:                url,
			MaxIdleConnections: 10,
		}))
		if err == nil {
			return client
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	tb.Fatalf("cache server at %s did not become ready: %v", url, lastErr)
	return nil
}
