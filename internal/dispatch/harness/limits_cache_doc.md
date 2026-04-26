# LimitsCache Documentation

## Overview

`LimitsCache` manages rate-limit information for AI provider APIs. It combines two data sources:

1. **Background Poller** — Periodically calls `Provider.CheckLimits()` to fetch fresh limits
2. **Response Headers** — Updates from HTTP response headers after each harness invocation

The cache ensures the hot-path (`Router` selection) never blocks on HTTP calls, while maintaining fresh rate-limit state through both polling and header-driven updates.

## Configuration

### TTL (Time-To-Live)

- **Default:** 30 seconds
- **Poller Interval:** TTL / 2 (15 seconds by default)

The poller runs at half the TTL to ensure limits are refreshed before the cache stale time. Header-derived limits remain cached until their TTL expires, at which point the poller takes over.

## Priority & Semantics

**Priority Order (highest to lowest):**

1. **Header-derived limits** (from `UpdateFromHeaders`) — Fresh data from response headers
2. **Poller-derived limits** — Fresh data from `Provider.CheckLimits()`
3. **Nil** — Provider not yet polled

When `UpdateFromHeaders` is called, it immediately overrides the poller cache and sets an expiry. The poller will not update the cache until this expiry passes. This ensures that more recent header data is trusted over the periodic poller.

## Supported Header Families

### Standard x-ratelimit Headers

```
x-ratelimit-limit-requests: 1000
x-ratelimit-remaining-requests: 950
```

Used by OpenAI and others. Maps to:
- `Total` = limit value
- `Used` = limit - remaining

### Anthropic-style Headers

```
anthropic-ratelimit-requests-limit: 100000
anthropic-ratelimit-requests-remaining: 99900
```

Maps identically to x-ratelimit headers.

### Absent or Unknown Headers

If a provider doesn't expose rate-limit headers, or headers are unrecognized, `UpdateFromHeaders` is a no-op. The poller continues to provide cached data. No error is raised.

## API

### `NewLimitsCache(ttl time.Duration) *LimitsCache`

Creates a new cache with the given TTL. If `ttl` is 0, defaults to 30 seconds.

### `Start(ctx context.Context, providers []Provider)`

Launches background pollers for each provider. Each poller:
- Runs immediately (initial check)
- Repeats every TTL/2
- Respects context cancellation
- Stops on `Stop()` call

All pollers are independent goroutines; they do not block the caller.

### `Stop()`

Terminates all pollers cleanly. Safe to call multiple times. After `Stop()`, the cache is no longer updated, but `Get()` still returns the last known values.

### `Get(provider string) *Limits`

Returns the most recent cached `Limits` for the provider, or `nil` if not found. Never blocks — reads are lock-free via `sync.Map`. Thread-safe for concurrent readers.

### `UpdateFromHeaders(provider string, hdrs http.Header)`

Parses `http.Header` for rate-limit fields and updates the cache immediately. The update includes:
- Extraction of `remaining` and `limit` from standard headers
- Calculation of `Used` = `Total - remaining`
- Setting `Source` = `"headers/" + provider`
- Setting `CheckedAt` = `time.Now().UTC()`
- Setting expiry to `now + ttl`

Header parsing is case-insensitive (delegated to `http.Header.Get()`). If no recognized headers are found, the call is a no-op and no error is raised.

## Thread Safety

- `Get()` is lock-free via `sync.Map`; any number of concurrent readers
- `UpdateFromHeaders()` is thread-safe; multiple writers are serialized by `sync.Map`
- Pollers run in independent goroutines with no shared mutable state beyond the cache
- Race detector passes when run with `-race` flag

## Hot-Path Performance

- `Get()` p99 latency: < 100 µs (typical < 10 µs)
- No locks, no allocations beyond cache lookups
- Suitable for per-request rate-limit checks in `Router.Route`

## Example Usage

```go
// Create and start
cache := harness.NewLimitsCache(30 * time.Second)
defer cache.Stop()

providers := []harness.Provider{
    openai.NewProvider(),
    anthropic.NewProvider(),
}
cache.Start(ctx, providers)

// Hot-path: get limits without blocking
limits := cache.Get("openai")
if limits != nil && limits.UsagePercent() > 0.8 {
    // Switch to less-used provider
}

// After each invocation: update from response headers
cache.UpdateFromHeaders("openai", resp.Header)
```

## Edge Cases

1. **Provider not registered:** `Get()` returns `nil` (no panic)
2. **No headers or unrecognized headers:** `UpdateFromHeaders()` is a no-op
3. **Very short TTL (< 1s):** Poller interval becomes 500ms; fine but high CPU cost
4. **Context cancelled during Start:** Pollers exit gracefully without goroutine leak
5. **Stop called before Start:** Safe; subsequent Start call works normally

## Integration Notes

- `LimitsCache` lives in package `harness` (not `providers`) so it's importable by both Router and harness impls
- Each `CascadingInvoker` instance should have its own `LimitsCache` (or share one if managing multiple invokers)
- Header parsing is **per-provider** — each provider may expose different header names; parsing is generic and delegates to standard names
- If a provider's limits are unknown after TTL expiry, the cache returns `nil`; callers should handle gracefully
