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
	"sort"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// Season is one of the four astronomical seasons, bounded by the equinoxes and
// solstices. The zero value is Spring. The numbering is chosen so that the
// season half a year away is (s + 2) mod 4, which is what the Southern
// Hemisphere observes for the same instant.
type Season int

const (
	// Spring begins at the observer's spring equinox.
	Spring Season = iota
	// Summer begins at the observer's summer solstice (the longest day).
	Summer
	// Fall begins at the observer's autumnal equinox.
	Fall
	// Winter begins at the observer's winter solstice (the shortest day).
	Winter
)

// String returns a lowercase name for the season.
func (s Season) String() string {
	switch s {
	case Spring:
		return "spring"
	case Summer:
		return "summer"
	case Fall:
		return "fall"
	case Winter:
		return "winter"
	default:
		return "unknown"
	}
}

// SeasonInfo describes where an instant falls within the astronomical seasons
// for an observer. The seasons are bounded by the equinox and solstice instants
// and are flipped for the Southern Hemisphere, where a given instant is the
// season six months from the Northern one.
type SeasonInfo struct {
	// Season is the season in progress at the queried instant.
	Season Season
	// Next is the season that begins at End.
	Next Season
	// Start is the UTC instant the current season began (its opening equinox or
	// solstice).
	Start time.Time
	// End is the UTC instant the current season ends, which is when Next begins.
	End time.Time
	// Progress is the fraction of the current season elapsed at the queried
	// instant, in [0, 1).
	Progress float64
	// DaysElapsed is the number of days from Start to the queried instant.
	DaysElapsed float64
	// DaysUntilNext is the number of days from the queried instant to End.
	DaysUntilNext float64
}

// northernSeasonStarts lists the four equinox and solstice instants of one year
// paired with the Northern-Hemisphere season each one begins, in chronological
// order.
func northernSeasonStarts(year int) [4]struct {
	when   time.Time
	season Season
} {
	return [4]struct {
		when   time.Time
		season Season
	}{
		{julian.FromTT(ephemeris.MarchEquinox(year)), Spring},
		{julian.FromTT(ephemeris.JuneSolstice(year)), Summer},
		{julian.FromTT(ephemeris.SeptemberEquinox(year)), Fall},
		{julian.FromTT(ephemeris.DecemberSolstice(year)), Winter},
	}
}

// SeasonAt returns the astronomical season in progress for the observer at
// instant t, computed from the equinox and solstice instants and flipped for the
// Southern Hemisphere by the sign of the observer's latitude: a negative
// latitude observes the season six months from the Northern one. The result
// carries the current and next season, the season's UTC start and end, progress
// through it, and the day counts on either side of the queried instant.
//
// It returns a go-apperr coded error when the observer's latitude or longitude
// is out of range: match the condition with errors.Is against ErrInvalidLatitude
// or ErrInvalidLongitude, or recover the code with apperr.Code (see the
// astronomy package's code registry).
func SeasonAt(obs astronomy.Observer, t time.Time) (SeasonInfo, error) {
	if err := validateObserver(obs); err != nil {
		return SeasonInfo{}, err
	}

	utc := t.UTC()

	// Build the boundary instants for the year of t plus the years on either
	// side, so the season containing t (which may have opened in the previous
	// year or close in the next) is always bracketed.
	type boundary struct {
		when   time.Time
		season Season
	}
	year := utc.Year()
	var boundaries []boundary
	for y := year - 1; y <= year+1; y++ {
		for _, s := range northernSeasonStarts(y) {
			boundaries = append(boundaries, boundary{s.when, s.season})
		}
	}
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].when.Before(boundaries[j].when)
	})

	// Locate the interval [boundaries[i], boundaries[i+1]) that contains t. The
	// three-year window always brackets t for supported years; the clamp is a
	// defensive fallback that keeps the index in range at the window edges.
	idx := 0
	for i := 0; i+1 < len(boundaries); i++ {
		if !utc.Before(boundaries[i].when) && utc.Before(boundaries[i+1].when) {
			idx = i
			break
		}
		if !utc.Before(boundaries[i+1].when) {
			idx = i + 1
		}
	}
	if idx > len(boundaries)-2 {
		idx = len(boundaries) - 2
	}

	start := boundaries[idx].when
	end := boundaries[idx+1].when
	season := hemisphereSeason(boundaries[idx].season, obs.Lat)
	next := hemisphereSeason(boundaries[idx+1].season, obs.Lat)

	total := end.Sub(start).Seconds()
	elapsed := utc.Sub(start).Seconds()
	progress := 0.0
	if total > 0 {
		progress = elapsed / total
	}

	return SeasonInfo{
		Season:        season,
		Next:          next,
		Start:         start,
		End:           end,
		Progress:      progress,
		DaysElapsed:   elapsed / secondsPerDay,
		DaysUntilNext: end.Sub(utc).Seconds() / secondsPerDay,
	}, nil
}

// secondsPerDay is the number of seconds in a nominal day, used to convert the
// season spans into day counts.
const secondsPerDay = 24 * 60 * 60

// hemisphereSeason maps a Northern-Hemisphere season to the season observed at
// the given latitude. Southern latitudes (lat < 0) observe the season two slots
// away, six months apart; the equator and Northern latitudes keep the Northern
// season.
func hemisphereSeason(northern Season, lat float64) Season {
	if lat < 0 {
		return Season((int(northern) + 2) % 4)
	}
	return northern
}
