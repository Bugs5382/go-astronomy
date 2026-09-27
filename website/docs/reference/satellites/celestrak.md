---
id: celestrak
title: celestrak
sidebar_position: 4
---

# 🌐 Package `celestrak`

Import path: `github.com/Bugs5382/go-astronomy/satellite/celestrak`

An explicit element-set fetcher for CelesTrak's GP API (`https://celestrak.org/NORAD/elements/gp.php`, OMM JSON). It implements `satellite.ElementSource`, so it plugs into any tracker: `iss.New(celestrak.New())`. It follows the same pattern as the elevation resolver: it takes `ctx`, has a default timeout, takes an injectable `*http.Client`, and uses a pluggable cache.

## 📦 API

```go
const (
	DefaultBaseURL    = "https://celestrak.org/NORAD/elements/gp.php"
	DefaultRefreshAge = 3 * 24 * time.Hour
	MinRefetch        = 2 * time.Hour
	DefaultTimeout    = 15 * time.Second
)

var ErrFetch error

type Result struct {
	Elements  satellite.Elements
	FetchedAt time.Time     // when the set was fetched from CelesTrak
	Epoch     time.Time     // the set's epoch
	Age       time.Duration // how old the set was when this call answered
}

func New(opts ...Option) *Client
func (c *Client) Fetch(ctx context.Context, catalog int) (Result, error)
func (c *Client) Elements(ctx context.Context, catalog int) (satellite.Elements, error)

func WithHTTPClient(c *http.Client) Option
func WithBaseURL(u string) Option
func WithTimeout(d time.Duration) Option
func WithRefreshAge(d time.Duration) Option // never below MinRefetch
func NeverExpire() Option                   // fetch once, never refresh
func WithCache(c satellite.Cache) Option    // default: satellite.NewMemoryCache()
func WithClock(now func() time.Time) Option
func WithLogger(l log.Logger) Option
```

## 🗄️ What is cached

**The element set, keyed by catalogue number only** (`celestrak:gp:25544`), never anything per observer, time, or position. Every position and every pass, for any observer at any time or date, is propagated locally with SGP4 from that one cached set, with no fetch. A service answering thousands of visitors in different places makes one request per satellite, and then one quiet refresh every few days.

## 🔁 Two modes

### Default: never block, refresh quietly

- **First call.** The first call for a catalogue number fetches synchronously. Callers that arrive together share that one request. A failed first fetch is an error wrapping `ErrFetch`; further calls get the same error without a new request until `MinRefetch` has passed.
- **Every later call answers straight from the cache.** Elements never expire for callers.
- **Background refresh.** Once the cached set is older than the refresh age (`DefaultRefreshAge`, 3 days; `WithRefreshAge` to change it), the call that notices starts one refresh in the background and still answers at once. The refresh is de-duplicated, so there is one in flight per catalogue number. It is never sooner than `MinRefetch` (2 hours) after the last request, following CelesTrak's usage guidance: sets change a few times a day, so fetch each rarely and reuse it.
- **A failed refresh** keeps the old set, is logged at warn, and is retried after `MinRefetch`.
- **Age.** `Result.Epoch` and `Result.Age`, and `ElementEpoch` on every `satellite.Look` and `satellite.Pass`, show how old the data behind an answer is.

### `NeverExpire()`: fetch once

The set is fetched once and never refreshed. That suits a batch job or a demo. It does not suit a long-running service, because propagation from an old set drifts (see below). When a set is so old that SGP4 cannot propagate it (its drag model takes the orbit out of range, or the satellite decays), the tracker returns that propagation error with no result, never a wrong position.

## ⏳ How element sets drift

An element set is a fit to tracking data at its epoch. It is propagated forward by a model that only approximates the atmosphere, so its error grows with age. Rough figures for a low orbit such as the ISS:

| age of the set | position error | what it means |
| --- | --- | --- |
| hours to a day | about a kilometre | as good as it gets |
| a few days | a few kilometres | under a second of pass timing (the ISS moves 7.7 km/s); fine |
| about a week | tens to hundreds of kilometres | pass times off by seconds to tens of seconds |
| a month or more | can be far off, and SGP4 may refuse to propagate | not usable |
| after an ISS reboost | tens of kilometres at once, whatever the age | refresh after a burn |

Higher and steadier orbits, such as Hubble, degrade more slowly. The default refresh every 3 days keeps a low orbit within a few kilometres.

## 💾 Restarts and replicas

The default `MemoryCache` lives in the process, so a restart loses it and the next call fetches again. To keep sets across restarts and share them between replicas, plug in a `satellite.Cache` backed by your own store. With Redis, the adapter is a few lines in your code; go-astronomy has no Redis dependency:

```go
// RedisCache adapts a go-redis client to satellite.Cache. It lives in your
// code; go-astronomy has no Redis dependency.
type RedisCache struct{ R *redis.Client }

func (c RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := c.R.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	return b, err == nil, err
}

func (c RedisCache) Set(ctx context.Context, key string, v []byte, expiry time.Duration) error {
	return c.R.Set(ctx, key, v, expiry).Err() // an expiry of 0 keeps it
}

elements := celestrak.New(celestrak.WithCache(RedisCache{R: rdb}))
tracker := iss.New(elements)
```


Sets are stored with no expiry; each refresh replaces the old one. The fetcher logs through go-log: catalogue numbers, cache hits, requests, refreshes, and failures. It logs nothing else about the caller.

## 🚀 Examples

Each example is an `Example` test in `satellite/celestrak/example_test.go`; where an output is shown, it is what the test prints.

### Client_Fetch

A client pointed at a stand-in for CelesTrak serving the recorded ISS set: the first call fetches, the second is served from the cache.

```go
requests := 0
api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	requests++
	b, _ := os.ReadFile("testdata/gp-" + r.URL.Query().Get("CATNR") + ".json")
	_, _ = w.Write(b)
}))
defer api.Close()

c := celestrak.New(celestrak.WithBaseURL(api.URL))
for range 2 {
	r, err := c.Fetch(context.Background(), 25544)
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Elements.SatNum, r.Epoch.Format(time.RFC3339), "fetched", !r.FetchedAt.IsZero())
}
fmt.Println("requests:", requests)
```

```text
25544 2026-09-26T20:26:13Z fetched true
25544 2026-09-26T20:26:13Z fetched true
requests: 1
```

### WithCache

Sharing the cache between replicas: any satellite.Cache works, for example one backed by Redis. Here two clients share one in-process cache.

```go
api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	b, _ := os.ReadFile("testdata/gp-" + r.URL.Query().Get("CATNR") + ".json")
	_, _ = w.Write(b)
}))
defer api.Close()
shared := satellite.NewMemoryCache()
a := celestrak.New(celestrak.WithBaseURL(api.URL), celestrak.WithCache(shared))
b := celestrak.New(celestrak.WithBaseURL("http://127.0.0.1:1"), celestrak.WithCache(shared)) // never reached
_, _ = a.Fetch(context.Background(), 20580)
r, err := b.Fetch(context.Background(), 20580)
fmt.Println(r.Elements.SatNum, err)
```

```text
20580 <nil>
```
