package constellation_test

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

	"github.com/Bugs5382/go-astronomy/constellation"
	"github.com/Bugs5382/go-astronomy/star"
)

func TestListHas88Sorted(t *testing.T) {
	t.Parallel()
	list := constellation.List()
	if len(list) != 88 {
		t.Fatalf("List() returned %d constellations, want 88", len(list))
	}
	seen := make(map[string]bool)
	for i, c := range list {
		if c.Abbrev == "" || c.Name == "" || c.Genitive == "" {
			t.Errorf("constellation %d incomplete: %+v", i, c)
		}
		if seen[c.Abbrev] {
			t.Errorf("duplicate abbreviation %q", c.Abbrev)
		}
		seen[c.Abbrev] = true
		if i > 0 && list[i-1].Name > c.Name {
			t.Errorf("List() not sorted by name at %d: %q then %q", i, list[i-1].Name, c.Name)
		}
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()
	c, ok := constellation.Lookup("Ori")
	if !ok {
		t.Fatal("Lookup(Ori) not found")
	}
	if c.Name != "Orion" || c.Genitive != "Orionis" {
		t.Errorf("Ori = %+v, want Orion/Orionis", c)
	}
	if _, ok := constellation.Lookup("Xyz"); ok {
		t.Error("Lookup(Xyz) unexpectedly found")
	}
}

func TestFindAtKnownPoints(t *testing.T) {
	t.Parallel()
	// J2000 positions of unambiguous, well-interior stars and points.
	cases := []struct {
		name    string
		ra, dec float64
		want    string
	}{
		{"Betelgeuse", 88.7929, 7.4071, "Ori"},
		{"Rigel", 78.6345, -8.2016, "Ori"},
		{"Dubhe (UMa)", 165.9320, 61.7511, "UMa"},
		{"Sirius", 101.2871, -16.7161, "CMa"},
		{"Vega", 279.2346, 38.7837, "Lyr"},
		{"Antares", 247.3519, -26.4320, "Sco"},
		{"Aldebaran", 68.9800, 16.5093, "Tau"},
		{"Spica", 201.2983, -11.1613, "Vir"},
		{"Deneb", 310.3580, 45.2803, "Cyg"},
		{"Regulus", 152.0930, 11.9672, "Leo"},
		{"north pole", 0, 90, "UMi"},
		{"south pole", 0, -90, "Oct"},
		{"near south pole", 123.4, -89.9, "Oct"},
	}
	for _, c := range cases {
		got, ok := constellation.FindAt(c.ra, c.dec)
		if !ok {
			t.Errorf("FindAt(%s) not found", c.name)
			continue
		}
		if got.Abbrev != c.want {
			t.Errorf("FindAt(%s) = %q, want %q", c.name, got.Abbrev, c.want)
		}
	}
}

func TestFindAtRANormalization(t *testing.T) {
	t.Parallel()
	base, ok := constellation.FindAt(88.7929, 7.4071) // Betelgeuse
	if !ok {
		t.Fatal("base lookup failed")
	}
	for _, ra := range []float64{88.7929 + 360, 88.7929 - 360, 88.7929 + 720} {
		got, ok := constellation.FindAt(ra, 7.4071)
		if !ok || got.Abbrev != base.Abbrev {
			t.Errorf("FindAt(ra=%.4f) = (%q,%v), want %q", ra, got.Abbrev, ok, base.Abbrev)
		}
	}
}

func TestFindAtDeclinationOutOfRange(t *testing.T) {
	t.Parallel()
	for _, dec := range []float64{90.001, -90.001, 120, -200} {
		if got, ok := constellation.FindAt(10, dec); ok {
			t.Errorf("FindAt(dec=%v) = %q, want not found", dec, got.Abbrev)
		}
	}
}

// TestFindAtMatchesCatalog cross-checks FindAt against the HYG catalog's own
// constellation assignment for every named star. Both derive from the same IAU
// boundaries, so agreement should be near total; a handful of stars sit within a
// boundary arc-width of an edge, where the two pipelines' precession differs
// slightly, so a small number of disagreements is tolerated.
func TestFindAtMatchesCatalog(t *testing.T) {
	t.Parallel()
	named := star.ListNamedStars()
	var checked, mismatch int
	for _, s := range named {
		if s.Constellation == "" {
			continue
		}
		checked++
		got, ok := constellation.FindAt(s.RA, s.Dec)
		if !ok || got.Abbrev != s.Constellation {
			mismatch++
			t.Logf("mismatch: %s (RA %.3f Dec %.3f) catalog=%s findAt=%s",
				s.ProperName, s.RA, s.Dec, s.Constellation, got.Abbrev)
		}
	}
	if checked == 0 {
		t.Fatal("no named stars with a constellation to check")
	}
	// Allow up to 2% edge disagreements; a higher rate signals a real defect.
	if float64(mismatch)/float64(checked) > 0.02 {
		t.Errorf("FindAt disagreed with catalog on %d/%d named stars", mismatch, checked)
	}
}
