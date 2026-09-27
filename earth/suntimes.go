package earth

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
	"sort"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// coarseStep is the sampling interval used to bracket altitude-threshold
// crossings across the day. The Sun's altitude changes by at most about a
// quarter degree per minute, so a one-minute step resolves every default band,
// the narrowest of which spans roughly two minutes near the horizon.
const coarseStep = time.Minute

// Sentinel causes wrapped by the constructors. Each is returned inside a
// go-apperr coded error, so errors.Is keeps matching these values while
// apperr.Code recovers the stable numeric code (see the astronomy package's code
// registry). Match a condition with errors.Is(err, ErrInvalid...) or branch on
// the code with apperr.Code(err).
var (
	// ErrInvalidLatitude is the cause when the observer's latitude is outside
	// the physical range [-90, 90]; the coded error carries
	// astronomy.CodeInvalidLatitude.
	ErrInvalidLatitude = errors.New("earth: observer latitude out of range")
	// ErrInvalidLongitude is the cause when the observer's longitude is outside
	// the range [-180, 180]; the coded error carries
	// astronomy.CodeInvalidLongitude.
	ErrInvalidLongitude = errors.New("earth: observer longitude out of range")
	// ErrInvalidSegmentation is the cause when a Segmentation has no levels or
	// its levels are not in strictly ascending altitude order; the coded error
	// carries astronomy.CodeInvalidSegmentation.
	ErrInvalidSegmentation = errors.New("earth: segmentation levels must be non-empty and strictly ascending")
)

// validateObserver reports the coded error for an out-of-range observer, or nil
// when the latitude and longitude are both physical. It centralizes the input
// contract shared by the day constructors and the instant resolver.
func validateObserver(obs astronomy.Observer) error {
	if obs.Lat < -90 || obs.Lat > 90 {
		return apperr.Coded(astronomy.CodeInvalidLatitude, ErrInvalidLatitude)
	}
	if obs.Lng < -180 || obs.Lng > 180 {
		return apperr.Coded(astronomy.CodeInvalidLongitude, ErrInvalidLongitude)
	}
	// A NaN or infinite height; any real height is used as given.
	return obs.Height.Err()
}

// SunTimes is the Sun's civil day for one observer: the ordered twilight and
// daylight bands, solar noon, and any polar state. It is a snapshot of a single
// resolved day and holds no live clock; call the package functions again for
// another instant.
type SunTimes struct {
	segments []Segment
	noon     time.Time
	polar    PolarState
	start    time.Time
	end      time.Time
}

// NewSunTimes resolves the observer's civil day containing date, in obs.TZ,
// using the Earth DefaultSegmentation. The day runs from local midnight to the
// following local midnight and honors daylight-saving transitions, so its length
// is 23 or 25 hours on such days. Only the year, month, and day of date are
// used; its time of day is ignored.
//
// It returns a go-apperr coded error when the observer's latitude or longitude
// is out of range: match the condition with errors.Is against ErrInvalidLatitude
// or ErrInvalidLongitude, or recover the code with apperr.Code (see the
// astronomy package's code registry).
func NewSunTimes(obs astronomy.Observer, date time.Time) (*SunTimes, error) {
	return NewSunTimesWith(obs, date, DefaultSegmentation)
}

// NewSunTimesWith is NewSunTimes with a caller-supplied Segmentation, letting a
// consumer redefine the thresholds and labels wholesale. For an observer above
// sea level the segmentation's horizon and its dip-corrected levels are
// lowered by the horizon dip first; the result must still be strictly
// ascending.
func NewSunTimesWith(obs astronomy.Observer, date time.Time, seg Segmentation) (*SunTimes, error) {
	if err := validateObserver(obs); err != nil {
		return nil, err
	}
	seg = seg.atHeight(obs.Height)
	if err := validateSegmentation(seg); err != nil {
		return nil, err
	}
	return build(obs, date, seg, sunAltitudes(obs)), nil
}

// validateSegmentation checks that the segmentation is usable: at least one
// level, ascending altitudes.
func validateSegmentation(seg Segmentation) error {
	if len(seg.Levels) == 0 {
		return apperr.Coded(astronomy.CodeInvalidSegmentation, ErrInvalidSegmentation)
	}
	for i := 1; i < len(seg.Levels); i++ {
		if seg.Levels[i].Altitude <= seg.Levels[i-1].Altitude {
			return apperr.Coded(astronomy.CodeInvalidSegmentation, ErrInvalidSegmentation)
		}
	}
	return nil
}

