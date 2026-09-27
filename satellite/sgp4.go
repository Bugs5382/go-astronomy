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

// SGP4/SDP4 orbit propagation for two-line element sets.
//
// Ported from the public-domain reference implementation of Vallado,
// Crawford, Hujsak and Kelso (2006), by way of its Python transliteration
// in brandon-rhodes/python-sgp4 (MIT).
//
// Reference: D. A. Vallado, P. Crawford, R. Hujsak and T. S. Kelso,
// "Revisiting Spacetrack Report #3", AIAA 2006-6753.
//
// The port keeps the structure and the variable names of the reference
// code (sgp4init, initl, dscom, dpper, dsinit, dspace, sgp4 and gstime) so
// the two can be read side by side. Only the WGS-72 constants and the
// "improved" operation mode are carried over: TLEs are fitted against
// WGS-72, and the improved mode is what the published verification output
// (tcppver.out) was produced with.
//
// One deliberate difference: the reference keeps the resonance integrator
// state (atime, xli, xni) in the satellite record between calls as a speed
// optimisation. Here the propagator state is read-only after
// initialisation, so the integrator always restarts from epoch. The result
// is the same, and an Elements value can be shared between goroutines.

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

const (
	twoPi  = 2.0 * math.Pi
	deg2rd = math.Pi / 180.0
	x2o3   = 2.0 / 3.0

	// xpdotp converts revolutions per day to radians per minute.
	xpdotp = 1440.0 / (2.0 * math.Pi)

	// jd1950 is the Julian date of 0 January 1950, 0h, the SGP4 epoch
	// origin.
	jd1950 = 2433281.5
)

// WGS-72 gravity constants, as returned by getgravconst("wgs72").
var (
	wgs72Mu            = 398600.8 // km^3/s^2
	wgs72RadiusEarthKm = 6378.135 // km
	wgs72Xke           = 60.0 / math.Sqrt(wgs72RadiusEarthKm*wgs72RadiusEarthKm*wgs72RadiusEarthKm/wgs72Mu)
	wgs72J2            = 0.001082616
	wgs72J3            = -0.00000253881
	wgs72J4            = -0.00000165597
	wgs72J3oJ2         = wgs72J3 / wgs72J2
)

// Propagation errors. The reference code reports these as numeric codes on
// the satellite record; here each code has a sentinel error, and Propagate
// returns a *PropagationError that wraps it.
var (
	// ErrEccentricity is code 1: the mean eccentricity left the range
	// -0.001 <= e < 1.
	ErrEccentricity = errors.New("satellite: mean eccentricity out of range")
	// ErrMeanMotion is code 2: the mean motion dropped to zero or below.
	ErrMeanMotion = errors.New("satellite: mean motion is not positive")
	// ErrPerturbedEccentricity is code 3: the eccentricity after the
	// lunar-solar periodics left the range 0 <= e <= 1.
	ErrPerturbedEccentricity = errors.New("satellite: perturbed eccentricity out of range")
	// ErrSemiLatusRectum is code 4: the semi-latus rectum went negative.
	ErrSemiLatusRectum = errors.New("satellite: semi-latus rectum is negative")
	// ErrDecayed is code 6: the computed radius is below the Earth's
	// surface. Position and velocity are still returned with this error.
	ErrDecayed = errors.New("satellite: satellite has decayed")
	// ErrNotInitialised is returned when propagating a zero Elements value
	// that did not come from ParseTLE.
	ErrNotInitialised = errors.New("satellite: elements not initialised")
)

// Parse errors returned by ParseTLE, wrapped with details of the problem.
var (
	// ErrMalformedTLE means a line does not follow the fixed-column TLE
	// format.
	ErrMalformedTLE = errors.New("satellite: malformed TLE")
	// ErrChecksum means a line's modulo-10 checksum does not match.
	ErrChecksum = errors.New("satellite: TLE checksum mismatch")
)

// PropagationError describes a failed propagation. Code is the numeric
// error code of the reference implementation (1, 2, 3, 4 or 6).
type PropagationError struct {
	Code    int     // reference error code
	Minutes float64 // minutes since epoch of the failing request
	Value   float64 // the offending quantity (eccentricity, radius and so on)
	err     error
}

func (e *PropagationError) Error() string {
	var what string
	switch e.Code {
	case 1:
		what = fmt.Sprintf("mean eccentricity %f not within range 0.0 <= e < 1.0", e.Value)
	case 2:
		what = fmt.Sprintf("mean motion %f is less than zero", e.Value)
	case 3:
		what = fmt.Sprintf("perturbed eccentricity %f not within range 0.0 <= e <= 1.0", e.Value)
	case 4:
		what = fmt.Sprintf("semilatus rectum %f is less than zero", e.Value)
	case 6:
		what = fmt.Sprintf("mrt %f is less than 1.0 indicating the satellite has decayed", e.Value)
	default:
		what = fmt.Sprintf("error code %d", e.Code)
	}
	return fmt.Sprintf("satellite: %s (%.6f minutes from epoch)", what, e.Minutes)
}

// Unwrap returns the sentinel error for the code, so errors.Is works.
func (e *PropagationError) Unwrap() error { return e.err }

func propErr(code int, tsince, value float64) error {
	var sentinel error
	switch code {
	case 1:
		sentinel = ErrEccentricity
	case 2:
		sentinel = ErrMeanMotion
	case 3:
		sentinel = ErrPerturbedEccentricity
	case 4:
		sentinel = ErrSemiLatusRectum
	case 6:
		sentinel = ErrDecayed
	}
	return &PropagationError{Code: code, Minutes: tsince, Value: value, err: sentinel}
}

// Elements is a parsed two-line element set together with the initialised
// SGP4 propagator state. Angles are in degrees as printed in the TLE; the
// propagator works from its own radian copies.
type Elements struct {
	CatalogNumber    string  // catalog number as printed, possibly Alpha-5 ("A0001")
	SatNum           int     // numeric catalog number, Alpha-5 decoded (A0001 is 100001)
	Classification   byte    // 'U', 'C' or 'S'
	IntlDesignator   string  // international designator, such as "98067A"
	EpochYear        int     // four-digit epoch year (57-99 is 1900s, 00-56 is 2000s)
	EpochDay         float64 // day of year with fraction, 1.0 is 1 January 0h
	NDot             float64 // first derivative of mean motion / 2, rev/day^2
	NDDot            float64 // second derivative of mean motion / 6, rev/day^3
	BStar            float64 // drag term, 1/earth radii
	EphemerisType    int     // usually 0
	ElementSetNumber int
	Inclination      float64 // degrees
	RAAN             float64 // right ascension of the ascending node, degrees
	Eccentricity     float64
	ArgPerigee       float64 // argument of perigee, degrees
	MeanAnomaly      float64 // degrees
	MeanMotion       float64 // revolutions per day
	RevNumber        int     // revolution number at epoch

	epoch time.Time
	rec   *satrec // read-only after initialisation
}

