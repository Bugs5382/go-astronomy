package iss_test

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
	"errors"
	"os"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite"
	"github.com/Bugs5382/go-astronomy/satellite/iss"
)

// recorded returns the element set CelesTrak served on 2026-09-26, from
// testdata.
func recorded(t *testing.T) satellite.Elements {
	t.Helper()
	f, err := os.Open("testdata/gp-25544.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	sets, err := satellite.ParseOMM(f)
	if err != nil || len(sets) != 1 {
		t.Fatalf("%v, %d sets", err, len(sets))
	}
	return sets[0]
}

// TestIdentity checks the package fixes the catalogue number and name.
func TestIdentity(t *testing.T) {
	t.Parallel()
	if iss.CatalogNumber != 25544 || iss.Name != "ISS (ZARYA)" {
		t.Errorf("%d %q", iss.CatalogNumber, iss.Name)
	}
	tr := iss.New(satellite.StaticElements(recorded(t)))
	if tr.CatalogNumber() != 25544 || tr.Name() != iss.Name {
		t.Errorf("tracker %d %q", tr.CatalogNumber(), tr.Name())
	}
	if recorded(t).SatNum != iss.CatalogNumber {
		t.Errorf("the recorded set is for %d", recorded(t).SatNum)
	}
}

// TestPositionAndPasses checks the tracker matches the engine on the recorded
// set.
func TestPositionAndPasses(t *testing.T) {
	t.Parallel()
	e := recorded(t)
	tr := iss.New(satellite.StaticElements(e))
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	ctx := context.Background()
	when := e.Epoch().Add(3 * time.Hour)
	got, err := tr.Position(ctx, obs, when)
	want, _ := satellite.Position(obs, e, when)
	if err != nil || got != want {
		t.Errorf("Position %+v %v, engine %+v", got, err, want)
	}
	passes, err := tr.Passes(ctx, obs, e.Epoch(), e.Epoch().Add(48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	enginePasses, _ := satellite.Passes(obs, e, e.Epoch(), e.Epoch().Add(48*time.Hour), satellite.DefaultPassOptions())
	if len(passes) != len(enginePasses) {
		t.Errorf("%d passes, engine %d", len(passes), len(enginePasses))
	}
}

// TestNoElements checks a source without this satellite's set is an error,
// and the package never looks one up itself.
func TestNoElements(t *testing.T) {
	t.Parallel()
	tr := iss.New(satellite.StaticElements())
	if _, err := tr.Position(context.Background(), astronomy.Observer{}, time.Now()); !errors.Is(err, satellite.ErrNoElements) {
		t.Errorf("empty source: %v", err)
	}
}
