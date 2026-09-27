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

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/openmeteo"
)

// Lookup with the defaults: the shared resolver and the real Open-Meteo API.
// (Not run as a test, which never reaches the network.)
func ExampleResolveObserver() {
	obs, source, err := openmeteo.ResolveObserver(context.Background(), 39.74, -104.99)
	if err != nil {
		return // the context was cancelled or expired
	}
	if source == astronomy.SourceSeaLevel {
		fmt.Println("lookup failed; using sea level")
	}
	fmt.Println(obs.Height)
}

// The composed form: a static height first, then Open-Meteo through an
// injected HTTP client, and sea level if neither answers. A stand-in server
// plays the API here.
func ExampleNew() {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"elevation":[1597.0]}`)
	}))
	defer api.Close()
	client := api.Client()

	lookup := openmeteo.New(openmeteo.WithHTTPClient(client), openmeteo.WithBaseURL(api.URL))

	// Nothing configured, so Open-Meteo answers.
	h, source, _ := astronomy.ChainElevation(astronomy.CallerElevation(astronomy.SeaLevel, false), lookup).
		Elevation(context.Background(), 39.74, -104.99)
	fmt.Println(h, "from", source)

	// A configured height wins, and the API is not asked.
	h, source, _ = astronomy.ChainElevation(astronomy.StaticElevation(astronomy.Meters(1609)), lookup).
		Elevation(context.Background(), 39.74, -104.99)
	fmt.Println(h, "from", source)
	// Output:
	// 1597 m from open-meteo
	// 1609 m from static
}

// Store and Cached connect the in-process cache to a durable one.
func ExampleResolver_Store() {
	r := openmeteo.New()
	r.Store(-16.50, -68.15, astronomy.Meters(3640)) // seeded from the consumer's own store
	h, ok := r.Cached(-16.5004, -68.1498)
	fmt.Println(h, ok)
	// Output:
	// 3640 m true
}