// Epoch returns the element set epoch in UTC.
func (e Elements) Epoch() time.Time { return e.epoch }

// Age returns how far t is from the element set epoch. It is negative when
// t is before the epoch.
func (e Elements) Age(t time.Time) time.Duration { return t.Sub(e.epoch) }

// Propagate returns the TEME position (km) and velocity (km/s) at t.
func (e Elements) Propagate(t time.Time) (posKm, velKmS [3]float64, err error) {
	if e.rec == nil {
		return posKm, velKmS, ErrNotInitialised
	}
	// Whole seconds and nanoseconds separately, so very long spans do not
	// overflow time.Duration.
	sec := t.Unix() - e.epoch.Unix()
	nsec := int64(t.Nanosecond()) - int64(e.epoch.Nanosecond())
	tsince := float64(sec)/60.0 + float64(nsec)/6e10
	return e.PropagateMinutes(tsince)
}

// PropagateMinutes returns the TEME position (km) and velocity (km/s) at
// tsince minutes from the epoch. This is the form the verification data
// uses.
//
// Errors are go-apperr coded with astronomy.CodeSatellitePropagation and wrap
// a *PropagationError (errors.As) and its sentinel (errors.Is). For a decayed
// satellite the position and velocity are still returned with the error, as
// the reference code does.
func (e Elements) PropagateMinutes(tsince float64) (posKm, velKmS [3]float64, err error) {
	if e.rec == nil {
		return posKm, velKmS, apperr.Coded(astronomy.CodeSatellitePropagation, ErrNotInitialised)
	}
	posKm, velKmS, err = sgp4(e.rec, tsince)
	if err != nil {
		err = apperr.Coded(astronomy.CodeSatellitePropagation, err)
	}
	return posKm, velKmS, err
}

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

// satrec holds the initialised propagator state, mirroring the elsetrec
// structure of the reference code.
type satrec struct {
	// Mean elements at epoch, in radians and radians per minute.
	bstar, ecco, argpo, inclo, mo, noKozai, nodeo float64
	noUnkozai                                     float64

	// Near-earth coefficients.
	isimp                                   int
	method                                  byte // 'n' near earth, 'd' deep space
	aycof, con41, cc1, cc4, cc5, d2, d3, d4 float64
	delmo, eta, argpdot, omgcof, sinmao     float64
	t2cof, t3cof, t4cof, t5cof              float64
	x1mth2, x7thm1, mdot, nodedot, xlcof    float64
	xmcof, nodecf                           float64

	// Deep-space coefficients.
	irez                                           int
	d2201, d2211, d3210, d3222, d4410, d4422       float64
	d5220, d5232, d5421, d5433                     float64
	dedt, del1, del2, del3, didt, dmdt, dnodt      float64
	domdt, e3, ee2, peo, pgho, pho, pinco, plo     float64
	se2, se3, sgh2, sgh3, sgh4, sh2, sh3, si2, si3 float64
	sl2, sl3, sl4, gsto, xfact                     float64
	xgh2, xgh3, xgh4, xh2, xh3, xi2, xi3           float64
	xl2, xl3, xl4, xlamo, zmol, zmos, xli, xni     float64
}

// dpper applies the lunar-solar long-period periodics. By design they are
// zero at epoch. With init set it only evaluates them (the initial call);
// otherwise it applies them to the elements, using the Lyddane
// modification below 0.2 rad of inclination.
func dpper(r *satrec, t float64, init bool, ep, inclp, nodep, argpp, mp float64) (float64, float64, float64, float64, float64) {
	const (
		zns = 1.19459e-5
		zes = 0.01675
		znl = 1.5835218e-4
		zel = 0.05490
	)

	// Time-varying periodics. The initial call must use time zero.
	zm := r.zmos + zns*t
	if init {
		zm = r.zmos
	}
	zf := zm + 2.0*zes*math.Sin(zm)
	sinzf := math.Sin(zf)
	f2 := 0.5*sinzf*sinzf - 0.25
	f3 := -0.5 * sinzf * math.Cos(zf)
	ses := r.se2*f2 + r.se3*f3
	sis := r.si2*f2 + r.si3*f3
	sls := r.sl2*f2 + r.sl3*f3 + r.sl4*sinzf
	sghs := r.sgh2*f2 + r.sgh3*f3 + r.sgh4*sinzf
	shs := r.sh2*f2 + r.sh3*f3
	zm = r.zmol + znl*t
	if init {
		zm = r.zmol
	}
	zf = zm + 2.0*zel*math.Sin(zm)
	sinzf = math.Sin(zf)
	f2 = 0.5*sinzf*sinzf - 0.25
	f3 = -0.5 * sinzf * math.Cos(zf)
	sel := r.ee2*f2 + r.e3*f3
	sil := r.xi2*f2 + r.xi3*f3
	sll := r.xl2*f2 + r.xl3*f3 + r.xl4*sinzf
	sghl := r.xgh2*f2 + r.xgh3*f3 + r.xgh4*sinzf
	shll := r.xh2*f2 + r.xh3*f3
	pe := ses + sel
	pinc := sis + sil
	pl := sls + sll
	pgh := sghs + sghl
	ph := shs + shll

	if !init {
		pe -= r.peo
		pinc -= r.pinco
		pl -= r.plo
		pgh -= r.pgho
		ph -= r.pho
		inclp += pinc
		ep += pe
		sinip := math.Sin(inclp)
		cosip := math.Cos(inclp)

		// The GSFC form tests the perturbed inclination here; the
		// original STR#3 tested the epoch inclination.
		if inclp >= 0.2 {
			ph /= sinip
			pgh -= cosip * ph
			argpp += pgh
			nodep += ph
			mp += pl
		} else {
			// Lyddane modification.
			sinop := math.Sin(nodep)
			cosop := math.Cos(nodep)
			alfdp := sinip * sinop
			betdp := sinip * cosop
			dalf := ph*cosop + pinc*cosip*sinop
			dbet := -ph*sinop + pinc*cosip*cosop
			alfdp += dalf
			betdp += dbet
			nodep = math.Mod(nodep, twoPi)
			xls := mp + argpp + pl + pgh + (cosip-pinc*sinip)*nodep
			xnoh := nodep
			nodep = math.Atan2(alfdp, betdp)
			if math.Abs(xnoh-nodep) > math.Pi {
				if nodep < xnoh {
					nodep += twoPi
				} else {
					nodep -= twoPi
				}
			}
			mp += pl
			argpp = xls - mp - cosip*nodep
		}
	}
	return ep, inclp, nodep, argpp, mp
}

