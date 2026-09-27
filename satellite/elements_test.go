package satellite

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
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

type pyReference struct {
	Line1, Line2 string
	Cases        []struct {
		UTC   []float64 `json:"utc"`
		Error int       `json:"error"`
		R     []float64 `json:"r"`
		V     []float64 `json:"v"`
	} `json:"cases"`
	GMST []struct {
		JD   float64 `json:"jd"`
		GMST float64 `json:"gmst"`
	} `json:"gmst"`
}

func readPyReference(t *testing.T) pyReference {
	t.Helper()
	data, err := os.ReadFile("testdata/iss_python.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref pyReference
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	return ref
}

// TestISSAgainstPython checks Propagate(time.Time) against python-sgp4 2.25
// (scripts/iss_crosscheck.py) at several instants around the epoch.
func TestISSAgainstPython(t *testing.T) {
	ref := readPyReference(t)
	if ref.Line1 != issLine1 || ref.Line2 != issLine2 {
		t.Fatal("reference file was made from different TLE lines")
	}
	e, err := ParseTLE(issLine1, issLine2)
	if err != nil {
		t.Fatal(err)
	}
	var maxPos, maxVel float64
	for _, c := range ref.Cases {
		sec, frac := math.Modf(c.UTC[5])
		ts := time.Date(int(c.UTC[0]), time.Month(c.UTC[1]), int(c.UTC[2]),
			int(c.UTC[3]), int(c.UTC[4]), int(sec), int(math.Round(frac*1e9)), time.UTC)
		r, v, err := e.Propagate(ts)
		if err != nil || c.Error != 0 {
			t.Errorf("%v: go error %v, python error %d", ts, err, c.Error)
			continue
		}
		for k := 0; k < 3; k++ {
			maxPos = math.Max(maxPos, math.Abs(r[k]-c.R[k]))
			maxVel = math.Max(maxVel, math.Abs(v[k]-c.V[k]))
		}
	}
	t.Logf("ISS vs python-sgp4 over %d instants: max position error %.3g km, max velocity error %.3g km/s",
		len(ref.Cases), maxPos, maxVel)
	if maxPos > 1e-6 || maxVel > 1e-9 {
		t.Errorf("ISS differs from python-sgp4: %g km, %g km/s", maxPos, maxVel)
	}
}

// Propagate must agree with PropagateMinutes, and Elements must be safe to
// share between goroutines.
func TestPropagateTimeAndConcurrency(t *testing.T) {
	e, err := ParseTLE(issLine1, issLine2)
	if err != nil {
		t.Fatal(err)
	}
	at := e.Epoch().Add(1234*time.Minute + 30*time.Second)
	r1, v1, err := e.Propagate(at)
	if err != nil {
		t.Fatal(err)
	}
	r2, v2, _ := e.PropagateMinutes(1234.5)
	if r1 != r2 || v1 != v2 {
		t.Errorf("Propagate %v %v != PropagateMinutes %v %v", r1, v1, r2, v2)
	}

	tles := readVerTLE(t)
	deep, err := parseTLE(tles[3].line1, tles[3].line2, false) // 08195, 12 h resonant
	if err != nil {
		t.Fatal(err)
	}
	want, _, _ := deep.PropagateMinutes(2880)
	done := make(chan [3]float64)
	for i := 0; i < 8; i++ {
		go func() {
			_, _, _ = deep.PropagateMinutes(float64(i * 360))
			r, _, _ := deep.PropagateMinutes(2880)
			done <- r
		}()
	}
	for i := 0; i < 8; i++ {
		if r := <-done; r != want {
			t.Errorf("concurrent result %v, want %v", r, want)
		}
	}
}
