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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
)

// NightDarknessDayAltitude is the Sun's geometric center altitude, in degrees,
// at or above which NightDarkness is 0: the ground is in full daylight and needs
// no darkening. It sits at the geometric horizon.
const NightDarknessDayAltitude = 0.0

// NightDarknessNightAltitude is the Sun's geometric center altitude, in degrees,
// at or below which NightDarkness is 1: the sky is fully dark. It matches the
// astronomical twilight threshold of the default segmentation (the lowest level
// of DefaultSegmentation, -18 degrees), so the darkening ramp ends exactly where
// deep night begins.
const NightDarknessNightAltitude = -18.0

// NightDarkness returns a geometry factor in [0, 1] describing how dark the
// ground is at instant t for the observer, driven purely by the Sun's altitude.
// It is 0 when the Sun is at or above the horizon (full day) and ramps smoothly
// to 1 as the Sun sinks through twilight to the astronomical threshold
// (NightDarknessNightAltitude), staying at 1 through deep night. The ramp uses a
// smoothstep so the factor and its first derivative are continuous at both
// endpoints and it rises monotonically as the Sun descends.
//
// This is geometry, not appearance: the library emits the factor and the
// consumer multiplies it into its own ground colors to darken them at night. The
// library never owns a palette. The thresholds line up with the twilight bands
// in DefaultSegmentation, so the darkening tracks the same civil, nautical, and
// astronomical crossings the segmentation reports.
func NightDarkness(obs astronomy.Observer, t time.Time) float64 {
	alt := SunPosition(obs, t).Altitude
	return darknessFactor(alt)
}

// darknessFactor maps a Sun altitude in degrees to the darkness factor: 0 at or
// above the day threshold, 1 at or below the night threshold, and a smoothstep
// ramp in between. It is the pure kernel behind NightDarkness.
func darknessFactor(altDeg float64) float64 {
	if altDeg >= NightDarknessDayAltitude {
		return 0
	}
	if altDeg <= NightDarknessNightAltitude {
		return 1
	}
	// Normalize so x is 0 at the day threshold and 1 at the night threshold, the
	// deeper the Sun the larger x.
	x := (NightDarknessDayAltitude - altDeg) / (NightDarknessDayAltitude - NightDarknessNightAltitude)
	return smoothstep(x)
}

// smoothstep is the classic cubic Hermite ease 3x^2 - 2x^3 on x in [0, 1]. It is
// 0 at 0 and 1 at 1 with zero slope at both ends, and is strictly increasing on
// the open interval, so it gives a smooth, monotonic twilight ramp.
func smoothstep(x float64) float64 {
	return x * x * (3 - 2*x)
}
