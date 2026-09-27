package satellite_test

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
	"os"
	"strings"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite"
)

// TestParseOMMAgainstPython reads the JSON and XML OMM samples and checks the
// propagated states against python-sgp4's own OMM reader.
func TestParseOMMAgainstPython(t *testing.T) {
	t.Parallel()
	sets := map[string]satellite.Elements{}
	for _, name := range []string{"omm-vanguard.json", "omm-vanguard.xml"} {
		f, err := os.Open("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		els, err := satellite.ParseOMM(f)
		_ = f.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(els) != 1 || els[0].SatNum != 5 || els[0].IntlDesignator != "58002B" {
			t.Fatalf("%s: parsed %+v", name, els)
		}
		sets[name] = els[0]
	}
	rows := fields(t, "testdata/omm_python.txt")
	for _, r := range rows {
		e := sets[r[0]]
		pos, vel, err := e.PropagateMinutes(num(t, r[1]))
		if err != nil {
			t.Fatal(err)
		}
		for i := range 3 {
			if d := math.Abs(pos[i] - num(t, r[2+i])); d > 1e-6 {
				t.Errorf("%s %s min: position[%d] off by %.2e km", r[0], r[1], i, d)
			}
			if d := math.Abs(vel[i] - num(t, r[5+i])); d > 1e-9 {
				t.Errorf("%s %s min: velocity[%d] off by %.2e km/s", r[0], r[1], i, d)
			}
		}
	}
	if len(rows) != 8 {
		t.Fatalf("%d reference rows, want 8", len(rows))
	}
	want := time.Date(2025, 2, 14, 14, 36, 48, 662784000, time.UTC)
	if got := sets["omm-vanguard.json"].Epoch(); !got.Equal(want) {
		t.Errorf("epoch %s, want %s", got, want)
	}
}

// TestParseOMMErrors checks malformed and unsuitable messages are rejected
// with the coded ErrMalformedOMM.
func TestParseOMMErrors(t *testing.T) {
	t.Parallel()
	base := `{"OBJECT_ID":"1958-002B","EPOCH":"2025-02-14T14:36:48.662784","MEAN_MOTION":10.85873516,` +
		`"ECCENTRICITY":0.1841322,"INCLINATION":34.2493,"RA_OF_ASC_NODE":19.2327,"ARG_OF_PERICENTER":100.1057,` +
		`"MEAN_ANOMALY":281.1229,"NORAD_CAT_ID":5,"BSTAR":0.00035436`
	cases := map[string]string{
		"empty":        "  ",
		"not json/xml": "1 25544U",
		"bad json":     "[{",
		"no records":   "[]",
		"missing":      `{"EPOCH":"2025-02-14T14:36:48"}`,
		"bad epoch":    strings.Replace(base, "2025-02-14T14:36:48.662784", "yesterday", 1) + "}",
		"not sgp4":     base + `,"MEAN_ELEMENT_THEORY":"DSST"}`,
		"not teme":     base + `,"REF_FRAME":"GCRF"}`,
		"bad number":   strings.Replace(base, "0.1841322", `"x"`, 1) + "}",
	}
	for name, msg := range cases {
		_, err := satellite.ParseOMM(strings.NewReader(msg))
		code, _ := apperr.Code(err)
		if !errors.Is(err, satellite.ErrMalformedOMM) || code != astronomy.CodeInvalidElements {
			t.Errorf("%s: error = %v (code %d)", name, err, code)
		}
	}
	if _, err := satellite.ParseOMM(strings.NewReader(base + "}")); err != nil {
		t.Errorf("single object: %v", err)
	}
}