// Segments returns the day's bands in chronological order. The slice is a copy;
// mutating it does not affect the SunTimes.
func (s *SunTimes) Segments() []Segment {
	out := make([]Segment, len(s.segments))
	copy(out, s.segments)
	return out
}

// SolarNoon returns the instant of the Sun's upper culmination, when its
// altitude is greatest, and true. The culmination always exists, so ok is
// always true; the pairing keeps the accessor uniform with Polar.
func (s *SunTimes) SolarNoon() (time.Time, bool) {
	return s.noon, true
}

// Polar returns the day's polar state and whether it is polar. When the Sun
// crosses the horizon normally the state is NotPolar and ok is false; on a day
// with no horizon crossing the state is MidnightSun or PolarNight and ok is
// true.
func (s *SunTimes) Polar() (PolarState, bool) {
	return s.polar, s.polar != NotPolar
}

// DayStart returns the local midnight that opens the civil day.
func (s *SunTimes) DayStart() time.Time { return s.start }

// DayEnd returns the following local midnight that closes the civil day.
func (s *SunTimes) DayEnd() time.Time { return s.end }

// DayLength returns the civil day's duration, which is 23 or 25 hours on a
// daylight-saving transition day and 24 hours otherwise.
func (s *SunTimes) DayLength() time.Duration { return s.end.Sub(s.start) }

// build resolves the whole civil day: the band boundaries, solar noon, and the
// polar state, with altAt the Sun's altitude for the observer.
func build(obs astronomy.Observer, date time.Time, seg Segmentation, altAt func(time.Time) float64) *SunTimes {
	loc := obs.Location()
	local := date.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)

	noon := solarNoon(altAt, start, end)
	minAlt := dayMinAltitude(altAt, start, end)
	maxAlt := altAt(noon)

	polar := NotPolar
	switch {
	case minAlt > seg.Horizon:
		polar = MidnightSun
	case maxAlt < seg.Horizon:
		polar = PolarNight
	}

	segments := segmentDay(altAt, seg, start, end, noon)

	return &SunTimes{
		segments: segments,
		noon:     noon,
		polar:    polar,
		start:    start,
		end:      end,
	}
}

// segmentDay locates every altitude-threshold crossing across the day, adds the
// day boundaries, solar noon, and any lower culmination (solar midnight) as
// splits, and labels each resulting interval by the altitude zone at its
// midpoint and whether the Sun is climbing or sinking there.
//
// Splitting at both culminations means no interval contains an altitude
// extremum, so the trend at the midpoint holds for the whole interval. A band
// that runs through local midnight therefore keeps one label on both sides of
// the date boundary (issue 50). A lower culmination below the lowest level sits
// inside the night band, whose label does not depend on the trend, so it is not
// added as a split there.
func segmentDay(altAt func(time.Time) float64, seg Segmentation, start, end, noon time.Time) []Segment {
	boundaries := []time.Time{start, end, noon}
	boundaries = append(boundaries, crossings(altAt, seg, start, end)...)
	for _, low := range lowerCulminations(altAt, start, end) {
		if altAt(low) >= seg.Levels[0].Altitude {
			boundaries = append(boundaries, low)
		}
	}

	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].Before(boundaries[j]) })
	boundaries = dedupTimes(boundaries, start, end)

	segments := make([]Segment, 0, len(boundaries)-1)
	for i := 0; i+1 < len(boundaries); i++ {
		from, to := boundaries[i], boundaries[i+1]
		mid := from.Add(to.Sub(from) / 2)
		label := zoneLabel(seg, altAt(mid), climbing(altAt, mid))
		segments = append(segments, Segment{
			Label:   label,
			From:    from,
			To:      to,
			Seconds: to.Sub(from).Seconds(),
		})
	}
	return segments
}

// crossings returns every instant, refined to sub-second precision, at which the
// Sun's altitude crosses one of the segmentation's level altitudes during the
// day.
func crossings(altAt func(time.Time) float64, seg Segmentation, start, end time.Time) []time.Time {
	var samples []struct {
		t time.Time
		a float64
	}
	for t := start; t.Before(end); t = t.Add(coarseStep) {
		samples = append(samples, struct {
			t time.Time
			a float64
		}{t, altAt(t)})
	}
	samples = append(samples, struct {
		t time.Time
		a float64
	}{end, altAt(end)})

	var out []time.Time
	for i := 0; i+1 < len(samples); i++ {
		lo, hi := samples[i], samples[i+1]
		for _, lvl := range seg.Levels {
			d0, d1 := lo.a-lvl.Altitude, hi.a-lvl.Altitude
			if d0 == 0 {
				out = append(out, lo.t)
				continue
			}
			if (d0 < 0) != (d1 < 0) {
				out = append(out, bisect(altAt, lo.t, hi.t, lvl.Altitude))
			}
		}
	}
	return out
}