// dscomOut carries the dscom quantities that dsinit needs. The lunar-solar
// coefficients that dpper uses later go straight into the satrec.
type dscomOut struct {
	sinim, cosim, emsq, em, nm           float64
	s1, s2, s3, s4, s5                   float64
	ss1, ss2, ss3, ss4, ss5              float64
	sz1, sz3, sz11, sz13, sz21, sz23     float64
	sz31, sz33                           float64
	z1, z3, z11, z13, z21, z23, z31, z33 float64
}

// dscom computes the deep-space terms shared by the secular and periodic
// parts: solar and lunar geometry at the epoch.
func dscom(r *satrec, epoch, ep, argpp, tc, inclp, nodep, np float64) dscomOut {
	const (
		zes    = 0.01675
		zel    = 0.05490
		c1ss   = 2.9864797e-6
		c1l    = 4.7968065e-7
		zsinis = 0.39785416
		zcosis = 0.91744867
		zcosgs = 0.1945905
		zsings = -0.98088458
	)

	var o dscomOut
	nm := np
	em := ep
	snodm := math.Sin(nodep)
	cnodm := math.Cos(nodep)
	sinomm := math.Sin(argpp)
	cosomm := math.Cos(argpp)
	sinim := math.Sin(inclp)
	cosim := math.Cos(inclp)
	emsq := em * em
	betasq := 1.0 - emsq
	rtemsq := math.Sqrt(betasq)

	// Lunar-solar terms.
	r.peo = 0.0
	r.pinco = 0.0
	r.plo = 0.0
	r.pgho = 0.0
	r.pho = 0.0
	day := epoch + 18261.5 + tc/1440.0
	xnodce := math.Mod(4.5236020-9.2422029e-4*day, twoPi)
	stem := math.Sin(xnodce)
	ctem := math.Cos(xnodce)
	zcosil := 0.91375164 - 0.03568096*ctem
	zsinil := math.Sqrt(1.0 - zcosil*zcosil)
	zsinhl := 0.089683511 * stem / zsinil
	zcoshl := math.Sqrt(1.0 - zsinhl*zsinhl)
	gam := 5.8351514 + 0.0019443680*day
	zx := 0.39785416 * stem / zsinil
	zy := zcoshl*ctem + 0.91744867*zsinhl*stem
	zx = math.Atan2(zx, zy)
	zx = gam + zx - xnodce
	zcosgl := math.Cos(zx)
	zsingl := math.Sin(zx)

	// Solar terms first, then lunar on the second pass.
	zcosg := zcosgs
	zsing := zsings
	zcosi := zcosis
	zsini := zsinis
	zcosh := cnodm
	zsinh := snodm
	cc := c1ss
	xnoi := 1.0 / nm

	var s1, s2, s3, s4, s5, s6, s7 float64
	var ss1, ss2, ss3, ss4, ss5, ss6, ss7 float64
	var z1, z2, z3, z11, z12, z13, z21, z22, z23, z31, z32, z33 float64
	var sz1, sz2, sz3, sz11, sz12, sz13, sz21, sz22, sz23, sz31, sz32, sz33 float64

	for lsflg := 1; lsflg <= 2; lsflg++ {
		a1 := zcosg*zcosh + zsing*zcosi*zsinh
		a3 := -zsing*zcosh + zcosg*zcosi*zsinh
		a7 := -zcosg*zsinh + zsing*zcosi*zcosh
		a8 := zsing * zsini
		a9 := zsing*zsinh + zcosg*zcosi*zcosh
		a10 := zcosg * zsini
		a2 := cosim*a7 + sinim*a8
		a4 := cosim*a9 + sinim*a10
		a5 := -sinim*a7 + cosim*a8
		a6 := -sinim*a9 + cosim*a10

		x1 := a1*cosomm + a2*sinomm
		x2 := a3*cosomm + a4*sinomm
		x3 := -a1*sinomm + a2*cosomm
		x4 := -a3*sinomm + a4*cosomm
		x5 := a5 * sinomm
		x6 := a6 * sinomm
		x7 := a5 * cosomm
		x8 := a6 * cosomm

		z31 = 12.0*x1*x1 - 3.0*x3*x3
		z32 = 24.0*x1*x2 - 6.0*x3*x4
		z33 = 12.0*x2*x2 - 3.0*x4*x4
		z1 = 3.0*(a1*a1+a2*a2) + z31*emsq
		z2 = 6.0*(a1*a3+a2*a4) + z32*emsq
		z3 = 3.0*(a3*a3+a4*a4) + z33*emsq
		z11 = -6.0*a1*a5 + emsq*(-24.0*x1*x7-6.0*x3*x5)
		z12 = -6.0*(a1*a6+a3*a5) + emsq*
			(-24.0*(x2*x7+x1*x8)-6.0*(x3*x6+x4*x5))
		z13 = -6.0*a3*a6 + emsq*(-24.0*x2*x8-6.0*x4*x6)
		z21 = 6.0*a2*a5 + emsq*(24.0*x1*x5-6.0*x3*x7)
		z22 = 6.0*(a4*a5+a2*a6) + emsq*
			(24.0*(x2*x5+x1*x6)-6.0*(x4*x7+x3*x8))
		z23 = 6.0*a4*a6 + emsq*(24.0*x2*x6-6.0*x4*x8)
		z1 = z1 + z1 + betasq*z31
		z2 = z2 + z2 + betasq*z32
		z3 = z3 + z3 + betasq*z33
		s3 = cc * xnoi
		s2 = -0.5 * s3 / rtemsq
		s4 = s3 * rtemsq
		s1 = -15.0 * em * s4
		s5 = x1*x3 + x2*x4
		s6 = x2*x3 + x1*x4
		s7 = x2*x4 - x1*x3

		if lsflg == 1 {
			ss1 = s1
			ss2 = s2
			ss3 = s3
			ss4 = s4
			ss5 = s5
			ss6 = s6
			ss7 = s7
			sz1 = z1
			sz2 = z2
			sz3 = z3
			sz11 = z11
			sz12 = z12
			sz13 = z13
			sz21 = z21
			sz22 = z22
			sz23 = z23
			sz31 = z31
			sz32 = z32
			sz33 = z33
			zcosg = zcosgl
			zsing = zsingl
			zcosi = zcosil
			zsini = zsinil
			zcosh = zcoshl*cnodm + zsinhl*snodm
			zsinh = snodm*zcoshl - cnodm*zsinhl
			cc = c1l
		}
	}

	r.zmol = math.Mod(4.7199672+0.22997150*day-gam, twoPi)
	r.zmos = math.Mod(6.2565837+0.017201977*day, twoPi)

	// Solar coefficients.
	r.se2 = 2.0 * ss1 * ss6
	r.se3 = 2.0 * ss1 * ss7
	r.si2 = 2.0 * ss2 * sz12
	r.si3 = 2.0 * ss2 * (sz13 - sz11)
	r.sl2 = -2.0 * ss3 * sz2
	r.sl3 = -2.0 * ss3 * (sz3 - sz1)
	r.sl4 = -2.0 * ss3 * (-21.0 - 9.0*emsq) * zes
	r.sgh2 = 2.0 * ss4 * sz32
	r.sgh3 = 2.0 * ss4 * (sz33 - sz31)
	r.sgh4 = -18.0 * ss4 * zes
	r.sh2 = -2.0 * ss2 * sz22
	r.sh3 = -2.0 * ss2 * (sz23 - sz21)

	// Lunar coefficients.
	r.ee2 = 2.0 * s1 * s6
	r.e3 = 2.0 * s1 * s7
	r.xi2 = 2.0 * s2 * z12
	r.xi3 = 2.0 * s2 * (z13 - z11)
	r.xl2 = -2.0 * s3 * z2
	r.xl3 = -2.0 * s3 * (z3 - z1)
	r.xl4 = -2.0 * s3 * (-21.0 - 9.0*emsq) * zel
	r.xgh2 = 2.0 * s4 * z32
	r.xgh3 = 2.0 * s4 * (z33 - z31)
	r.xgh4 = -18.0 * s4 * zel
	r.xh2 = -2.0 * s2 * z22
	r.xh3 = -2.0 * s2 * (z23 - z21)

	o.sinim, o.cosim, o.emsq, o.em, o.nm = sinim, cosim, emsq, em, nm
	o.s1, o.s2, o.s3, o.s4, o.s5 = s1, s2, s3, s4, s5
	o.ss1, o.ss2, o.ss3, o.ss4, o.ss5 = ss1, ss2, ss3, ss4, ss5
	o.sz1, o.sz3, o.sz11, o.sz13, o.sz21, o.sz23 = sz1, sz3, sz11, sz13, sz21, sz23
	o.sz31, o.sz33 = sz31, sz33
	o.z1, o.z3, o.z11, o.z13, o.z21, o.z23, o.z31, o.z33 = z1, z3, z11, z13, z21, z23, z31, z33
	return o
}

