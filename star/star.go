// Package star provides a universal, observer-independent catalog of stars: the
// quantities that belong to a star itself, not to any vantage. Each Star carries
// its J2000 equatorial position (right ascension and declination in degrees),
// its distance in parsecs, its apparent visual magnitude, and its naming and
// catalog identifiers. There is deliberately no altitude, azimuth, or observer
// here: turning a star's equatorial position into a local alt-az direction needs
// a rotating body and an observer, which is a vantage concern handled by a body
// package (the Earth vantage lives in the earth package as StarPosition).
//
// The catalog is a curated subset of the HYG database (Hipparcos-Yale-Gliese,
// https://codeberg.org/astronexus/hyg), embedded at build time. To keep the
// binary small the subset is the naked-eye sky plus every star with a proper
// name: all stars with apparent magnitude at or below 6.5, together with every
// named star regardless of magnitude. This is a few thousand stars rather than
// the full ~120,000-row catalog. The HYG data is distributed under the Creative
// Commons Attribution-ShareAlike 4.0 license; see NOTICE and star/data.
//
// Distance is retained in parsecs so a future vantage other than Earth can
// reproject a star from its true three-dimensional position rather than treating
// the sky as an infinitely distant sphere. The package is stateless and
// concurrency-safe; the catalog is parsed once on first use and never mutated.
package star

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
	"bytes"
	"embed"
	"encoding/csv"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// MagnitudeCutoff is the faintest apparent visual magnitude included in the
// embedded catalog by brightness alone. Fainter stars are present only when they
// carry a proper name. Larger magnitude numbers are fainter, so a star is in the
// catalog when its magnitude is at or below this value or it has a proper name.
const MagnitudeCutoff = 6.5

//go:embed data/stars.csv
var catalogFS embed.FS

// ErrEmptyMagnitudeRange is the cause returned by InMagnitudeRange when its
// bright bound is fainter than its faint bound, which can never select any star.
// The coded error carries astronomy.CodeInvalidMagnitudeRange. Match it with
// errors.Is(err, ErrEmptyMagnitudeRange) or recover the code with
// apperr.Code(err).
var ErrEmptyMagnitudeRange = errors.New("star: magnitude range is empty (bright bound fainter than faint bound)")

// Star is one entry in the catalog: a single star described by the quantities
// that are intrinsic to it, independent of any observer. All angles are in
// degrees and the position is referred to the J2000 equinox and equator.
type Star struct {
	// ProperName is the star's common or IAU-approved name, such as "Sirius",
	// or the empty string when the star has none.
	ProperName string
	// Designation is the Bayer or Flamsteed designation, such as "Alp CMa" or
	// "9Alp CMa", or the empty string when the star has none.
	Designation string
	// Constellation is the three-letter IAU abbreviation of the constellation
	// the star lies in, such as "CMa".
	Constellation string
	// RA is the right ascension in degrees, in [0, 360), at the J2000 equinox.
	RA float64
	// Dec is the declination in degrees, in [-90, 90], at the J2000 equinox.
	Dec float64
	// Distance is the distance from the Sun in parsecs, or 0 when unknown.
	Distance float64
	// Magnitude is the apparent visual magnitude; smaller values are brighter.
	Magnitude float64
	// HIP is the Hipparcos catalog number, or the empty string when absent.
	HIP string
	// HD is the Henry Draper catalog number, or the empty string when absent.
	HD string
	// HR is the Harvard Revised (Bright Star) catalog number, or the empty
	// string when absent.
	HR string
	// Gliese is the Gliese-Jahreiss nearby-star catalog identifier, or the empty
	// string when absent.
	Gliese string
}

// catalog holds the parsed, immutable catalog and the lookup index built from
// it. It is constructed once on first use.
type catalog struct {
	all    []Star
	byName map[string]int // lowercased proper name -> index into all
}

