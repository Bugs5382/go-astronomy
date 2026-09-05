package star_test

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
	"errors"
	"math"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/star"
)

func TestGetNamedStarKnownPositions(t *testing.T) {
	t.Parallel()
	// J2000 right ascension and declination in degrees, from the reference
	// literature, used as an independent cross-check of the embedded catalog.
	cases := []struct {
		name    string
		ra, dec float64
		magMax  float64 // sanity upper bound on apparent magnitude
	}{
		{"Sirius", 101.2871, -16.7161, 0.0},
		{"Vega", 279.2346, 38.7837, 0.5},
		{"Polaris", 37.9529, 89.2641, 2.5},
		{"Betelgeuse", 88.7929, 7.4071, 1.5},
		{"Rigel", 78.6345, -8.2016, 0.5},
	}
	for _, c := range cases {
		s, ok := star.GetNamedStar(c.name)
		if !ok {
			t.Errorf("GetNamedStar(%q) not found", c.name)
			continue
		}
		if math.Abs(s.RA-c.ra) > 0.02 {
			t.Errorf("%s RA = %.4f, want ~%.4f", c.name, s.RA, c.ra)
		}
		if math.Abs(s.Dec-c.dec) > 0.02 {
			t.Errorf("%s Dec = %.4f, want ~%.4f", c.name, s.Dec, c.dec)
		}
		if s.Magnitude > c.magMax {
			t.Errorf("%s magnitude = %.2f, want <= %.2f", c.name, s.Magnitude, c.magMax)
		}
		if s.RA < 0 || s.RA >= 360 {
			t.Errorf("%s RA = %v out of [0,360)", c.name, s.RA)
		}
		if s.Dec < -90 || s.Dec > 90 {
			t.Errorf("%s Dec = %v out of [-90,90]", c.name, s.Dec)
		}
	}
}

func TestGetNamedStarCaseInsensitiveAndTrimmed(t *testing.T) {
	t.Parallel()
	want, ok := star.GetNamedStar("Sirius")
	if !ok {
		t.Fatal("Sirius not found")
	}
	for _, q := range []string{"sirius", "SIRIUS", "  Sirius  ", "sIrIuS"} {
		got, ok := star.GetNamedStar(q)
		if !ok {
			t.Errorf("GetNamedStar(%q) not found", q)
			continue
		}
		if got.ProperName != want.ProperName || got.RA != want.RA {
			t.Errorf("GetNamedStar(%q) = %+v, want %+v", q, got, want)
		}
	}
}

func TestGetNamedStarUnknown(t *testing.T) {
	t.Parallel()
	for _, q := range []string{"", "Nemo", "Not A Star", "   "} {
		if s, ok := star.GetNamedStar(q); ok {
			t.Errorf("GetNamedStar(%q) = %+v, want not found", q, s)
		}
	}
}

func TestSiriusMetadata(t *testing.T) {
	t.Parallel()
	s, ok := star.GetNamedStar("Sirius")
	if !ok {
		t.Fatal("Sirius not found")
	}
	if s.Constellation != "CMa" {
		t.Errorf("Sirius constellation = %q, want CMa", s.Constellation)
	}
	if s.Distance <= 0 || s.Distance > 5 {
		t.Errorf("Sirius distance = %v pc, want a few pc", s.Distance)
	}
	if s.HIP == "" && s.HD == "" && s.HR == "" {
		t.Error("Sirius has no catalog identifiers")
	}
}

func TestListNamedStars(t *testing.T) {
	t.Parallel()
	named := star.ListNamedStars()
	if len(named) < 100 {
		t.Fatalf("only %d named stars, expected many", len(named))
	}
	for i, s := range named {
		if s.ProperName == "" {
			t.Errorf("ListNamedStars[%d] has no proper name", i)
		}
		if i > 0 && named[i-1].Magnitude > s.Magnitude {
			t.Errorf("named stars not brightest-first at %d: %.2f then %.2f",
				i, named[i-1].Magnitude, s.Magnitude)
		}
	}
}

func TestCatalogSubsetInvariant(t *testing.T) {
	t.Parallel()
	all := star.All()
	if len(all) != star.Count() {
		t.Fatalf("All() len %d != Count() %d", len(all), star.Count())
	}
	if len(all) < 5000 {
		t.Fatalf("catalog has only %d stars, expected the naked-eye sky", len(all))
	}
	for i, s := range all {
		// Every embedded star is either naked-eye bright or has a proper name.
		if s.Magnitude > star.MagnitudeCutoff && s.ProperName == "" {
			t.Errorf("star %d (%q) mag %.2f exceeds cutoff and is unnamed",
				i, s.Designation, s.Magnitude)
		}
		if i > 0 && all[i-1].Magnitude > s.Magnitude {
			t.Errorf("catalog not brightest-first at %d", i)
		}
	}
}

func TestBrighter(t *testing.T) {
	t.Parallel()
	bright := star.Brighter(1.0)
	if len(bright) == 0 {
		t.Fatal("no stars brighter than magnitude 1.0")
	}
	for _, s := range bright {
		if s.Magnitude > 1.0 {
			t.Errorf("Brighter(1.0) returned %q at mag %.2f", s.ProperName, s.Magnitude)
		}
	}
	// The first-magnitude sky has about two dozen stars; sanity-bound it.
	if len(bright) > 40 {
		t.Errorf("Brighter(1.0) returned %d stars, more than expected", len(bright))
	}
	// A very faint bound returns the whole catalog.
	if got := len(star.Brighter(100)); got != star.Count() {
		t.Errorf("Brighter(100) = %d, want %d", got, star.Count())
	}
}

func TestInMagnitudeRange(t *testing.T) {
	t.Parallel()
	in, err := star.InMagnitudeRange(-2, 2)
	if err != nil {
		t.Fatalf("InMagnitudeRange(-2,2) error: %v", err)
	}
	if len(in) == 0 {
		t.Fatal("no stars in [-2,2]")
	}
	for _, s := range in {
		if s.Magnitude < -2 || s.Magnitude > 2 {
			t.Errorf("%q mag %.2f outside [-2,2]", s.ProperName, s.Magnitude)
		}
	}
}

func TestInMagnitudeRangeEmptyIsCodedError(t *testing.T) {
	t.Parallel()
	_, err := star.InMagnitudeRange(3, 1)
	if err == nil {
		t.Fatal("InMagnitudeRange(3,1) expected an error")
	}
	if !errors.Is(err, star.ErrEmptyMagnitudeRange) {
		t.Errorf("error %v does not wrap ErrEmptyMagnitudeRange", err)
	}
	if code, ok := apperr.Code(err); !ok || code != astronomy.CodeInvalidMagnitudeRange {
		t.Errorf("apperr.Code = (%d,%v), want %d", code, ok, astronomy.CodeInvalidMagnitudeRange)
	}
}

func TestAllReturnsCopy(t *testing.T) {
	t.Parallel()
	a := star.All()
	if len(a) == 0 {
		t.Fatal("empty catalog")
	}
	orig := a[0].ProperName
	a[0].ProperName = "MUTATED"
	if star.All()[0].ProperName != orig {
		t.Error("All() shares backing storage; mutation leaked")
	}
}