// dsinit sets up the deep-space secular rates and, for half-day and
// one-day orbits, the geopotential resonance terms.
func dsinit(r *satrec, d dscomOut, t, tc, xpidot, eccsq, inclm float64) {
	const (
		q22    = 1.7891679e-6
		q31    = 2.1460748e-6
		q33    = 2.2123015e-7
		root22 = 1.7891679e-6
		root44 = 7.3636953e-9
		root54 = 2.1765803e-9
		rptim  = 4.37526908801129966e-3 // 7.29211514668855e-5 rad/s
		root32 = 3.7393792e-7
		root52 = 1.1428639e-7
		znl    = 1.5835218e-4
		zns    = 1.19459e-5
	)

	cosim, sinim, emsq := d.cosim, d.sinim, d.emsq
	em, nm := d.em, d.nm

	r.irez = 0
	if 0.0034906585 < nm && nm < 0.0052359877 {
		r.irez = 1
	}
	if 8.26e-3 <= nm && nm <= 9.24e-3 && em >= 0.5 {
		r.irez = 2
	}

	// Solar terms.
	ses := d.ss1 * zns * d.ss5
	sis := d.ss2 * zns * (d.sz11 + d.sz13)
	sls := -zns * d.ss3 * (d.sz1 + d.sz3 - 14.0 - 6.0*emsq)
	sghs := d.ss4 * zns * (d.sz31 + d.sz33 - 6.0)
	shs := -zns * d.ss2 * (d.sz21 + d.sz23)
	// Guard for inclinations near 0 and 180 degrees.
	if inclm < 5.2359877e-2 || inclm > math.Pi-5.2359877e-2 {
		shs = 0.0
	}
	if sinim != 0.0 {
		shs = shs / sinim
	}
	sgs := sghs - cosim*shs

	// Lunar terms.
	r.dedt = ses + d.s1*znl*d.s5
	r.didt = sis + d.s2*znl*(d.z11+d.z13)
	r.dmdt = sls - znl*d.s3*(d.z1+d.z3-14.0-6.0*emsq)
	sghl := d.s4 * znl * (d.z31 + d.z33 - 6.0)
	shll := -znl * d.s2 * (d.z21 + d.z23)
	if inclm < 5.2359877e-2 || inclm > math.Pi-5.2359877e-2 {
		shll = 0.0
	}
	r.domdt = sgs + sghl
	r.dnodt = shs
	if sinim != 0.0 {
		r.domdt = r.domdt - cosim/sinim*shll
		r.dnodt = r.dnodt + shll/sinim
	}

	// Resonance effects. The reference also applies the secular updates of
	// em, inclm and the rest with t here; they are zero because dsinit only
	// runs at epoch, and nothing below reads them, so they are left out.
	theta := math.Mod(r.gsto+tc*rptim, twoPi)

	if r.irez != 0 {
		aonv := math.Pow(nm/wgs72Xke, x2o3)

		// Geopotential resonance for 12-hour orbits.
		if r.irez == 2 {
			cosisq := cosim * cosim
			// The reference saves em and restores it afterwards; nothing
			// reads it after this block, so only emsq is restored.
			em = r.ecco
			emsqo := emsq
			emsq = eccsq
			eoc := em * emsq
			g201 := -0.306 - (em-0.64)*0.440

			var g211, g310, g322, g410, g422, g520, g521, g532, g533 float64
			if em <= 0.65 {
				g211 = 3.616 - 13.2470*em + 16.2900*emsq
				g310 = -19.302 + 117.3900*em - 228.4190*emsq + 156.5910*eoc
				g322 = -18.9068 + 109.7927*em - 214.6334*emsq + 146.5816*eoc
				g410 = -41.122 + 242.6940*em - 471.0940*emsq + 313.9530*eoc
				g422 = -146.407 + 841.8800*em - 1629.014*emsq + 1083.4350*eoc
				g520 = -532.114 + 3017.977*em - 5740.032*emsq + 3708.2760*eoc
			} else {
				g211 = -72.099 + 331.819*em - 508.738*emsq + 266.724*eoc
				g310 = -346.844 + 1582.851*em - 2415.925*emsq + 1246.113*eoc
				g322 = -342.585 + 1554.908*em - 2366.899*emsq + 1215.972*eoc
				g410 = -1052.797 + 4758.686*em - 7193.992*emsq + 3651.957*eoc
				g422 = -3581.690 + 16178.110*em - 24462.770*emsq + 12422.520*eoc
				if em > 0.715 {
					g520 = -5149.66 + 29936.92*em - 54087.36*emsq + 31324.56*eoc
				} else {
					g520 = 1464.74 - 4664.75*em + 3763.64*emsq
				}
			}
			if em < 0.7 {
				g533 = -919.22770 + 4988.6100*em - 9064.7700*emsq + 5542.21*eoc
				g521 = -822.71072 + 4568.6173*em - 8491.4146*emsq + 5337.524*eoc
				g532 = -853.66600 + 4690.2500*em - 8624.7700*emsq + 5341.4*eoc
			} else {
				g533 = -37995.780 + 161616.52*em - 229838.20*emsq + 109377.94*eoc
				g521 = -51752.104 + 218913.95*em - 309468.16*emsq + 146349.42*eoc
				g532 = -40023.880 + 170470.89*em - 242699.48*emsq + 115605.82*eoc
			}

			sini2 := sinim * sinim
			f220 := 0.75 * (1.0 + 2.0*cosim + cosisq)
			f221 := 1.5 * sini2
			f321 := 1.875 * sinim * (1.0 - 2.0*cosim - 3.0*cosisq)
			f322 := -1.875 * sinim * (1.0 + 2.0*cosim - 3.0*cosisq)
			f441 := 35.0 * sini2 * f220
			f442 := 39.3750 * sini2 * sini2
			f522 := 9.84375 * sinim * (sini2*(1.0-2.0*cosim-5.0*cosisq) +
				0.33333333*(-2.0+4.0*cosim+6.0*cosisq))
			f523 := sinim * (4.92187512*sini2*(-2.0-4.0*cosim+
				10.0*cosisq) + 6.56250012*(1.0+2.0*cosim-3.0*cosisq))
			f542 := 29.53125 * sinim * (2.0 - 8.0*cosim + cosisq*
				(-12.0+8.0*cosim+10.0*cosisq))
			f543 := 29.53125 * sinim * (-2.0 - 8.0*cosim + cosisq*
				(12.0+8.0*cosim-10.0*cosisq))
			xno2 := nm * nm
			ainv2 := aonv * aonv
			temp1 := 3.0 * xno2 * ainv2
			temp := temp1 * root22
			r.d2201 = temp * f220 * g201
			r.d2211 = temp * f221 * g211
			temp1 = temp1 * aonv
			temp = temp1 * root32
			r.d3210 = temp * f321 * g310
			r.d3222 = temp * f322 * g322
			temp1 = temp1 * aonv
			temp = 2.0 * temp1 * root44
			r.d4410 = temp * f441 * g410
			r.d4422 = temp * f442 * g422
			temp1 = temp1 * aonv
			temp = temp1 * root52
			r.d5220 = temp * f522 * g520
			r.d5232 = temp * f523 * g532
			temp = 2.0 * temp1 * root54
			r.d5421 = temp * f542 * g521
			r.d5433 = temp * f543 * g533
			r.xlamo = math.Mod(r.mo+r.nodeo+r.nodeo-theta-theta, twoPi)
			r.xfact = r.mdot + r.dmdt + 2.0*(r.nodedot+r.dnodt-rptim) - r.noUnkozai
			emsq = emsqo
		}

		// Synchronous resonance terms.
		if r.irez == 1 {
			g200 := 1.0 + emsq*(-2.5+0.8125*emsq)
			g310 := 1.0 + 2.0*emsq
			g300 := 1.0 + emsq*(-6.0+6.60937*emsq)
			f220 := 0.75 * (1.0 + cosim) * (1.0 + cosim)
			f311 := 0.9375*sinim*sinim*(1.0+3.0*cosim) - 0.75*(1.0+cosim)
			f330 := 1.0 + cosim
			f330 = 1.875 * f330 * f330 * f330
			r.del1 = 3.0 * nm * nm * aonv * aonv
			r.del2 = 2.0 * r.del1 * f220 * g200 * q22
			r.del3 = 3.0 * r.del1 * f330 * g300 * q33 * aonv
			r.del1 = r.del1 * f311 * g310 * q31 * aonv
			r.xlamo = math.Mod(r.mo+r.nodeo+r.argpo-theta, twoPi)
			r.xfact = r.mdot + xpidot - rptim + r.dmdt + r.domdt + r.dnodt - r.noUnkozai
		}

		// Starting state for the integrator.
		r.xli = r.xlamo
		r.xni = r.noUnkozai
	}
}