// load parses the embedded catalog exactly once. The stars are sorted by
// ascending magnitude (brightest first), with proper name and designation as
// tie-breakers, so every accessor returns a stable, deterministic order.
var load = sync.OnceValue(func() *catalog {
	data, err := catalogFS.ReadFile("data/stars.csv")
	if err != nil {
		// The file is embedded at build time, so a read failure is a build
		// defect in this package, not a runtime input error.
		panic("star: reading embedded catalog: " + err.Error())
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = 11
	r.ReuseRecord = true

	// Discard the header row.
	if _, err := r.Read(); err != nil {
		panic("star: reading catalog header: " + err.Error())
	}

	var all []Star
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic("star: parsing catalog: " + err.Error())
		}
		all = append(all, Star{
			ProperName:    rec[0],
			Designation:   rec[1],
			Constellation: rec[2],
			RA:            mustFloat(rec[3]),
			Dec:           mustFloat(rec[4]),
			Distance:      mustFloat(rec[5]),
			Magnitude:     mustFloat(rec[6]),
			HIP:           rec[7],
			HD:            rec[8],
			HR:            rec[9],
			Gliese:        rec[10],
		})
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Magnitude != all[j].Magnitude {
			return all[i].Magnitude < all[j].Magnitude
		}
		if all[i].ProperName != all[j].ProperName {
			return all[i].ProperName < all[j].ProperName
		}
		return all[i].Designation < all[j].Designation
	})

	byName := make(map[string]int)
	for i, s := range all {
		if s.ProperName == "" {
			continue
		}
		key := strings.ToLower(s.ProperName)
		// Keep the brightest star for any duplicated name; the slice is already
		// sorted brightest-first, so only record the first occurrence.
		if _, ok := byName[key]; !ok {
			byName[key] = i
		}
	}
	return &catalog{all: all, byName: byName}
})

// mustFloat parses a catalog float field, panicking on malformed embedded data,
// which would be a build defect rather than a runtime input error.
func mustFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic("star: malformed numeric field " + strconv.Quote(s) + ": " + err.Error())
	}
	return v
}

// All returns every star in the catalog, ordered from brightest to faintest.
// The returned slice is a fresh copy that the caller may modify freely.
func All() []Star {
	c := load()
	out := make([]Star, len(c.all))
	copy(out, c.all)
	return out
}

// Count returns the number of stars in the embedded catalog.
func Count() int { return len(load().all) }

// GetNamedStar looks a star up by its proper name, case-insensitively, and
// reports whether one was found. Surrounding whitespace in name is ignored. When
// two catalog stars share a name, the brighter one is returned.
func GetNamedStar(name string) (Star, bool) {
	c := load()
	key := strings.ToLower(strings.TrimSpace(name))
	i, ok := c.byName[key]
	if !ok {
		return Star{}, false
	}
	return c.all[i], true
}

// ListNamedStars returns every star that has a proper name, ordered from
// brightest to faintest. The returned slice is a fresh copy.
func ListNamedStars() []Star {
	c := load()
	out := make([]Star, 0, len(c.byName))
	for _, s := range c.all {
		if s.ProperName != "" {
			out = append(out, s)
		}
	}
	return out
}

// Brighter returns every star at least as bright as maxMagnitude, that is with
// apparent magnitude at or below it, ordered from brightest to faintest. A very
// large bound returns the whole catalog. The returned slice is a fresh copy.
func Brighter(maxMagnitude float64) []Star {
	c := load()
	var out []Star
	for _, s := range c.all {
		if s.Magnitude <= maxMagnitude {
			out = append(out, s)
		}
	}
	return out
}

// InMagnitudeRange returns every star whose apparent magnitude lies in the
// inclusive range [brightest, faintest], ordered from brightest to faintest.
// Because smaller magnitudes are brighter, brightest must be the smaller number;
// InMagnitudeRange returns a go-apperr coded error wrapping ErrEmptyMagnitudeRange
// when brightest exceeds faintest. The returned slice is a fresh copy.
func InMagnitudeRange(brightest, faintest float64) ([]Star, error) {
	if brightest > faintest {
		return nil, apperr.Coded(astronomy.CodeInvalidMagnitudeRange, ErrEmptyMagnitudeRange)
	}
	c := load()
	var out []Star
	for _, s := range c.all {
		if s.Magnitude >= brightest && s.Magnitude <= faintest {
			out = append(out, s)
		}
	}
	return out, nil
}