// bisect refines a crossing of the threshold thr in [lo, hi], where altAt-thr
// changes sign across the bracket, to sub-second precision.
func bisect(altAt func(time.Time) float64, lo, hi time.Time, thr float64) time.Time {
	dLo := altAt(lo) - thr
	for i := 0; i < 60 && hi.Sub(lo) > 200*time.Millisecond; i++ {
		mid := lo.Add(hi.Sub(lo) / 2)
		dMid := altAt(mid) - thr
		if (dLo < 0) == (dMid < 0) {
			lo, dLo = mid, dMid
		} else {
			hi = mid
		}
	}
	return lo.Add(hi.Sub(lo) / 2)
}

// trendStep is the half-width of the window used to read the Sun's altitude
// trend at an instant. Near a culmination the altitude changes by well under a
// thousandth of a degree over it, but segments never contain a culmination, so
// the sign of the difference is reliable at every segment midpoint.
const trendStep = 30 * time.Second

// climbing reports whether the Sun's altitude is increasing at t.
func climbing(altAt func(time.Time) float64, t time.Time) bool {
	return altAt(t.Add(trendStep)) > altAt(t.Add(-trendStep))
}

// lowerCulminations returns every lower culmination of the Sun (solar
// midnight, a local minimum of altitude) strictly inside (start, end). Minima
// are bracketed by the coarse scan, starting one step before the day so a
// minimum near either edge is still seen, and refined by ternary search.
func lowerCulminations(altAt func(time.Time) float64, start, end time.Time) []time.Time {
	var out []time.Time
	prev2, prev1 := altAt(start.Add(-coarseStep)), altAt(start)
	for t := start.Add(coarseStep); !t.After(end.Add(coarseStep)); t = t.Add(coarseStep) {
		a := altAt(t)
		if prev1 <= prev2 && prev1 < a {
			centre := t.Add(-coarseStep)
			low := ternaryMax(func(x time.Time) float64 { return -altAt(x) }, centre.Add(-coarseStep), t)
			if low.After(start) && low.Before(end) {
				out = append(out, low)
			}
		}
		prev2, prev1 = prev1, a
	}
	return out
}

// zoneLabel names the band an altitude falls in: the deep-night label below the
// lowest level, otherwise the rising or setting label of the highest level at or
// below the altitude, chosen by whether the Sun is climbing or sinking.
func zoneLabel(seg Segmentation, alt float64, rising bool) string {
	if alt < seg.Levels[0].Altitude {
		return seg.Night
	}
	owner := seg.Levels[0]
	for _, lvl := range seg.Levels {
		if lvl.Altitude <= alt {
			owner = lvl
		} else {
			break
		}
	}
	if rising {
		return owner.Rising
	}
	return owner.Setting
}