// dspace adds the deep-space secular effects and integrates the resonance
// terms (Euler-Maclaurin, 720-minute steps) from epoch to t.
func dspace(r *satrec, t, tc, em, argpm, inclm, mm, nodem, nm float64) (float64, float64, float64, float64, float64, float64) {
	const (
		fasx2 = 0.13130908
		fasx4 = 2.8843198
		fasx6 = 0.37448087
		g22   = 5.7686396
		g32   = 0.95240898
		g44   = 1.8014998
		g52   = 1.0508330
		g54   = 4.4108898
		rptim = 4.37526908801129966e-3 // 7.29211514668855e-5 rad/s
		stepp = 720.0
		stepn = -720.0
		step2 = 259200.0
	)

	theta := math.Mod(r.gsto+tc*rptim, twoPi)
	em = em + r.dedt*t
	inclm = inclm + r.didt*t
	argpm = argpm + r.domdt*t
	nodem = nodem + r.dnodt*t
	mm = mm + r.dmdt*t

	if r.irez == 0 {
		return em, argpm, inclm, mm, nodem, nm
	}

	// The reference caches atime, xli and xni between calls; starting
	// from epoch every time gives the same answer.
	atime := 0.0
	xni := r.noUnkozai
	xli := r.xlamo

	delt := stepn
	if t > 0.0 {
		delt = stepp
	}

	var xndt, xldot, xnddt, ft float64
	for {
		if r.irez != 2 {
			// Near-synchronous resonance terms.
			xndt = r.del1*math.Sin(xli-fasx2) + r.del2*math.Sin(2.0*(xli-fasx4)) +
				r.del3*math.Sin(3.0*(xli-fasx6))
			xldot = xni + r.xfact
			xnddt = r.del1*math.Cos(xli-fasx2) +
				2.0*r.del2*math.Cos(2.0*(xli-fasx4)) +
				3.0*r.del3*math.Cos(3.0*(xli-fasx6))
			xnddt = xnddt * xldot
		} else {
			// Near half-day resonance terms.
			xomi := r.argpo + r.argpdot*atime
			x2omi := xomi + xomi
			x2li := xli + xli
			xndt = r.d2201*math.Sin(x2omi+xli-g22) + r.d2211*math.Sin(xli-g22) +
				r.d3210*math.Sin(xomi+xli-g32) + r.d3222*math.Sin(-xomi+xli-g32) +
				r.d4410*math.Sin(x2omi+x2li-g44) + r.d4422*math.Sin(x2li-g44) +
				r.d5220*math.Sin(xomi+xli-g52) + r.d5232*math.Sin(-xomi+xli-g52) +
				r.d5421*math.Sin(xomi+x2li-g54) + r.d5433*math.Sin(-xomi+x2li-g54)
			xldot = xni + r.xfact
			xnddt = r.d2201*math.Cos(x2omi+xli-g22) + r.d2211*math.Cos(xli-g22) +
				r.d3210*math.Cos(xomi+xli-g32) + r.d3222*math.Cos(-xomi+xli-g32) +
				r.d5220*math.Cos(xomi+xli-g52) + r.d5232*math.Cos(-xomi+xli-g52) +
				2.0*(r.d4410*math.Cos(x2omi+x2li-g44)+
					r.d4422*math.Cos(x2li-g44)+r.d5421*math.Cos(xomi+x2li-g54)+
					r.d5433*math.Cos(-xomi+x2li-g54))
			xnddt = xnddt * xldot
		}

		if math.Abs(t-atime) < stepp {
			ft = t - atime
			break
		}
		xli = xli + xldot*delt + xndt*step2
		xni = xni + xndt*delt + xnddt*step2
		atime = atime + delt
	}

	nm = xni + xndt*ft + xnddt*ft*ft*0.5
	xl := xli + xldot*ft + xndt*ft*ft*0.5
	var dndt float64
	if r.irez != 1 {
		mm = xl - 2.0*nodem + 2.0*theta
		dndt = nm - r.noUnkozai
	} else {
		mm = xl - nodem - argpm + theta
		dndt = nm - r.noUnkozai
	}
	nm = r.noUnkozai + dndt
	return em, argpm, inclm, mm, nodem, nm
}

