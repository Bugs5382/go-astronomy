package celestrak_test

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/Bugs5382/go-astronomy/satellite"
	"github.com/Bugs5382/go-astronomy/satellite/celestrak"
)

// A client pointed at a stand-in for CelesTrak serving the recorded ISS set:
// the first call fetches, the second is served from the cache.
func ExampleClient_Fetch() {
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
	// Output:
	// 25544 2026-09-26T20:26:13Z fetched true
	// 25544 2026-09-26T20:26:13Z fetched true
	// requests: 1
}

// Sharing the cache between replicas: any satellite.Cache works, for example
// one backed by Redis. Here two clients share one in-process cache.
func ExampleWithCache() {
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
	// Output:
	// 20580 <nil>
}
