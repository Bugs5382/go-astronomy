package sky

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
	"time"

	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// Segments divides the site's sky from from to to into labelled spans of the
// Sun's day, in chronological order, covering the window with no gaps.
//
// On Earth they are the bands of earth.NewSunTimes (twilight, sunrise,
// golden hour, day), taken from the UTC civil days the window touches and cut
// to it; a band that runs across midnight is one span. A body with no
// twilight (see Body.Twilight) divides only at the Sun's horizon crossings:
// the spans are earth.LabelDay while the Sun's upper limb is above the horizon
// and earth.LabelNight while it is below. On the Moon a day and a night each
// last about two weeks.
//
// A window whose end is not after its start gives no spans. The cost grows
// with the window: a year on Mars is about 64 000 steps.
func Segments(site Site, from, to time.Time) ([]earth.Segment, error) {
	if err := validateSite(site); err != nil {
		return nil, err
	}
	if !to.After(from) {
		return nil, nil
	}
	if site.Body.kind == kindEarth {
		return earthSegments(site, from, to)
	}
	f := func(when time.Time) float64 {
		l := site.look(Sun, julian.TT(when))
		return l.alt + l.semi + site.Body.refraction + site.dip()
	}
	step := site.Body.searchStep()
	bounds := []time.Time{from}
	prev, prevT := f(from), from
	for when := from.Add(step); ; when = when.Add(step) {
		if when.After(to) {
			when = to
		}
		cur := f(when)
		if (prev < 0) != (cur < 0) {
			bounds = append(bounds, bisect(f, prevT, when))
		}
		if !when.Before(to) {
			break
		}
		prev, prevT = cur, when
	}
	bounds = append(bounds, to)

	out := make([]earth.Segment, 0, len(bounds)-1)
	for i := 0; i+1 < len(bounds); i++ {
		a, b := bounds[i].UTC(), bounds[i+1].UTC()
		if !b.After(a) {
			continue
		}
		label := earth.LabelNight
		if f(a.Add(b.Sub(a)/2)) >= 0 {
			label = earth.LabelDay
		}
		out = append(out, earth.Segment{Label: label, From: a, To: b, Seconds: b.Sub(a).Seconds()})
	}
	return out, nil
}

// earthSegments assembles the v1 civil-day bands over the window.
func earthSegments(site Site, from, to time.Time) ([]earth.Segment, error) {
	obs := site.observer()
	var out []earth.Segment
	for day := from.UTC(); ; day = day.AddDate(0, 0, 1) {
		st, err := earth.NewSunTimes(obs, day)
		if err != nil {
			return nil, err
		}
		for _, s := range st.Segments() {
			if !s.To.After(from) || !s.From.Before(to) {
				continue
			}
			atMidnight := s.From.Equal(st.DayStart())
			if s.From.Before(from) {
				s.From = from
			}
			if s.To.After(to) {
				s.To = to
			}
			s.From, s.To = s.From.UTC(), s.To.UTC()
			// A band that runs through midnight is one span, as SegmentAt
			// reports it; the day's other splits (solar noon) are kept.
			if n := len(out); atMidnight && n > 0 && out[n-1].Label == s.Label && out[n-1].To.Equal(s.From) {
				out[n-1].To = s.To
				out[n-1].Seconds = out[n-1].To.Sub(out[n-1].From).Seconds()
				continue
			}
			s.Seconds = s.To.Sub(s.From).Seconds()
			out = append(out, s)
		}
		if !st.DayEnd().Before(to) {
			break
		}
	}
	return out, nil
}
