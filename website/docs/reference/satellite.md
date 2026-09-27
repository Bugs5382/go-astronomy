---
id: satellite
title: satellite
sidebar_position: 5
---

# 🛰️ Package `satellite`

Import path: `github.com/Bugs5382/go-astronomy/satellite`

Earth satellites, the ISS among them, placed in an observer's sky from an element set the caller supplies. An element set is measured, not derived, and goes stale in days, so it is an input: the package parses it and never fetches one.

## 📄 Element sets

```go
func ParseTLE(line1, line2 string) (Elements, error)
func ParseOMM(r io.Reader) ([]Elements, error)

func (e Elements) Epoch() time.Time
func (e Elements) Age(t time.Time) time.Duration
func (e Elements) Propagate(t time.Time) (posKm, velKmS [3]float64, err error) // TEME
```

`ParseTLE` validates both checksums and the column layout and reads Alpha-5 catalogue numbers (`A0001` is 100001). `ParseOMM` reads CCSDS Orbit Mean-Elements Messages as JSON (an object or an array, as CelesTrak serves them) or XML, and rejects any message whose mean element theory is not SGP4 or whose frame is not TEME. `Age` says how far an instant is from the epoch, so a consumer can decide whether an answer is still worth trusting: a low orbit's position degrades by kilometres a day.

Propagation is SGP4/SDP4 (near-Earth and deep-space), ported from the public-domain reference code of Vallado, Crawford, Hujsak and Kelso (2006, *Revisiting Spacetrack Report #3*), with WGS-72 constants and the improved operation mode. It matches the reference verification output to under 0.1 m.

## 🔭 Position

```go
type Look struct {
	astronomy.Horizontal         // geometric altitude and azimuth, degrees
	RangeKm, RangeRateKmS float64
	Sunlit                bool    // the Sun's centre is visible from the satellite
	SunAltitude           float64 // the Sun's altitude for the observer
	PhaseAngle            float64 // Sun-satellite-observer, degrees
	Latitude, Longitude, AltitudeKm float64 // sub-satellite point, WGS84
}

func Position(obs astronomy.Observer, e Elements, t time.Time) (Look, error)
func (l Look) Magnitude(stdMag float64) float64
const ISSStandardMagnitude = -1.8
```

The TEME state is rotated to the Earth-fixed frame through GMST 1982, the angle SGP4 defines TEME against (polar motion, a metre-level effect, is ignored, and UT1 is taken as UTC). `Magnitude` models the satellite as a diffusely reflecting sphere from its standard magnitude at 1000 km and a 90° phase angle. The standard magnitude is not part of an element set, so the caller supplies it; the ISS's is exported for convenience. It returns `+Inf` in shadow.

## 🌠 Passes

```go
type PassOptions struct {
	MinAltitude     float64       // degrees, 0 for the geometric horizon
	DarkSunAltitude float64       // the observer is dark below this Sun altitude, -6 by default
	Step            time.Duration // scan interval, 10 s by default
	StdMagnitude    float64       // NaN (the default) for no magnitudes
}
func DefaultPassOptions() PassOptions

type Pass struct {
	Rise, Peak, Set         PassEvent
	Visible                 bool
	VisibleFrom, VisibleTo  time.Time
	ShadowEntry, ShadowExit time.Time // zero when the pass has none
}

func Passes(obs astronomy.Observer, e Elements, from, to time.Time, opt PassOptions) ([]Pass, error)
```

A pass is an interval, not an instant: the station crosses the sky in minutes, and anything sampled at the Sun's cadence misses it. `Visible` means that for part of the pass the satellite is sunlit while the Sun is below `DarkSunAltitude` for the observer, and `ShadowEntry` is where it vanishes into the Earth's shadow. The window is at most 31 days (`MaxPassWindow`).

Against Skyfield, run on the same element set for two observers over two days, rise, peak, and set agree to 0.2 s, shadow crossings to 0.05 s, and peak altitude to 0.01°.

## 🚑 Errors

Every error is go-apperr coded: `CodeInvalidElements` (7014) for a TLE or OMM that cannot be parsed (`ErrMalformedTLE`, `ErrChecksum`, `ErrMalformedOMM`), `CodeSatellitePropagation` (7015) when SGP4 fails (`ErrDecayed` and the other propagation sentinels, with the details in a `*PropagationError`), `CodeInvalidPassWindow` (7016), and the usual observer codes.
