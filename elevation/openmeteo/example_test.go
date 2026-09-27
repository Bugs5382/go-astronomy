package openmeteo_test

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

	"github.com/Bugs5382/go-astronomy/elevation/openmeteo"
)

// A client pointed at a stand-in for the Open-Meteo API. In production, use
// openmeteo.New() with no options.
func ExampleClient_Elevation() {
	requests := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = fmt.Fprint(w, `{"elevation":[1597.0]}`)
	}))
	defer api.Close()

	client := openmeteo.New(openmeteo.WithBaseURL(api.URL))
	h, err := client.Elevation(context.Background(), 39.7392, -104.9903)
	if err != nil {
		panic(err)
	}
	fmt.Println(h, "m")

	// A point in the same 0.01 degree cell comes from the cache.
	h, _ = client.Elevation(context.Background(), 39.7401, -104.9897)
	fmt.Println(h, "m, requests:", requests)
	// Output:
	// 1597 m
	// 1597 m, requests: 1
}

// Store and Cached connect the in-process cache to a durable one.
func ExampleClient_Store() {
	client := openmeteo.New()
	client.Store(-16.50, -68.15, 3640) // seeded from the consumer's own store
	h, ok := client.Cached(-16.5004, -68.1498)
	fmt.Println(h, ok)
	// Output:
	// 3640 true
}