// initl un-Kozais the mean motion and computes the epoch quantities shared
// by the near-earth and deep-space set-up.
type initlOut struct {
	ainv, ao, con42, cosio, cosio2, eccsq, omeosq, posq, rp, rteosq, sinio float64
}

func initl(r *satrec, epoch float64) initlOut {
	var o initlOut
	ecco := r.ecco

	o.eccsq = ecco * ecco
	o.omeosq = 1.0 - o.eccsq
	o.rteosq = math.Sqrt(o.omeosq)
	o.cosio = math.Cos(r.inclo)
	o.cosio2 = o.cosio * o.cosio

	// Un-Kozai the mean motion.
	ak := math.Pow(wgs72Xke/r.noKozai, x2o3)
	d1 := 0.75 * wgs72J2 * (3.0*o.cosio2 - 1.0) / (o.rteosq * o.omeosq)
	del := d1 / (ak * ak)
	adel := ak * (1.0 - del*del - del*
		(1.0/3.0+134.0*del*del/81.0))
	del = d1 / (adel * adel)
	r.noUnkozai = r.noKozai / (1.0 + del)

	o.ao = math.Pow(wgs72Xke/r.noUnkozai, x2o3)
	o.sinio = math.Sin(r.inclo)
	po := o.ao * o.omeosq
	o.con42 = 1.0 - 5.0*o.cosio2
	r.con41 = -o.con42 - o.cosio2 - o.cosio2
	o.ainv = 1.0 / o.ao
	o.posq = po * po
	o.rp = o.ao * (1.0 - ecco)
	r.method = 'n'

	r.gsto = gmst82(epoch + jd1950)
	return o
}

