<div align="center">

# 📑&nbsp;&nbsp;go-cachestore

**Simple cache access & storage layer using [Redis](https://redis.io/) or [FreeCache](https://github.com/coocood/freecache).**

<br/>

<a href="https://github.com/mrz1836/go-cachestore/releases"><img src="https://img.shields.io/github/release-pre/mrz1836/go-cachestore?include_prereleases&style=flat-square&logo=github&color=black" alt="Release"></a>
<a href="https://golang.org/"><img src="https://img.shields.io/github/go-mod/go-version/mrz1836/go-cachestore?style=flat-square&logo=go&color=00ADD8" alt="Go Version"></a>
<a href="https://github.com/mrz1836/go-cachestore/blob/master/LICENSE"><img src="https://img.shields.io/github/license/mrz1836/go-cachestore?style=flat-square&color=blue" alt="License"></a>

<br/>

<table align="center" border="0">
  <tr>
    <td align="right">
       <code>CI / CD</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/mrz1836/go-cachestore/actions"><img src="https://img.shields.io/github/actions/workflow/status/mrz1836/go-cachestore/fortress.yml?branch=master&label=build&logo=github&style=flat-square" alt="Build"></a>
       <a href="https://github.com/mrz1836/go-cachestore/actions"><img src="https://img.shields.io/github/last-commit/mrz1836/go-cachestore?style=flat-square&logo=git&logoColor=white&label=last%20update" alt="Last Commit"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Quality</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://codecov.io/gh/mrz1836/go-cachestore"><img src="https://codecov.io/gh/mrz1836/go-cachestore/branch/master/graph/badge.svg?style=flat-square" alt="Coverage"></a>
    </td>
  </tr>

  <tr>
    <td align="right">
       <code>Security</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://scorecard.dev/viewer/?uri=github.com/mrz1836/go-cachestore"><img src="https://api.scorecard.dev/projects/github.com/mrz1836/go-cachestore/badge?style=flat-square" alt="Scorecard"></a>
       <a href=".github/SECURITY.md"><img src="https://img.shields.io/badge/policy-active-success?style=flat-square&logo=security&logoColor=white" alt="Security"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Community</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/mrz1836/go-cachestore/graphs/contributors"><img src="https://img.shields.io/github/contributors/mrz1836/go-cachestore?style=flat-square&color=orange" alt="Contributors"></a>
       <a href="https://mrz1818.com/"><img src="https://img.shields.io/badge/donate-bitcoin-ff9900?style=flat-square&logo=bitcoin" alt="Bitcoin"></a>
    </td>
  </tr>
</table>

</div>

<br/>
<br/>

<div align="center">

### <code>Project Navigation</code>

</div>

<table align="center">
  <tr>
    <td align="center" width="33%">
       🚀&nbsp;<a href="#installation"><code>Installation</code></a>
    </td>
    <td align="center" width="33%">
       🧪&nbsp;<a href="#examples--tests"><code>Examples&nbsp;&&nbsp;Tests</code></a>
    </td>
    <td align="center" width="33%">
       📚&nbsp;<a href="#documentation"><code>Documentation</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       🤝&nbsp;<a href="#contributing"><code>Contributing</code></a>
    </td>
    <td align="center">
      🛠️&nbsp;<a href="#code-standards"><code>Code&nbsp;Standards</code></a>
    </td>
    <td align="center">
      ⚡&nbsp;<a href="#benchmarks"><code>Benchmarks</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
      🤖&nbsp;<a href="#-ai-usage--assistant-guidelines"><code>AI&nbsp;Usage</code></a>
    </td>
    <td align="center">
       ⚖️&nbsp;<a href="#license"><code>License</code></a>
    </td>
    <td align="center">
       👥&nbsp;<a href="#maintainers"><code>Maintainers</code></a>
    </td>
  </tr>
</table>

<br/>

## Installation

**go-cachestore** requires a [supported release of Go](https://golang.org/doc/devel/release.html#policy).
```shell script
go get github.com/mrz1836/go-cachestore
```

<br/>

## Documentation
View the generated [documentation](https://pkg.go.dev/github.com/mrz1836/go-cachestore)

<details>
<summary><strong><code>Development Setup (Getting Started)</code></strong></summary>
<br/>

Install [MAGE-X](https://github.com/mrz1836/mage-x) build tool for development:

```bash
# Install MAGE-X for development and building
go install github.com/mrz1836/mage-x/cmd/magex@latest
magex update:install
```
</details>

<details>
<summary><strong><code>Library Deployment</code></strong></summary>
<br/>

This project uses [goreleaser](https://github.com/goreleaser/goreleaser) for streamlined binary and library deployment to GitHub. To get started, install it via:

```bash
brew install goreleaser
```

The release process is defined in the [.goreleaser.yml](.goreleaser.yml) configuration file.

Then create and push a new Git tag using:

```bash
magex version:bump bump=patch push=true branch=master
```

This process ensures consistent, repeatable releases with properly versioned artifacts and citation metadata.

</details>

<details>
<summary><strong><code>Build Commands</code></strong></summary>
<br/>

View all build commands

```bash script
magex help
```

</details>

<details>
<summary><strong><code>GitHub Workflows</code></strong></summary>
<br/>

All workflows are driven by modular configuration in [`.github/env/`](.github/env/README.md) — no YAML editing required.

**[View all workflows and the control center →](.github/docs/workflows.md)**

</details>

<details>
<summary><strong><code>Updating Dependencies</code></strong></summary>
<br/>

To update all dependencies (Go modules, linters, and related tools), run:

```bash
magex deps:update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any managed tools. It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<br/>

## Redis & Valkey

The Redis engine works with both [Redis](https://redis.io) and [Valkey](https://valkey.io) — they are wire-compatible, so the same `RedisConfig` connects to either engine; just point `URL` at your instance. Supported URL schemes: `redis://`, `rediss://`, `valkey://`, `valkeys://` (the `s` variants enable TLS).

```go
// Local (Redis or Valkey)
client, err := cachestore.NewClient(ctx, cachestore.WithRedis(&cachestore.RedisConfig{
    URL:                "redis://localhost:6379", // or valkey://localhost:6379
    MaxIdleConnections: 10,
}))
```

### AWS ElastiCache (TLS + RBAC / ACL)

For ElastiCache (Redis **or** Valkey) with in-transit encryption and RBAC, use a `rediss://` URL (or set `UseTLS: true`) and supply the ACL `Username`/`Password`. Keeping credentials in the config (rather than the URL) keeps them out of logs. Leave `TLSServerName` empty and it defaults to the endpoint host — the correct SNI for ElastiCache.

```go
client, err := cachestore.NewClient(ctx, cachestore.WithRedis(&cachestore.RedisConfig{
    URL:                "rediss://my-cluster.abc123.use1.cache.amazonaws.com:6379",
    MaxIdleConnections: 10,
    Username:           "app-user", // ElastiCache RBAC / Redis ACL user
    Password:           os.Getenv("REDIS_PASSWORD"),
    // Optional TLS controls (all optional; rediss:// already enables TLS):
    //   TLSConfig             *tls.Config // full custom config (wins over the fields below)
    //   TLSServerName         string      // SNI override
    //   TLSCACertPath         string      // custom CA bundle (PEM)
    //   TLSInsecureSkipVerify bool         // testing only
}))
```

> Single-endpoint (cluster-mode-disabled) deployments are supported. Sharded cluster mode is not yet implemented; the config is designed so it can be added later without breaking the API.

### Local testing with real Redis/Valkey

The [`cachestoretest`](./cachestoretest) subpackage can spin up throwaway **real** Redis or Valkey servers for your tests using the local Docker CLI. It adds **no dependencies** to your module and skips automatically when Docker is unavailable:

```go
import "github.com/mrz1836/go-cachestore/cachestoretest"

func TestWithValkey(t *testing.T) {
    client, _ := cachestoretest.StartValkey(t) // or StartRedis(t)
    require.NoError(t, client.Set(context.Background(), "key", "value"))
}
```

<br/>

## Values, TTLs, locks, and `GetOrLoad`

### TTLs

`SetTTL`, `SetModel`, and `SetModelValue` follow one rule on both engines:

| TTL                       | Stored as                                                                                             |
|---------------------------|-------------------------------------------------------------------------------------------------------|
| A whole number of seconds | That many seconds                                                                                     |
| A fraction of a second    | Rounded up to whole seconds: `500*time.Millisecond` lives one second, and `1500*time.Millisecond` two |
| `0`                       | No expiry, as `Set` stores                                                                            |
| Negative                  | Refused with `ErrInvalidTTL`; nothing is written                                                      |

Rounding up means a fraction never shortens a TTL or turns it into no expiry. FreeCache counts time in whole seconds, so there an entry expires within the last second of its TTL (a two-second entry lives between one and two seconds). Redis and Valkey keep milliseconds.

### Locks

```go
secret, err := client.WriteLock(ctx, "reports:daily", 30) // TTL in seconds, at least 1
switch {
case errors.Is(err, cachestore.ErrLockExists):
    // another secret holds the lock: try again later
case err != nil:
    // the lock couldn't be taken (err wraps the cause and ErrLockCreateFailed)
default:
    defer func() { _, _ = client.ReleaseLock(ctx, "reports:daily", secret) }()
    // ... the work the lock protects
}
```

- **TTLs.** A lock's TTL is in seconds and must be at least one; zero or less returns `ErrInvalidTTL`.
- **Errors.** Every failure to take a lock wraps `ErrLockCreateFailed` and its cause, and reads `<cause>: failed creating cache lock`. When another secret holds the lock, the error also wraps `ErrLockExists`, so `errors.Is` tells a held lock from a store that failed.
- **Atomic on both engines.** Taking a lock and releasing it are each one atomic step on FreeCache too. FreeCache has no compare-and-delete, so a released FreeCache lock leaves an empty value for up to a second; `Get` reads it as missing, and the next lock takes it.
- **Waiting.** `WaitWriteLock(ctx, key, ttl, ttw)` tries again every 10 ms while another secret holds the lock, for up to `ttw` seconds. It returns `ctx`'s error as soon as `ctx` is done, any other failure at once, and, when `ttw` runs out, the last attempt's error (which wraps `ErrLockExists`). A nil `ctx` waits with no cancellation.

### GetOrLoad

`GetOrLoad` reads a key and decodes its JSON into a `T`. On a miss it calls `load` once, stores the result with the TTL (under the rule above), and returns it:

```go
ctx := context.Background()
client, err := cachestore.NewClient(ctx, cachestore.WithFreeCache())
if err != nil {
    fmt.Println(err)
    return
}
defer client.Close(ctx)

// load stands in for an expensive query
loads := 0
load := func(context.Context) (int, error) {
    loads++
    return 42, nil
}

var answer int
for range 3 {
    if answer, err = cachestore.GetOrLoad(ctx, client, "answer", time.Minute, load); err != nil {
        fmt.Println(err)
        return
    }
    fmt.Println(answer) // 42, three times
}
fmt.Println("loads:", loads) // loads: 1
```

- **One load per key, per client.** Callers on the same client and key share one call to `load`; the others wait for its result. Each caller decodes its own copy, so `T` must round-trip through `encoding/json`.
- **Across processes, on Redis and Valkey only.** The call runs only while it holds the key's fill lock, `<key>:fill-lock`. A caller in another process that finds the lock held re-reads the key every `PollInterval`, takes the lock itself once it is free and the key is still empty, and returns `ErrLoadInProgress` when `LockWait` passes without a value. FreeCache is one process and has no fill lock, so there the shared call is the whole guarantee.
- **Errors are never cached.** An error from `load` reaches every caller of that call, as `load` returned it, and nothing is stored, so the next call loads again. A stored value that no longer decodes counts as a miss, and the load replaces it.
- **Contexts.** `load` runs with the first caller's context values but not its cancellation, under a deadline of `LockTTL`, so one caller leaving fails none of the others. Each caller returns its own context's error as soon as its context is done. A panic in `load` is raised again in each waiting caller.

`GetOrLoadWithOptions` takes `LoadOptions`, whose zero or negative fields take their defaults:

| Option         | Default | What it sets                                                                    |
|----------------|---------|---------------------------------------------------------------------------------|
| `LockTTL`      | 10 s    | The fill lock's TTL, rounded up to whole seconds, and the longest one load runs |
| `LockWait`     | 5 s     | How long a caller waits for another process's load                              |
| `PollInterval` | 50 ms   | How often a waiting caller re-reads the key                                     |

```go
report, err := cachestore.GetOrLoadWithOptions(ctx, client, "report:daily", time.Minute, buildReport,
    cachestore.LoadOptions{LockWait: time.Second})
if errors.Is(err, cachestore.ErrLoadInProgress) {
    // another process is still building the report: ask the caller to retry shortly
}
```

<br/>

## Examples & Tests
All unit tests run via [GitHub Actions](https://github.com/mrz1836/go-template/actions) and use [Go version 1.26.x](https://go.dev/doc/go1.26). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
magex test
```

Run all tests with race detector (slower):
```bash script
magex test:race
```

<br/>

## Benchmarks
Run the Go benchmarks:
```shell script
magex bench
```

<br/>

## Code Standards
Read more about this Go project's [code standards](.github/CODE_STANDARDS.md).

<br>

## 🤖 AI Usage & Assistant Guidelines
Read the [AI Usage & Assistant Guidelines](.github/tech-conventions/ai-compliance.md) for details on how AI is used in this project and how to interact with AI assistants.

<br/>

## Maintainers
| [<img src="https://github.com/mrz1836.png" height="50" alt="MrZ" />](https://github.com/mrz1836) |
|:------------------------------------------------------------------------------------------------:|
|                                [MrZ](https://github.com/mrz1836)                                 |

<br/>

## Contributing
View the [contributing guidelines](.github/CONTRIBUTING.md) and please follow the [code of conduct](.github/CODE_OF_CONDUCT.md).

### How can I help?
All kinds of contributions are welcome :raised_hands:!
The most basic way to show your support is to star :star2: the project, or to raise issues :speech_balloon:.
You can also support this project by [becoming a sponsor on GitHub](https://github.com/sponsors/mrz1836) :clap:
or by making a [**bitcoin donation**](https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-cachestore&utm_term=go-cachestore&utm_content=go-cachestore) to ensure this journey continues indefinitely! :rocket:


[![Stars](https://img.shields.io/github/stars/mrz1836/go-cachestore?label=Please%20like%20us&style=social)](https://github.com/mrz1836/go-cachestore/stargazers)

<br/>

## License

[![License](https://img.shields.io/github/license/mrz1836/go-cachestore.svg?style=flat)](LICENSE)
