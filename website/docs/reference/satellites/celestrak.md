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
	DefaultBaseURL = "https://celestrak.org/NORAD/elements/gp.php"
	DefaultTTL     = 24 * time.Hour
	MinRefetch     = 2 * time.Hour
	DefaultTimeout = 15 * time.Second
)

var ErrFetch error

type Result struct {
	Elements  satellite.Elements
	FetchedAt time.Time
	Stale     bool
}

func New(opts ...Option) *Client
func (c *Client) Fetch(ctx context.Context, catalog int) (Result, error)
func (c *Client) Elements(ctx context.Context, catalog int) (satellite.Elements, error)

func WithHTTPClient(c *http.Client) Option
func WithBaseURL(u string) Option
func WithTimeout(d time.Duration) Option
func WithTTL(d time.Duration) Option      // never below MinRefetch
func WithCache(c satellite.Cache) Option  // default: satellite.NewMemoryCache()
func WithClock(now func() time.Time) Option
func WithLogger(l log.Logger) Option
```

## 🗄️ Caching: the network is hit rarely

**What is cached is the element set, keyed by catalogue number only** (`celestrak:gp:25544`), never anything per observer, time, or position. Every position and every pass, for any observer at any time or date, is propagated locally with SGP4 from that one cached set, with no fetch. A service answering thousands of visitors in different places makes one request per satellite per day.


- **One request per set per day.** Each set is cached by catalogue number for `DefaultTTL` (24 hours) and never refetched sooner than `MinRefetch` (2 hours), following CelesTrak's usage guidance: sets change a few times a day, so download each once and reuse it. All positions and passes are then propagated locally from the cached set.
- **Failures.** A failed fetch returns an error wrapping `ErrFetch`. When a set was cached, it comes back too, with `Stale` set and an error wrapping `satellite.ErrStaleElements`, and the fetch is not retried for `MinRefetch`. With nothing cached, there is only the error. A stale set is never returned unflagged.
- **Shared cache.** `WithCache` takes any `satellite.Cache`, for example one backed by Redis, so replicas and restarts share one fetch. The cache keeps a set for 14 days after its fetch, so it can still be served as stale.
- **Restarts.** The default `MemoryCache` lives in the process, so a restart loses it and the next call fetches again. To keep sets across restarts and share them between replicas, plug in a `satellite.Cache` backed by your own store, for example Redis:

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

- **Logging.** It logs through go-log: catalogue numbers, cache hits, requests, and failures. It logs nothing else about the caller.

## ⏳ How element sets go stale

An element set is a fit to tracking data at its epoch. It is propagated forward by a model that only approximates the atmosphere, so its error grows with age:

- **ISS.** The station is low enough for drag to matter, and it is reboosted every few weeks. A fresh set is good to about a kilometre, and the error grows by roughly a few kilometres a day. After a reboost, a set from before the burn can be tens of kilometres off. At 7.7 km/s a few kilometres is well under a second of pass timing and a fraction of a degree on the sky.
- **Higher and steadier orbits** (Hubble, Tiangong between reboosts) degrade more slowly.

CelesTrak updates the sets a few times a day, so refreshing once a day keeps the error to a few kilometres, which is plenty for placing a satellite in the sky and timing a pass. `Elements.Age` reports how old the set in hand is. A stale set (a refresh that failed) is flagged with `satellite.ErrStaleElements`, so a caller can decide whether it is still good enough.

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
	fmt.Println(r.Elements.SatNum, r.Elements.Epoch().Format(time.RFC3339), "stale:", r.Stale)
}
fmt.Println("requests:", requests)
```

```text
25544 2026-09-26T20:26:13Z stale: false
25544 2026-09-26T20:26:13Z stale: false
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