// sgp4init fills in the propagator state from the mean elements already set
// on r. epoch is in days since 0 January 1950, 0h UTC.
func sgp4init(r *satrec, epoch float64) {
	const temp4 = 1.5e-12

	ss := 78.0/wgs72RadiusEarthKm + 1.0
	qzms2ttemp := (120.0 - 78.0) / wgs72RadiusEarthKm
	qzms2t := qzms2ttemp * qzms2ttemp * qzms2ttemp * qzms2ttemp

	r.method = 'n'
	in := initl(r, epoch)
	ao, cosio, cosio2, sinio := in.ao, in.cosio, in.cosio2, in.sinio
	omeosq, rteosq, con42 := in.omeosq, in.rteosq, in.con42

	// The reference dropped its sub-orbital (rp < 1) check: the radius test
	// in sgp4 catches decaying objects even when they start below the
	// surface.
	if omeosq >= 0.0 || r.noUnkozai >= 0.0 {
		r.isimp = 0
		if in.rp < 220.0/wgs72RadiusEarthKm+1.0 {
			r.isimp = 1
		}
		sfour := ss
		qzms24 := qzms2t
		perige := (in.rp - 1.0) * wgs72RadiusEarthKm

		// For perigees below 156 km, s and qoms2t are altered.
		if perige < 156.0 {
			sfour = perige - 78.0
			if perige < 98.0 {
				sfour = 20.0
			}
			qzms24temp := (120.0 - sfour) / wgs72RadiusEarthKm
			qzms24 = qzms24temp * qzms24temp * qzms24temp * qzms24temp
			sfour = sfour/wgs72RadiusEarthKm + 1.0
		}
		pinvsq := 1.0 / in.posq

		tsi := 1.0 / (ao - sfour)
		r.eta = ao * r.ecco * tsi
		etasq := r.eta * r.eta
		eeta := r.ecco * r.eta
		psisq := math.Abs(1.0 - etasq)
		coef := qzms24 * math.Pow(tsi, 4.0)
		coef1 := coef / math.Pow(psisq, 3.5)
		cc2 := coef1 * r.noUnkozai * (ao*(1.0+1.5*etasq+eeta*
			(4.0+etasq)) + 0.375*wgs72J2*tsi/psisq*r.con41*
			(8.0+3.0*etasq*(8.0+etasq)))
		r.cc1 = r.bstar * cc2
		cc3 := 0.0
		if r.ecco > 1.0e-4 {
			cc3 = -2.0 * coef * tsi * wgs72J3oJ2 * r.noUnkozai * sinio / r.ecco
		}
		r.x1mth2 = 1.0 - cosio2
		r.cc4 = 2.0 * r.noUnkozai * coef1 * ao * omeosq *
			(r.eta*(2.0+0.5*etasq) + r.ecco*
				(0.5+2.0*etasq) - wgs72J2*tsi/(ao*psisq)*
				(-3.0*r.con41*(1.0-2.0*eeta+etasq*
					(1.5-0.5*eeta))+0.75*r.x1mth2*
					(2.0*etasq-eeta*(1.0+etasq))*math.Cos(2.0*r.argpo)))
		r.cc5 = 2.0 * coef1 * ao * omeosq * (1.0 + 2.75*
			(etasq+eeta) + eeta*etasq)
		cosio4 := cosio2 * cosio2
		temp1 := 1.5 * wgs72J2 * pinvsq * r.noUnkozai
		temp2 := 0.5 * temp1 * wgs72J2 * pinvsq
		temp3 := -0.46875 * wgs72J4 * pinvsq * pinvsq * r.noUnkozai
		r.mdot = r.noUnkozai + 0.5*temp1*rteosq*r.con41 + 0.0625*
			temp2*rteosq*(13.0-78.0*cosio2+137.0*cosio4)
		r.argpdot = (-0.5*temp1*con42 + 0.0625*temp2*
			(7.0-114.0*cosio2+395.0*cosio4) +
			temp3*(3.0-36.0*cosio2+49.0*cosio4))
		xhdot1 := -temp1 * cosio
		r.nodedot = xhdot1 + (0.5*temp2*(4.0-19.0*cosio2)+
			2.0*temp3*(3.0-7.0*cosio2))*cosio
		xpidot := r.argpdot + r.nodedot
		r.omgcof = r.bstar * cc3 * math.Cos(r.argpo)
		r.xmcof = 0.0
		if r.ecco > 1.0e-4 {
			r.xmcof = -x2o3 * coef * r.bstar / eeta
		}
		r.nodecf = 3.5 * omeosq * xhdot1 * r.cc1
		r.t2cof = 1.5 * r.cc1
		// Guard against dividing by zero at 180 degrees inclination.
		if math.Abs(cosio+1.0) > 1.5e-12 {
			r.xlcof = -0.25 * wgs72J3oJ2 * sinio * (3.0 + 5.0*cosio) / (1.0 + cosio)
		} else {
			r.xlcof = -0.25 * wgs72J3oJ2 * sinio * (3.0 + 5.0*cosio) / temp4
		}
		r.aycof = -0.5 * wgs72J3oJ2 * sinio
		delmotemp := 1.0 + r.eta*math.Cos(r.mo)
		r.delmo = delmotemp * delmotemp * delmotemp
		r.sinmao = math.Sin(r.mo)
		r.x7thm1 = 7.0*cosio2 - 1.0

		// Deep-space initialisation for periods of 225 minutes or more.
		if 2*math.Pi/r.noUnkozai >= 225.0 {
			r.method = 'd'
			r.isimp = 1
			tc := 0.0
			inclm := r.inclo

			d := dscom(r, epoch, r.ecco, r.argpo, tc, r.inclo, r.nodeo, r.noUnkozai)
			r.ecco, r.inclo, r.nodeo, r.argpo, r.mo = dpper(r, 0.0, true,
				r.ecco, r.inclo, r.nodeo, r.argpo, r.mo)
			dsinit(r, d, 0.0, tc, xpidot, in.eccsq, inclm)
		}

		// Remaining near-earth drag terms.
		if r.isimp != 1 {
			cc1sq := r.cc1 * r.cc1
			r.d2 = 4.0 * ao * tsi * cc1sq
			temp := r.d2 * tsi * r.cc1 / 3.0
			r.d3 = (17.0*ao + sfour) * temp
			r.d4 = 0.5 * temp * ao * tsi * (221.0*ao + 31.0*sfour) *
				r.cc1
			r.t3cof = r.d2 + 2.0*cc1sq
			r.t4cof = 0.25 * (3.0*r.d3 + r.cc1*
				(12.0*r.d2+10.0*cc1sq))
			r.t5cof = 0.2 * (3.0*r.d4 +
				12.0*r.cc1*r.d3 +
				6.0*r.d2*r.d2 +
				15.0*cc1sq*(2.0*r.d2+cc1sq))
		}
	}
	// The reference finishes with a call to sgp4 at tsince zero only to set
	// its error flag. Nothing here depends on that call, so it is left to
	// Propagate to report problems.
}

