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

// Two-line element set parsing, following twoline2rv of the reference code.

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// Parse errors returned by ParseTLE, wrapped with details of the problem.
var (
	// ErrMalformedTLE means a line does not follow the fixed-column TLE
	// format.
	ErrMalformedTLE = errors.New("satellite: malformed TLE")
	// ErrChecksum means a line's modulo-10 checksum does not match.
	ErrChecksum = errors.New("satellite: TLE checksum mismatch")
)

// ParseTLE parses and validates a two-line element set and initialises the
// propagator. Both lines must be exactly 69 characters (trailing whitespace
// is ignored) and carry a valid checksum in column 69.
//
// Parsing succeeds even when the orbit cannot be propagated (a decayed
// object, say); those problems surface as errors from Propagate.
//
// Errors are go-apperr coded with astronomy.CodeInvalidElements and wrap
// ErrMalformedTLE or ErrChecksum.
func ParseTLE(line1, line2 string) (Elements, error) {
	e, err := parseTLE(line1, line2, true)
	if err != nil {
		return Elements{}, apperr.Coded(astronomy.CodeInvalidElements, err)
	}
	return e, nil
}

// parseTLE follows twoline2rv. With strict unset it skips the length and
// checksum checks, which the verification file needs: some of its entries
// have deliberately bad checksums and carry extra columns.
func parseTLE(line1, line2 string, strict bool) (Elements, error) {
	var e Elements

	l1 := strings.TrimRight(line1, " \t\r\n")
	l2 := strings.TrimRight(line2, " \t\r\n")
	for i, l := range []string{l1, l2} {
		for j := 0; j < len(l); j++ {
			if l[j] > 126 || (l[j] < 32) {
				return e, fmt.Errorf("%w: line %d has a non-ASCII or control character at column %d", ErrMalformedTLE, i+1, j+1)
			}
		}
	}
	if strict {
		for i, l := range []string{l1, l2} {
			if len(l) != 69 {
				return e, fmt.Errorf("%w: line %d is %d characters long, want 69", ErrMalformedTLE, i+1, len(l))
			}
		}
		for i, l := range []string{l1, l2} {
			want := l[68]
			if want < '0' || want > '9' {
				return e, fmt.Errorf("%w: line %d has no checksum digit in column 69", ErrMalformedTLE, i+1)
			}
			if got := tleChecksum(l); int(want-'0') != got {
				return e, fmt.Errorf("%w: line %d gives %c but tallies to %d", ErrChecksum, i+1, want, got)
			}
		}
	}

	if !line1Layout(l1) {
		return e, fmt.Errorf("%w: line 1 does not match the column layout %q", ErrMalformedTLE, tleLine1Layout)
	}
	if !line2Layout(l2) {
		return e, fmt.Errorf("%w: line 2 does not match the column layout %q", ErrMalformedTLE, tleLine2Layout)
	}
	// Pad so short optional trailing fields slice cleanly.
	l1 = padRight(l1, 69)
	l2 = padRight(l2, 69)

	p := fieldParser{}
	e.CatalogNumber = strings.TrimSpace(l1[2:7])
	e.SatNum = p.alpha5(1, 3, l1[2:7])
	e.Classification = l1[7]
	if e.Classification == ' ' {
		e.Classification = 'U'
	}
	e.IntlDesignator = strings.TrimSpace(l1[9:17])
	twoDigitYear := p.int(1, 19, l1[18:20], false)
	e.EpochDay = p.float(1, 21, l1[20:32])
	e.NDot = p.float(1, 34, l1[33:43])
	nddotMant := p.float(1, 45, l1[44:45]+"."+l1[45:50])
	nexp := p.int(1, 51, l1[50:52], false)
	bstarMant := p.float(1, 54, l1[53:54]+"."+l1[54:59])
	ibexp := p.int(1, 60, l1[59:61], false)
	e.EphemerisType = p.int(1, 63, l1[62:63], true)
	e.ElementSetNumber = p.int(1, 65, l1[64:68], true)

	if strings.TrimSpace(l2[2:7]) != e.CatalogNumber {
		return Elements{}, fmt.Errorf("%w: catalog numbers in lines 1 and 2 do not match (%q, %q)",
			ErrMalformedTLE, strings.TrimSpace(l1[2:7]), strings.TrimSpace(l2[2:7]))
	}
	e.Inclination = p.float(2, 9, l2[8:16])
	e.RAAN = p.float(2, 18, l2[17:25])
	e.Eccentricity = p.float(2, 27, "0."+strings.ReplaceAll(l2[26:33], " ", "0"))
	e.ArgPerigee = p.float(2, 35, l2[34:42])
	e.MeanAnomaly = p.float(2, 44, l2[43:51])
	e.MeanMotion = p.float(2, 53, l2[52:63])
	e.RevNumber = p.int(2, 64, l2[63:68], true)
	if p.err != nil {
		return Elements{}, p.err
	}

	e.NDDot = nddotMant * math.Pow(10.0, float64(nexp))
	e.BStar = bstarMant * math.Pow(10.0, float64(ibexp))

	// Two-digit years: 57-99 are 1957-1999, 00-56 are 2000-2056.
	year := twoDigitYear + 1900
	if twoDigitYear < 57 {
		year = twoDigitYear + 2000
	}
	e.EpochYear = year
	if e.EpochDay < 1.0 || e.EpochDay >= 367.0 {
		return Elements{}, fmt.Errorf("%w: line 1 epoch day %v out of range", ErrMalformedTLE, e.EpochDay)
	}

	// Split Julian date of the epoch, built straight from the year and day
	// to keep precision: the TLE gives the day fraction to 8 digits.
	days, fraction := math.Modf(e.EpochDay)
	jdEpoch := float64(year*365+(year-1)/4) + days + 1721044.5
	jdEpochF := math.Round(fraction*1e8) / 1e8

	// 1e-8 day is 864 microseconds, so the rounded fraction is a whole
	// number of nanoseconds.
	e.epoch = time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, int(days)-1).
		Add(time.Duration(math.Round(jdEpochF * 86400e9)))

	rec := &satrec{
		bstar:   e.BStar,
		ecco:    e.Eccentricity,
		argpo:   e.ArgPerigee * deg2rd,
		inclo:   e.Inclination * deg2rd,
		mo:      e.MeanAnomaly * deg2rd,
		noKozai: e.MeanMotion / xpdotp,
		nodeo:   e.RAAN * deg2rd,
	}
	sgp4init(rec, (jdEpoch+jdEpochF)-jd1950)
	e.rec = rec
	return e, nil
}