// dedupTimes returns the sorted times with near-duplicates removed, always
// keeping start as the first element and end as the last so the day is covered
// contiguously with no zero-length segments.
func dedupTimes(sorted []time.Time, start, end time.Time) []time.Time {
	const minGap = time.Second
	out := make([]time.Time, 0, len(sorted))
	for _, t := range sorted {
		if t.Before(start) || t.After(end) {
			continue
		}
		if len(out) > 0 && t.Sub(out[len(out)-1]) < minGap {
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 || !out[0].Equal(start) {
		out = append([]time.Time{start}, out...)
	}
	if out[len(out)-1].Before(end) {
		if end.Sub(out[len(out)-1]) < minGap {
			out[len(out)-1] = end
		} else {
			out = append(out, end)
		}
	}
	return out
}

// solarNoon returns the instant of greatest altitude during the day, found by a
// coarse scan bracketing the peak and a ternary search refining it.
func solarNoon(altAt func(time.Time) float64, start, end time.Time) time.Time {
	best := start
	bestAlt := altAt(start)
	for t := start.Add(coarseStep); t.Before(end); t = t.Add(coarseStep) {
		if a := altAt(t); a > bestAlt {
			best, bestAlt = t, a
		}
	}
	lo := best.Add(-coarseStep)
	if lo.Before(start) {
		lo = start
	}
	hi := best.Add(coarseStep)
	if hi.After(end) {
		hi = end
	}
	return ternaryMax(altAt, lo, hi)
}

// dayMinAltitude returns the least altitude during the day, found by a coarse
// scan bracketing the trough and a ternary search refining it. It supports
// polar-state detection.
func dayMinAltitude(altAt func(time.Time) float64, start, end time.Time) float64 {
	worst := start
	worstAlt := altAt(start)
	for t := start.Add(coarseStep); !t.After(end); t = t.Add(coarseStep) {
		if a := altAt(t); a < worstAlt {
			worst, worstAlt = t, a
		}
	}
	lo := worst.Add(-coarseStep)
	if lo.Before(start) {
		lo = start
	}
	hi := worst.Add(coarseStep)
	if hi.After(end) {
		hi = end
	}
	return altAt(ternaryMax(func(t time.Time) float64 { return -altAt(t) }, lo, hi))
}

// ternaryMax returns the instant of maximum value of f in [lo, hi], assuming f
// is unimodal on the interval.
func ternaryMax(f func(time.Time) float64, lo, hi time.Time) time.Time {
	for i := 0; i < 60 && hi.Sub(lo) > 200*time.Millisecond; i++ {
		third := hi.Sub(lo) / 3
		m1 := lo.Add(third)
		m2 := hi.Add(-third)
		if f(m1) < f(m2) {
			lo = m1
		} else {
			hi = m2
		}
	}
	return lo.Add(hi.Sub(lo) / 2)
}

// SegmentAt returns the band containing t, the fraction of that band elapsed at
// t (in [0, 1)), and any error, using the Earth DefaultSegmentation. It resolves
// the correct civil day internally and stitches a band that runs through
// midnight, so a live clock reads continuously from one civil day into the next:
// the day's trailing band and the next day's leading band, when they carry the
// same label, are reported as one span.
func SegmentAt(obs astronomy.Observer, t time.Time) (Segment, float64, error) {
	return SegmentAtWith(obs, t, DefaultSegmentation)
}

// SegmentAtWith is SegmentAt with a caller-supplied Segmentation, adjusted for
// the observer's height as in NewSunTimesWith.
func SegmentAtWith(obs astronomy.Observer, t time.Time, seg Segmentation) (Segment, float64, error) {
	if err := validateObserver(obs); err != nil {
		return Segment{}, 0, err
	}
	seg = seg.atHeight(obs.Height)
	if err := validateSegmentation(seg); err != nil {
		return Segment{}, 0, err
	}

	altAt := sunAltitudes(obs)
	day := build(obs, t, seg, altAt)
	idx := day.indexAt(t)
	if idx < 0 {
		// t sits exactly on the closing midnight; fold it into the next day.
		day = build(obs, day.end, seg, altAt)
		idx = day.indexAt(t)
		if idx < 0 {
			idx = len(day.segments) - 1
		}
	}
	band := day.segments[idx]

	// A band that runs through midnight carries the same label on both dates, so
	// join it with its neighbour across the boundary. A band that ends exactly at
	// midnight has a differently labelled neighbour and stays as it is.
	seg2 := band
	switch {
	case idx == 0:
		prev := build(obs, day.start.AddDate(0, 0, -1), seg, altAt)
		if last := prev.segments[len(prev.segments)-1]; last.Label == band.Label {
			seg2 = Segment{Label: band.Label, From: last.From, To: band.To}
		}
	case idx == len(day.segments)-1:
		next := build(obs, day.end, seg, altAt)
		if first := next.segments[0]; first.Label == band.Label {
			seg2 = Segment{Label: band.Label, From: band.From, To: first.To}
		}
	}
	seg2.Seconds = seg2.To.Sub(seg2.From).Seconds()

	span := seg2.To.Sub(seg2.From).Seconds()
	frac := 0.0
	if span > 0 {
		frac = t.Sub(seg2.From).Seconds() / span
	}
	return seg2, frac, nil
}

// indexAt returns the index of the segment containing t, or -1 if t is not
// within the day.
func (s *SunTimes) indexAt(t time.Time) int {
	for i, seg := range s.segments {
		if seg.Contains(t) {
			return i
		}
	}
	return -1
}
