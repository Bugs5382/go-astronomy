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
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

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
