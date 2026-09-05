// Package constellation identifies which of the 88 IAU constellations a
// celestial position falls in, and lists the constellations. Identification is a
// universal, observer-independent geometry problem: it depends only on a point's
// equatorial coordinates, not on where or when it is seen, so it lives outside
// any vantage-body package.
//
// The boundaries are the official IAU constellation boundaries fixed by Delporte
// (1930) and rearranged for position lookup by Roman (1987), distributed as
// VizieR catalog VI/42 and embedded here. Because those boundaries are defined
// at the B1875.0 equinox, FindAt precesses the query position (given at the
// J2000.0 equinox, matching the star package) back to B1875.0 before testing it
// against the table. The lookup follows Roman's method: the southern boundary
// arcs are ordered by declination, and the first arc the point lies above and
// within in right ascension names the constellation.
//
// Constellations are a convention of the Earth's sky; a different vantage would
// draw different figures. This package ships the IAU convention as its data. It
// is stateless and concurrency-safe; the boundary table is parsed once on first
// use.
package constellation

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
	"sync"

	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
)

// epochJ2000 is the equinox of the coordinates FindAt accepts, and epochB1875 is
// the equinox of the embedded boundary table. B1875.0 is a Besselian epoch; its
// Julian-year equivalent, used by the precession model, is 1875.0287.
const (
	epochJ2000 = 2000.0
	epochB1875 = 1875.0287
)

//go:embed data/boundaries.csv data/constellations.csv
var dataFS embed.FS

// Constellation is one of the 88 IAU constellations, identified by its standard
// three-letter abbreviation together with its Latin name and genitive form (the
// genitive is the form used in star designations, as in "Alpha Centauri" from
// "Centaurus"/"Centauri").
type Constellation struct {
	// Abbrev is the three-letter IAU abbreviation, such as "Ori".
	Abbrev string
	// Name is the Latin nominative name, such as "Orion".
	Name string
	// Genitive is the Latin genitive name, such as "Orionis".
	Genitive string
}

// boundary is one southern boundary arc from the Roman table: at declinations at
// or above deLow, over the right-ascension span [raLow, raUp) in hours, the
// constellation con lies immediately above the arc. All values are at the
// B1875.0 equinox.
type boundary struct {
	raLow float64 // hours
	raUp  float64 // hours
	deLow float64 // degrees
	con   string
}

// table holds the parsed boundary arcs in file order (descending deLow) together
// with the constellation metadata, built once on first use.
type table struct {
	arcs   []boundary
	byAbbr map[string]Constellation
	list   []Constellation
}

var load = sync.OnceValue(func() *table {
	t := &table{byAbbr: make(map[string]Constellation)}

	metaData, err := dataFS.ReadFile("data/constellations.csv")
	if err != nil {
		panic("constellation: reading embedded metadata: " + err.Error())
	}
	mr := csv.NewReader(bytes.NewReader(metaData))
	mr.FieldsPerRecord = 3
	if _, err := mr.Read(); err != nil { // header
		panic("constellation: reading metadata header: " + err.Error())
	}
	for {
		rec, err := mr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic("constellation: parsing metadata: " + err.Error())
		}
		c := Constellation{Abbrev: rec[0], Name: rec[1], Genitive: rec[2]}
		t.byAbbr[c.Abbrev] = c
		t.list = append(t.list, c)
	}
	sort.Slice(t.list, func(i, j int) bool { return t.list[i].Name < t.list[j].Name })

	bndData, err := dataFS.ReadFile("data/boundaries.csv")
	if err != nil {
		panic("constellation: reading embedded boundaries: " + err.Error())
	}
	br := csv.NewReader(bytes.NewReader(bndData))
	br.FieldsPerRecord = 4
	if _, err := br.Read(); err != nil { // header
		panic("constellation: reading boundaries header: " + err.Error())
	}
	for {
		rec, err := br.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic("constellation: parsing boundaries: " + err.Error())
		}
		t.arcs = append(t.arcs, boundary{
			raLow: mustFloat(rec[0]),
			raUp:  mustFloat(rec[1]),
			deLow: mustFloat(rec[2]),
			con:   rec[3],
		})
	}
	return t
})

func mustFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic("constellation: malformed numeric field " + strconv.Quote(s) + ": " + err.Error())
	}
	return v
}

// List returns the 88 IAU constellations ordered by Latin name. The returned
// slice is a fresh copy the caller may modify freely.
func List() []Constellation {
	t := load()
	out := make([]Constellation, len(t.list))
	copy(out, t.list)
	return out
}

// Lookup returns the constellation for a three-letter IAU abbreviation, such as
// "Ori", and reports whether it is one of the 88.
func Lookup(abbrev string) (Constellation, bool) {
	c, ok := load().byAbbr[abbrev]
	return c, ok
}

// FindAt returns the IAU constellation containing the given equatorial position
// and reports whether one was found. The right ascension and declination are in
// degrees at the J2000.0 equinox, matching the star package; raDeg is normalized
// into [0, 360) so any real value is accepted. It returns false only when decDeg
// lies outside the physical range [-90, 90]; every valid position lies in exactly
// one constellation.
//
// The position is precessed from J2000.0 to the boundary table's B1875.0 equinox
// and then matched against the Roman boundary arcs.
func FindAt(raDeg, decDeg float64) (Constellation, bool) {
	if decDeg < -90 || decDeg > 90 {
		return Constellation{}, false
	}
	t := load()

	eq := coordinates.PrecessEquatorial(
		coordinates.Equatorial{RA: angles.Normalize(raDeg), Dec: decDeg},
		epochJ2000, epochB1875,
	)
	raH := eq.RA / 15
	dec := eq.Dec

	for _, a := range t.arcs {
		if a.deLow > dec {
			continue
		}
		if a.raUp <= raH || a.raLow > raH {
			continue
		}
		if c, ok := t.byAbbr[a.con]; ok {
			return c, true
		}
		// The boundary table and metadata are embedded together and share the
		// same 88 abbreviations, so a miss here is a build defect.
		return Constellation{Abbrev: a.con}, true
	}
	return Constellation{}, false
}
