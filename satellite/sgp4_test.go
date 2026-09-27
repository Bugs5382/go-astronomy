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

// The reference verification set: SGP4-VER.TLE against tcppver.out.

import (
	"bufio"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// verLine is one row of tcppver.out: minutes since epoch, then TEME
// position (km) and velocity (km/s).
type verLine struct {
	tsince float64
	r, v   [3]float64
}

type verBlock struct {
	satnum int
	lines  []verLine
}

func readTcppver(t *testing.T) []verBlock {
	t.Helper()
	f, err := os.Open("testdata/tcppver.out")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var blocks []verBlock
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if len(fields) == 2 && fields[1] == "xx" {
			n, err := strconv.Atoi(fields[0])
			if err != nil {
				t.Fatalf("bad block header %q", sc.Text())
			}
			blocks = append(blocks, verBlock{satnum: n})
			continue
		}
		if len(fields) < 7 || len(blocks) == 0 {
			t.Fatalf("unexpected line %q", sc.Text())
		}
		var vals [7]float64
		for i := range vals {
			if vals[i], err = strconv.ParseFloat(fields[i], 64); err != nil {
				t.Fatalf("bad number in %q: %v", sc.Text(), err)
			}
		}
		b := &blocks[len(blocks)-1]
		b.lines = append(b.lines, verLine{
			tsince: vals[0],
			r:      [3]float64{vals[1], vals[2], vals[3]},
			v:      [3]float64{vals[4], vals[5], vals[6]},
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return blocks
}

type verTLE struct {
	line1, line2 string
}

func readVerTLE(t *testing.T) []verTLE {
	t.Helper()
	data, err := os.ReadFile("testdata/SGP4-VER.TLE")
	if err != nil {
		t.Fatal(err)
	}
	var out []verTLE
	lines := strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n")
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "1 ") {
			l2 := lines[i+1]
			// The verification file adds start, stop and step columns
			// after the checksum; they are not part of the TLE.
			out = append(out, verTLE{line1: lines[i], line2: l2[:69]})
			i++
		}
	}
	return out
}

// verErrors lists the propagation errors the reference produces for the
// verification set, keyed by the position of the satellite in the file:
// the time (minutes) of the first failing step and the error code. The
// tcppver.out output for each of these stops just before that step.
var verErrors = map[int]struct {
	tsince float64
	code   int
}{
	11: {494.2028672, 1}, // 22312, eccentricity driven negative by drag
	22: {1560.0, 1},      // 28350
	25: {55.0, 6},        // 28872, sub-orbital, decayed
	26: {440.0, 6},       // 29141, decayed
	29: {25.0, 4},        // 33333, semi-latus rectum
	30: {0.0, 3},         // 33334, perturbed eccentricity at epoch
	32: {1844345.0, 6},   // 20413 at the far end, decayed
}

func TestVerificationSet(t *testing.T) {
	tles := readVerTLE(t)
	blocks := readTcppver(t)
	if len(tles) != len(blocks) {
		t.Fatalf("%d TLEs but %d output blocks", len(tles), len(blocks))
	}

	var maxPos, maxVel float64
	var maxPosAt, maxVelAt string
	compared := 0
	for i, tle := range tles {
		e, err := parseTLE(tle.line1, tle.line2, false)
		if err != nil {
			t.Fatalf("satellite %d: parse: %v", blocks[i].satnum, err)
		}
		if e.SatNum != blocks[i].satnum {
			t.Fatalf("entry %d: TLE is %d but output block is %d", i, e.SatNum, blocks[i].satnum)
		}
		want, hasErr := verErrors[i]
		for _, l := range blocks[i].lines {
			if hasErr && want.tsince == 0 {
				// tcppver.out repeats the previous satellite's line here.
				break
			}
			r, v, err := e.PropagateMinutes(l.tsince)
			if err != nil {
				t.Errorf("satellite %d at %v min: %v", e.SatNum, l.tsince, err)
				continue
			}
			for k := 0; k < 3; k++ {
				if d := math.Abs(r[k] - l.r[k]); d > maxPos {
					maxPos, maxPosAt = d, e.CatalogNumber+" @ "+strconv.FormatFloat(l.tsince, 'f', -1, 64)
				}
				if d := math.Abs(v[k] - l.v[k]); d > maxVel {
					maxVel, maxVelAt = d, e.CatalogNumber+" @ "+strconv.FormatFloat(l.tsince, 'f', -1, 64)
				}
			}
			compared++
		}
		if hasErr {
			r, _, err := e.PropagateMinutes(want.tsince)
			var pe *PropagationError
			if !errors.As(err, &pe) {
				t.Errorf("satellite %d at %v min: got error %v, want code %d", e.SatNum, want.tsince, err, want.code)
				continue
			}
			if pe.Code != want.code {
				t.Errorf("satellite %d at %v min: got code %d (%v), want %d", e.SatNum, want.tsince, pe.Code, err, want.code)
			}
			if want.code == 6 && r == ([3]float64{}) {
				t.Errorf("satellite %d: decayed result should still carry a position", e.SatNum)
			}
			t.Logf("satellite %d at %v min: %v", e.SatNum, want.tsince, err)
		}
	}

	// tcppver.out prints position to 1e-8 km and velocity to 1e-9 km/s, so
	// half of that is lost to rounding before any real difference shows.
	const posTol, velTol = 1e-6, 1e-9
	t.Logf("compared %d states; max position error %.3g km (%s), max velocity error %.3g km/s (%s)",
		compared, maxPos, maxPosAt, maxVel, maxVelAt)
	if maxPos > posTol {
		t.Errorf("max position error %g km exceeds %g", maxPos, posTol)
	}
	if maxVel > velTol {
		t.Errorf("max velocity error %g km/s exceeds %g", maxVel, velTol)
	}
	if compared < 500 {
		t.Errorf("only %d states compared", compared)
	}
}
