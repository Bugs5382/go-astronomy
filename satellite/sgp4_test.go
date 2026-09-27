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
	"bufio"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
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

// The error cases must fail the same way whatever the entry path.
func TestErrorSentinels(t *testing.T) {
	tles := readVerTLE(t)
	cases := []struct {
		idx      int
		sentinel error
	}{
		{11, ErrEccentricity},
		{25, ErrDecayed},
		{29, ErrSemiLatusRectum},
		{30, ErrPerturbedEccentricity},
	}
	for _, c := range cases {
		e, err := parseTLE(tles[c.idx].line1, tles[c.idx].line2, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = e.PropagateMinutes(verErrors[c.idx].tsince)
		if !errors.Is(err, c.sentinel) {
			t.Errorf("satellite %s: got %v, want %v", e.CatalogNumber, err, c.sentinel)
		}
	}

	var zero Elements
	if _, _, err := zero.Propagate(time.Now()); !errors.Is(err, ErrNotInitialised) {
		t.Errorf("zero Elements: got %v", err)
	}
}

const (
	issLine1 = "1 25544U 98067A   26269.51782528  .00016717  00000-0  30306-3 0  9990"
	issLine2 = "2 25544  51.6416 247.4627 0006703 130.5360 325.0288 15.49450470 12348"
)

func TestParseTLEFields(t *testing.T) {
	e, err := ParseTLE(issLine1, issLine2)
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"CatalogNumber", e.CatalogNumber, "25544"},
		{"SatNum", e.SatNum, 25544},
		{"Classification", e.Classification, byte('U')},
		{"IntlDesignator", e.IntlDesignator, "98067A"},
		{"EpochYear", e.EpochYear, 2026},
		{"EpochDay", e.EpochDay, 269.51782528},
		{"NDot", e.NDot, 0.00016717},
		{"NDDot", e.NDDot, 0.0},
		{"BStar", e.BStar, 0.30306e-3},
		{"EphemerisType", e.EphemerisType, 0},
		{"ElementSetNumber", e.ElementSetNumber, 999},
		{"Inclination", e.Inclination, 51.6416},
		{"RAAN", e.RAAN, 247.4627},
		{"Eccentricity", e.Eccentricity, 0.0006703},
		{"ArgPerigee", e.ArgPerigee, 130.5360},
		{"MeanAnomaly", e.MeanAnomaly, 325.0288},
		{"MeanMotion", e.MeanMotion, 15.49450470},
		{"RevNumber", e.RevNumber, 1234}, // "12345" sits in columns 65-69, so the last digit is the checksum
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	wantEpoch := time.Date(2026, time.September, 26, 12, 25, 40, 104192000, time.UTC)
	if !e.Epoch().Equal(wantEpoch) {
		t.Errorf("Epoch = %v, want %v", e.Epoch(), wantEpoch)
	}
	if got := e.Age(wantEpoch.Add(90 * time.Minute)); got != 90*time.Minute {
		t.Errorf("Age = %v", got)
	}
	if got := e.Age(wantEpoch.Add(-time.Hour)); got != -time.Hour {
		t.Errorf("Age before epoch = %v", got)
	}
}

// withChecksum replaces column 69 with the correct checksum digit.
func withChecksum(line string) string {
	return line[:68] + strconv.Itoa(tleChecksum(line))
}

func TestParseTLEErrors(t *testing.T) {
	cases := []struct {
		name   string
		l1, l2 string
		want   error
	}{
		{"checksum line 1", issLine1[:68] + "3", issLine2, ErrChecksum},
		{"checksum line 2", issLine1, issLine2[:68] + "5", ErrChecksum},
		{"short line 1", issLine1[:60], issLine2, ErrMalformedTLE},
		{"short line 2", issLine1, issLine2[:68], ErrMalformedTLE},
		{"long line", issLine1 + "0", issLine2, ErrMalformedTLE},
		{"no checksum digit", issLine1[:68] + "x", issLine2, ErrMalformedTLE},
		{"lines swapped", issLine2, issLine1, ErrMalformedTLE},
		{"catalog mismatch", issLine1, withChecksum("2 25545" + issLine2[7:]), ErrMalformedTLE},
		{"bad number", withChecksum(issLine1[:20] + "2x9.51782528" + issLine1[32:]), issLine2, ErrMalformedTLE},
		{"bad alpha-5 letter", withChecksum("1 I0001" + issLine1[7:]), withChecksum("2 I0001" + issLine2[7:]), ErrMalformedTLE},
		{"non-ASCII", strings.Replace(issLine1, "U", "\u00dc", 1), issLine2, ErrMalformedTLE},
		{"empty", "", "", ErrMalformedTLE},
	}
	for _, c := range cases {
		_, err := ParseTLE(c.l1, c.l2)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		} else {
			t.Logf("%s: %v", c.name, err)
		}
	}

	// Trailing whitespace and CRLF endings are fine.
	if _, err := ParseTLE(issLine1+"  \r\n", issLine2+"\r"); err != nil {
		t.Errorf("trailing whitespace: %v", err)
	}
}

func TestAlpha5(t *testing.T) {
	cases := []struct {
		cat  string
		want int
	}{
		{"A0001", 100001},
		{"B1234", 111234},
		{"H9999", 179999},
		{"J0000", 180000}, // I is skipped
		{"P0000", 230000}, // O is skipped
		{"Z9999", 339999},
		{"00005", 5},
		{"99999", 99999},
	}
	for _, c := range cases {
		l1 := withChecksum("1 " + c.cat + issLine1[7:])
		l2 := withChecksum("2 " + c.cat + issLine2[7:])
		e, err := ParseTLE(l1, l2)
		if err != nil {
			t.Errorf("%s: %v", c.cat, err)
			continue
		}
		if e.SatNum != c.want || e.CatalogNumber != c.cat {
			t.Errorf("%s: got %d (%q), want %d", c.cat, e.SatNum, e.CatalogNumber, c.want)
		}
	}
}

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

func TestGMST82(t *testing.T) {
	ref := readPyReference(t)
	for _, g := range ref.GMST {
		if got := gmst82(g.JD); math.Abs(got-g.GMST) > 1e-12 {
			t.Errorf("gmst82(%v) = %v, want %v", g.JD, got, g.GMST)
		}
	}
	// J2000.0: 67310.54841 s of sidereal time = 280.46061837 degrees.
	if got := gmst82(2451545.0) / deg2rd; math.Abs(got-280.46061837504) > 1e-9 {
		t.Errorf("gmst82(J2000) = %v deg", got)
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