// sgp4 is the prediction model: secular gravity and drag, the deep-space
// terms where they apply, Kepler's equation, then the short-period
// periodics. It returns TEME position (km) and velocity (km/s).
func sgp4(r *satrec, tsince float64) (pos, vel [3]float64, err error) {
	const temp4 = 1.5e-12
	vkmpersec := wgs72RadiusEarthKm * wgs72Xke / 60.0

	t := tsince

	// Secular gravity and atmospheric drag.
	xmdf := r.mo + r.mdot*t
	argpdf := r.argpo + r.argpdot*t
	nodedf := r.nodeo + r.nodedot*t
	argpm := argpdf
	mm := xmdf
	t2 := t * t
	nodem := nodedf + r.nodecf*t2
	tempa := 1.0 - r.cc1*t
	tempe := r.bstar * r.cc4 * t
	templ := r.t2cof * t2

	if r.isimp != 1 {
		delomg := r.omgcof * t
		delmtemp := 1.0 + r.eta*math.Cos(xmdf)
		delm := r.xmcof *
			(delmtemp*delmtemp*delmtemp -
				r.delmo)
		temp := delomg + delm
		mm = xmdf + temp
		argpm = argpdf - temp
		t3 := t2 * t
		t4 := t3 * t
		tempa = tempa - r.d2*t2 - r.d3*t3 -
			r.d4*t4
		tempe = tempe + r.bstar*r.cc5*(math.Sin(mm)-
			r.sinmao)
		templ = templ + r.t3cof*t3 + t4*(r.t4cof+
			t*r.t5cof)
	}

	nm := r.noUnkozai
	em := r.ecco
	inclm := r.inclo
	if r.method == 'd' {
		tc := t
		em, argpm, inclm, mm, nodem, nm = dspace(r, t, tc, em, argpm, inclm, mm, nodem, nm)
	}

	if nm <= 0.0 {
		return pos, vel, propErr(2, tsince, nm)
	}

	am := math.Pow(wgs72Xke/nm, x2o3) * tempa * tempa
	nm = wgs72Xke / math.Pow(am, 1.5)
	em = em - tempe

	if em >= 1.0 || em < -0.001 {
		return pos, vel, propErr(1, tsince, em)
	}

	// Avoid a divide by zero for circular orbits.
	if em < 1.0e-6 {
		em = 1.0e-6
	}
	mm = mm + r.noUnkozai*templ
	xlm := mm + argpm + nodem

	nodem = math.Mod(nodem, twoPi)
	argpm = math.Mod(argpm, twoPi)
	xlm = math.Mod(xlm, twoPi)
	mm = math.Mod(xlm-argpm-nodem, twoPi)

	sinim := math.Sin(inclm)
	cosim := math.Cos(inclm)

	// Lunar-solar periodics.
	ep := em
	xincp := inclm
	argpp := argpm
	nodep := nodem
	mp := mm
	sinip := sinim
	cosip := cosim
	aycof, xlcof := r.aycof, r.xlcof
	con41, x1mth2, x7thm1 := r.con41, r.x1mth2, r.x7thm1
	if r.method == 'd' {
		ep, xincp, nodep, argpp, mp = dpper(r, t, false, ep, xincp, nodep, argpp, mp)
		if xincp < 0.0 {
			xincp = -xincp
			nodep = nodep + math.Pi
			argpp = argpp - math.Pi
		}
		if ep < 0.0 || ep > 1.0 {
			return pos, vel, propErr(3, tsince, ep)
		}

		// Long-period periodics with the perturbed inclination.
		sinip = math.Sin(xincp)
		cosip = math.Cos(xincp)
		aycof = -0.5 * wgs72J3oJ2 * sinip
		if math.Abs(cosip+1.0) > 1.5e-12 {
			xlcof = -0.25 * wgs72J3oJ2 * sinip * (3.0 + 5.0*cosip) / (1.0 + cosip)
		} else {
			xlcof = -0.25 * wgs72J3oJ2 * sinip * (3.0 + 5.0*cosip) / temp4
		}
	}

	axnl := ep * math.Cos(argpp)
	temp := 1.0 / (am * (1.0 - ep*ep))
	aynl := ep*math.Sin(argpp) + temp*aycof
	xl := mp + argpp + nodep + temp*xlcof*axnl

	// Kepler's equation, with the step limited to 0.95 rad.
	u := math.Mod(xl-nodep, twoPi)
	eo1 := u
	tem5 := 9999.9
	var sineo1, coseo1 float64
	for ktr := 1; math.Abs(tem5) >= 1.0e-12 && ktr <= 10; ktr++ {
		sineo1 = math.Sin(eo1)
		coseo1 = math.Cos(eo1)
		tem5 = 1.0 - coseo1*axnl - sineo1*aynl
		tem5 = (u - aynl*coseo1 + axnl*sineo1 - eo1) / tem5
		if math.Abs(tem5) >= 0.95 {
			if tem5 > 0.0 {
				tem5 = 0.95
			} else {
				tem5 = -0.95
			}
		}
		eo1 = eo1 + tem5
	}

	// Short-period preliminary quantities.
	ecose := axnl*coseo1 + aynl*sineo1
	esine := axnl*sineo1 - aynl*coseo1
	el2 := axnl*axnl + aynl*aynl
	pl := am * (1.0 - el2)
	if pl < 0.0 {
		return pos, vel, propErr(4, tsince, pl)
	}

	rl := am * (1.0 - ecose)
	rdotl := math.Sqrt(am) * esine / rl
	rvdotl := math.Sqrt(pl) / rl
	betal := math.Sqrt(1.0 - el2)
	temp = esine / (1.0 + betal)
	sinu := am / rl * (sineo1 - aynl - axnl*temp)
	cosu := am / rl * (coseo1 - axnl + aynl*temp)
	su := math.Atan2(sinu, cosu)
	sin2u := (cosu + cosu) * sinu
	cos2u := 1.0 - 2.0*sinu*sinu
	temp = 1.0 / pl
	temp1 := 0.5 * wgs72J2 * temp
	temp2 := temp1 * temp

	// Short-period periodics.
	if r.method == 'd' {
		cosisq := cosip * cosip
		con41 = 3.0*cosisq - 1.0
		x1mth2 = 1.0 - cosisq
		x7thm1 = 7.0*cosisq - 1.0
	}

	mrt := rl*(1.0-1.5*temp2*betal*con41) +
		0.5*temp1*x1mth2*cos2u
	su = su - 0.25*temp2*x7thm1*sin2u
	xnode := nodep + 1.5*temp2*cosip*sin2u
	xinc := xincp + 1.5*temp2*cosip*sinip*cos2u
	mvt := rdotl - nm*temp1*x1mth2*sin2u/wgs72Xke
	rvdot := rvdotl + nm*temp1*(x1mth2*cos2u+
		1.5*con41)/wgs72Xke

	// Orientation vectors.
	sinsu := math.Sin(su)
	cossu := math.Cos(su)
	snod := math.Sin(xnode)
	cnod := math.Cos(xnode)
	sini := math.Sin(xinc)
	cosi := math.Cos(xinc)
	xmx := -snod * cosi
	xmy := cnod * cosi
	ux := xmx*sinsu + cnod*cossu
	uy := xmy*sinsu + snod*cossu
	uz := sini * sinsu
	vx := xmx*cossu - cnod*sinsu
	vy := xmy*cossu - snod*sinsu
	vz := sini * cossu

	// Position and velocity in km and km/s.
	mr := mrt * wgs72RadiusEarthKm
	pos = [3]float64{mr * ux, mr * uy, mr * uz}
	vel = [3]float64{
		(mvt*ux + rvdot*vx) * vkmpersec,
		(mvt*uy + rvdot*vy) * vkmpersec,
		(mvt*uz + rvdot*vz) * vkmpersec,
	}

	// A radius below one earth radius means the object has come down. The
	// vectors are still returned, as the reference does.
	if mrt < 1.0 {
		return pos, vel, propErr(6, tsince, mrt)
	}
	return pos, vel, nil
}

// gmst82 returns Greenwich mean sidereal time in radians, 0 to 2 pi, from
// the IAU 1982 expression (Vallado 2004, eq. 3-45). SGP4 defines TEME
// against this angle, so it is the one to use when rotating TEME to an
// Earth-fixed frame.
func gmst82(jdUT1 float64) float64 {
	tut1 := (jdUT1 - 2451545.0) / 36525.0
	temp := -6.2e-6*tut1*tut1*tut1 + 0.093104*tut1*tut1 +
		(876600.0*3600+8640184.812866)*tut1 + 67310.54841 // seconds
	temp = math.Mod(temp*deg2rd/240.0, twoPi) // 360/86400 = 1/240, to degrees, to radians
	if temp < 0.0 {
		temp += twoPi
	}
	return temp
}
