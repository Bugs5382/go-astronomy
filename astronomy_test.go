package astronomy

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
	"testing"
	"time"
)

func TestObserverLocationDefaultsToUTC(t *testing.T) {
	t.Parallel()
	o := Observer{Lat: 40.678, Lng: -73.944}
	if got := o.Location(); got != time.UTC {
		t.Errorf("Location() = %v, want UTC when TZ is nil", got)
	}
}

func TestObserverLocationUsesTZ(t *testing.T) {
	t.Parallel()
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tz database unavailable: %v", err)
	}
	o := Observer{Lat: 40.678, Lng: -73.944, TZ: tz}
	if got := o.Location(); got != tz {
		t.Errorf("Location() = %v, want %v", got, tz)
	}
}

func TestHorizontalAboveHorizon(t *testing.T) {
	t.Parallel()
	cases := []struct {
		alt  float64
		want bool
	}{
		{10, true},
		{0, false},
		{-0.5, false},
		{89.9, true},
	}
	for _, c := range cases {
		h := Horizontal{Altitude: c.alt, Azimuth: 123}
		if got := h.AboveHorizon(); got != c.want {
			t.Errorf("AboveHorizon(alt=%v) = %v, want %v", c.alt, got, c.want)
		}
	}
}

func TestAngularDiameterRadius(t *testing.T) {
	t.Parallel()
	d := AngularDiameter(0.5)
	if got := d.Radius(); got != 0.25 {
		t.Errorf("Radius() = %v, want 0.25", got)
	}
}

func TestPositionEmbedsHorizontal(t *testing.T) {
	t.Parallel()
	p := Position{
		Horizontal: Horizontal{Altitude: 30, Azimuth: 200},
		Diameter:   AngularDiameter(0.53),
	}
	if !p.AboveHorizon() {
		t.Error("expected embedded AboveHorizon to report true")
	}
	if p.Azimuth != 200 {
		t.Errorf("embedded Azimuth = %v, want 200", p.Azimuth)
	}
}