// line1Layout and line2Layout are the column checks of twoline2rv.
func line1Layout(l string) bool {
	return len(l) >= 64 && strings.HasPrefix(l, "1 ") &&
		l[8] == ' ' && l[23] == '.' && l[32] == ' ' && l[34] == '.' &&
		l[43] == ' ' && l[52] == ' ' && l[61] == ' ' && l[63] == ' '
}

func line2Layout(l string) bool {
	return len(l) >= 68 && strings.HasPrefix(l, "2 ") &&
		l[7] == ' ' && l[11] == '.' && l[16] == ' ' && l[20] == '.' &&
		l[25] == ' ' && l[33] == ' ' && l[37] == '.' && l[42] == ' ' &&
		l[46] == '.' && l[51] == ' '
}

const (
	tleLine1Layout = "1 NNNNNC NNNNNAAA NNNNN.NNNNNNNN +.NNNNNNNN +NNNNN-N +NNNNN-N N NNNNN"
	tleLine2Layout = "2 NNNNN NNN.NNNN NNN.NNNN NNNNNNN NNN.NNNN NNN.NNNN NN.NNNNNNNNNNNNNN"
)

// tleChecksum sums the digits of the first 68 columns, counting each minus
// sign as 1, modulo 10.
func tleChecksum(line string) int {
	sum := 0
	for i := 0; i < 68 && i < len(line); i++ {
		c := line[i]
		switch {
		case c >= '0' && c <= '9':
			sum += int(c - '0')
		case c == '-':
			sum++
		}
	}
	return sum % 10
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// fieldParser keeps the first field error so the parse reads straight
// through.
type fieldParser struct{ err error }

func (p *fieldParser) fail(line, col int, field, what string) {
	if p.err == nil {
		p.err = fmt.Errorf("%w: line %d column %d: %s %q", ErrMalformedTLE, line, col, what, field)
	}
}

func (p *fieldParser) float(line, col int, field string) float64 {
	s := strings.TrimSpace(field)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		p.fail(line, col, field, "bad number")
		return 0
	}
	return v
}

func (p *fieldParser) int(line, col int, field string, blankOK bool) int {
	s := strings.TrimSpace(field)
	if s == "" && blankOK {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		p.fail(line, col, field, "bad integer")
		return 0
	}
	return v
}

// alpha5 decodes a catalog number. Numbers from 100000 to 339999 are
// written as a letter followed by four digits, with A for 10 and the
// letters I and O skipped.
func (p *fieldParser) alpha5(line, col int, field string) int {
	s := strings.TrimSpace(field)
	if s == "" {
		p.fail(line, col, field, "blank catalog number")
		return 0
	}
	c := s[0]
	if c >= '0' && c <= '9' {
		return p.int(line, col, s, false)
	}
	if c < 'A' || c > 'Z' || c == 'I' || c == 'O' || len(s) != 5 {
		p.fail(line, col, field, "bad Alpha-5 catalog number")
		return 0
	}
	for i := 1; i < 5; i++ {
		if s[i] < '0' || s[i] > '9' {
			p.fail(line, col, field, "bad Alpha-5 catalog number")
			return 0
		}
	}
	n := int(c-'A') + 10
	if c > 'I' {
		n--
	}
	if c > 'O' {
		n--
	}
	rest, _ := strconv.Atoi(s[1:])
	return n*10000 + rest
}
