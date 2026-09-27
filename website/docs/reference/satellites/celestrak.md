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

- **One request per set per day.** Each set is cached by catalogue number for `DefaultTTL` (24 hours) and never refetched sooner than `MinRefetch` (2 hours), following CelesTrak's usage guidance: sets change a few times a day, so download each once and reuse it. All positions and passes are then propagated locally from the cached set.
- **Failures.** A failed fetch returns an error wrapping `ErrFetch`. When a set was cached, it comes back too, with `Stale` set and an error wrapping `satellite.ErrStaleElements`, and the fetch is not retried for `MinRefetch`. With nothing cached, there is only the error. A stale set is never returned unflagged.
- **Shared cache.** `WithCache` takes any `satellite.Cache`, for example one backed by Redis, so replicas and restarts share one fetch. The cache keeps a set for 14 days after its fetch, so it can still be served as stale.
- **Logging.** It logs through go-log: catalogue numbers, cache hits, requests, and failures. It logs nothing else about the caller.

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
