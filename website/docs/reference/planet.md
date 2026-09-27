---
id: planet
title: planet
sidebar_position: 4
---

# 🪐 Package `planet`

Import path: `github.com/Bugs5382/go-astronomy/planet`

Mercury, Venus, Mars, Jupiter, Saturn, Uranus, and Neptune in an observer's sky, in the same shape and with the same conventions as `earth/moon`. The package also publishes each planet's heliocentric position, Earth's included, which belongs to no observer. It carries its own generated VSOP87 tables, so a program that never imports it never links them. It is stateless and concurrency-safe: time is always a parameter.

## 🚀 Quick example

```go
london := astronomy.Observer{Lat: 51.5074, Lng: -0.1278, TZ: time.UTC}
when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
for _, b := range planet.Bodies {
	r, err := planet.Position(london, b, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%-8s alt %6.1f az %6.1f mag %5.1f elong %5.1f near Sun %v\n",
		b, r.Altitude, r.Azimuth, r.Magnitude, r.Elongation, r.NearSun)
}
```

```text
mercury  alt  -44.6 az  319.2 mag   1.1 elong  20.1 near Sun false
venus    alt  -57.1 az  345.2 mag  -4.1 elong  40.3 near Sun false
mars     alt   44.8 az  127.0 mag  -1.1 elong 165.1 near Sun false
jupiter  alt   48.0 az  137.4 mag  -2.5 elong 158.6 near Sun false
saturn   alt   -4.6 az  280.8 mag   0.8 elong  32.2 near Sun false
uranus   alt   37.7 az  254.7 mag   5.7 elong  80.8 near Sun false
neptune  alt  -12.7 az  286.6 mag   7.8 elong  22.2 near Sun false
```

Every example on this page is a Go `Example` test in the package (`planet/example_test.go`), so the output shown is what the code prints.

## 🌍 Bodies

```go
type Body int

const (
	Mercury Body = iota + 1
	Venus
	Earth
	Mars
	Jupiter
	Saturn
	Uranus
	Neptune
)

var Bodies = []Body{Mercury, Venus, Mars, Jupiter, Saturn, Uranus, Neptune}

func (b Body) String() string
```

`Bodies` lists every planet an observer on Earth can look at, in order from the Sun. `Earth` is a `Body` so its heliocentric position can be differenced with another planet's, but it is not a valid target for `Position` or the rise and set functions (`ErrInvalidBody`). `String` returns the lowercase name (`"venus"`), or `"unknown"` for a value that is not a planet.

| body | radius (km) | near-Sun limit | magnitude formula |
| --- | --- | --- | --- |
| Mercury | 2440.53 | 10° | 6th-order polynomial in the phase angle |
| Venus | 6051.8 | 5° | two pieces, split at a 163.7° phase angle |
| Mars | 3396.19 | 11.5° | two pieces, split at 50° (orbital-longitude term left out) |
| Jupiter | 71492 | 9° | quadratic in the phase angle |
| Saturn | 60268 | 11° | globe and rings, from the ring tilt |
| Uranus | 25559 | 15° | quadratic, plus the sub-Earth latitude term |
| Neptune | 24764 | 15° | quadratic |

The radii are the IAU mean equatorial radii (Archinal et al. 2018) and size each disc.

## 🔭 Position

```go
type Result struct {
	astronomy.Position         // Altitude, Azimuth, Diameter
	RA, Dec     float64
	DistanceAU  float64
	LightTime   time.Duration
	Magnitude   float64
	PhaseAngle  float64
	Illuminated float64
	Elongation  float64
	NearSun     bool
}

func Position(obs astronomy.Observer, b Body, t time.Time) (Result, error)
```

| field | unit | meaning |
| --- | --- | --- |
| `Altitude`, `Azimuth` | degrees | topocentric, geometric (no refraction); azimuth clockwise from true north. A negative altitude is below the horizon and is a valid answer. |
| `Diameter` | degrees | apparent angular diameter, from the IAU radius and the same observer-to-planet distance as the position |
| `RA`, `Dec` | degrees | topocentric apparent right ascension, `[0, 360)`, and declination, on the true equator and equinox of date |
| `DistanceAU` | au | observer to planet, where the light left it |
| `LightTime` | duration | how long ago the light now arriving left the planet |
| `Magnitude` | magnitudes | apparent visual magnitude |
| `PhaseAngle` | degrees | Sun-planet-observer angle; 0 when the disc is fully lit |
| `Illuminated` | fraction | lit fraction of the disc, `[0, 1]` |
| `Elongation` | degrees | angle between the planet and the Sun seen from the Earth, `[0, 180]` |
| `NearSun` | | `Elongation < NearSunElongation(b)` |

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
when := time.Date(2027, 2, 19, 22, 0, 0, 0, time.UTC) // near opposition
r, err := planet.Position(greenwich, planet.Mars, when)
if err != nil {
	panic(err)
}
fmt.Printf("altitude %.2f, azimuth %.2f degrees\n", r.Altitude, r.Azimuth)
fmt.Printf("RA %.4f, Dec %.4f degrees (true equator and equinox of date)\n", r.RA, r.Dec)
fmt.Printf("distance %.4f au, light-time %v\n", r.DistanceAU, r.LightTime.Round(time.Second))
fmt.Printf("diameter %.2f arcsec, magnitude %.2f\n", float64(r.Diameter)*3600, r.Magnitude)
fmt.Printf("phase angle %.2f, %.1f%% lit, elongation %.1f\n", r.PhaseAngle, 100*r.Illuminated, r.Elongation)
```

```text
altitude 44.53, azimuth 129.60 degrees
RA 154.3613, Dec 15.4028 degrees (true equator and equinox of date)
distance 0.6779 au, light-time 5m38s
diameter 13.82 arcsec, magnitude -1.28
phase angle 2.66, 99.9% lit, elongation 175.5
```

### How a position is computed

The reduction follows Meeus, *Astronomical Algorithms*, 2nd ed., chapter 33:

1. The planet's and the Earth's heliocentric places come from VSOP87D at the instant the light left the planet, found by iterating the light-time (0.0057755183 days per au, Meeus 33.3). Evaluating the Earth at that same earlier instant folds the annual aberration in with the light-time.
2. The geocentric ecliptic place is moved to the FK5 system (Meeus 32.3), the nutation in longitude is added, and the true obliquity turns it into right ascension and declination of date.
3. The rigorous topocentric correction of Meeus chapter 40 moves it to the observer on the WGS84 ellipsoid, and apparent sidereal time gives the altitude and azimuth.
4. The distance, phase angle, and elongation come from the triangle of the Sun, the Earth at the observation instant, and the planet where the light left it.

### Magnitude

Magnitudes follow A. Mallama and J. L. Hilton, "Computing apparent planetary magnitudes for The Astronomical Almanac" (*Astronomy and Computing* 25, 10, 2018), the formulas the Almanac and JPL Horizons use: `V(1, α) + 5 log10(r Δ)`, with `α` the phase angle and `r` and `Δ` the distances from the Sun and the observer. Saturn's figure includes the rings, from their tilt toward the Earth (Meeus 45.3). Uranus's includes the sub-Earth latitude term, from the IAU pole. Mars's small orbital-longitude term, a few hundredths of a magnitude, is left out.

### Near the Sun

```go
func NearSunElongation(b Body) float64
```

```go
for _, b := range planet.Bodies {
	fmt.Printf("%s %.1f\n", b, planet.NearSunElongation(b))
}
// mercury 10.0, venus 5.0, mars 11.5, jupiter 9.0, saturn 11.0, uranus 15.0, neptune 15.0
```

`NearSun` flags a planet close enough to the Sun to be lost in its glare, whatever its altitude. The limits are the classical arcus visionis for the naked-eye planets and 15° for Uranus and Neptune. That is a heuristic: whether a planet is visible also depends on its magnitude, the sky, and the observer, so the elongation and magnitude are always reported for a consumer to decide.

## 🌅 Rise, set, and transit

```go
const HorizonAltitude = -0.5667

func NextRise(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error)
func NextTransit(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error)
```

Rise and set are the first instants strictly after `t` at which the planet's centre crosses `HorizonAltitude`, the geometric altitude of the refracted horizon (34 arc minutes of standard refraction; the discs are arc seconds, so the centre stands in for the planet). Transit is the upper culmination on the meridian, whether or not the planet is above the horizon then. Each returns `false` when nothing happens within 30 days, which can happen at high latitudes.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
from := time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC)
rise, _, _ := planet.NextRise(greenwich, planet.Jupiter, from)
transit, _, _ := planet.NextTransit(greenwich, planet.Jupiter, rise)
set, _, _ := planet.NextSet(greenwich, planet.Jupiter, transit)
// rise 05:06:53, transit 11:58:08, set 18:49:07 (UTC)
```

```go
// Saturn's culmination and its altitude there.
t, _, _ := planet.NextTransit(greenwich, planet.Saturn, time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))
r, _ := planet.Position(greenwich, planet.Saturn, t)
// 14:15:14 at altitude 41.6
```

```go
// A planet that never sets: Jupiter, high in Taurus, from Svalbard.
svalbard := astronomy.Observer{Lat: 78.22, Lng: 15.65, TZ: time.UTC}
set, ok, _ := planet.NextSet(svalbard, planet.Jupiter, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
// ok is false: no set within 30 days
```

## ☀️ Heliocentric positions

```go
type HeliocentricPosition struct {
	Lon        float64 // ecliptic longitude, degrees, [0, 360)
	Lat        float64 // ecliptic latitude, degrees
	DistanceAU float64 // from the Sun, au
}

func Heliocentric(b Body, t time.Time) (HeliocentricPosition, error)
func (h HeliocentricPosition) Vector() [3]float64
```

A heliocentric position is the planet seen from the centre of the Sun, on the ecliptic and equinox of date (the dynamical frame of VSOP87D). It is geometric: no light-time or aberration, because it belongs to no observer. `Vector` gives it as rectangular ecliptic coordinates in au, with x toward the equinox of date and z toward the north ecliptic pole. The geometric view of one planet from another is the difference of two vectors at the same instant, which is what an observer on another body (issue 49) builds on.

```go
when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
for _, b := range []planet.Body{planet.Earth, planet.Jupiter} {
	h, _ := planet.Heliocentric(b, when)
	fmt.Printf("%-7s L %.4f B %.4f R %.6f au\n", b, h.Lon, h.Lat, h.DistanceAU)
}
// earth   L 100.3232 B 0.0001 R 0.983343 au
// jupiter L 138.8042 B 0.8031 R 5.335777 au
```

```go
e, _ := planet.Heliocentric(planet.Earth, when)
m, _ := planet.Heliocentric(planet.Mars, when)
ev, mv := e.Vector(), m.Vector()
d := [3]float64{mv[0] - ev[0], mv[1] - ev[1], mv[2] - ev[2]}
// Mars from Earth: 0.9139 au, ecliptic longitude 159.87 degrees
```

## 📚 VSOP87 coverage

The positions come from VSOP87, version D (P. Bretagnon and G. Francou, "Planetary theories in rectangular and spherical variables. VSOP87 solutions", *Astronomy and Astrophysics* 202, 309, 1988): heliocentric ecliptic longitude, latitude, and radius on the dynamical ecliptic and equinox of date, as Poisson series in Julian millennia of TDB from J2000. The tables in `planet/vsop87_*.go` are generated from the IMCCE files by `internal/cmd/genvsop` (`go generate ./planet`). The generator checks each file's SHA-256 against a pinned value and records the source, digest, and cutoff in the generated header; the raw files are not committed.

Every term with `|A| × 0.3^α ≥ 3 × 10⁻⁸` is kept (radians or au, `α` the power of time), the same cut as the Earth table the Sun uses. The truncation error is the worst difference from the full series between 1700 and 2300, scaled to the planet's closest approach to the Earth.

| body | terms kept | of | truncation error, geocentric |
| --- | --- | --- | --- |
| Mercury | 405 | 6827 | 0.17″ |
| Venus | 256 | 1682 | 0.29″ |
| Mars | 791 | 5483 | 0.58″ |
| Jupiter | 919 | 3483 | 0.13″ |
| Saturn | 1643 | 5759 | 0.15″ |
| Uranus | 1908 | 3989 | 0.11″ |
| Neptune | 850 | 1929 | 0.12″ |

The theory itself is quoted by its authors as better than an arc second over a few thousand years around J2000 for most planets. Outside 1700 to 2300 the truncation grows slowly; the positions stay sound, but the accuracy table below was measured for 2020 to 2030.

## 🎯 Accuracy

Against JPL Horizons DE441 for a topocentric observer at Greenwich, at 13 epochs from 2020 to 2030 and at two conjunctions (Jupiter and Saturn on 2020-12-21, Venus and Jupiter on 2023-03-02):

| body | apparent place | altitude and azimuth | magnitude | diameter |
| --- | --- | --- | --- | --- |
| Mercury | 0.29″ | 2.5″ | 0.002 | < 0.001″ |
| Venus | 0.30″ | 2.5″ | 0.001 | < 0.001″ |
| Mars | 0.30″ | 2.2″ | 0.075 | < 0.001″ |
| Jupiter | 0.59″ | 2.1″ | 0.001 | < 0.001″ |
| Saturn | 0.54″ | 2.1″ | 0.046 | < 0.001″ |
| Uranus | 1.52″ | 3.1″ | 0.021 | < 0.001″ |
| Neptune | 1.70″ | 2.9″ | 0.015 | < 0.001″ |

- The Uranus and Neptune figures are VSOP87 against DE441. VSOP87 was fitted to an older ephemeris, and the truncation costs only a tenth of an arc second there.
- Altitude and azimuth also carry the sidereal time on UTC rather than UT1, up to 0.9 s of the Earth's rotation.
- Illuminated fraction, elongation, and phase angle agree to 0.01% and 0.02°. Distances agree to 5 × 10⁻⁷ au for the inner planets and 7 × 10⁻⁵ au for Neptune.
- Rise, set, and transit for Venus, Mars, Jupiter, and Saturn fall within Horizons' one-minute step.
- The package tests (`planet/planet_test.go`) hold these numbers against committed Horizons fixtures, with the queries and fetch date in each header.

## 🧭 Frames and units

| quantity | frame | unit |
| --- | --- | --- |
| `Heliocentric` | ecliptic and equinox of date (VSOP87D dynamical frame), geometric | degrees, au |
| `Result.RA`, `Result.Dec` | true equator and equinox of date, topocentric, apparent (light-time, aberration, nutation) | degrees |
| `Result.Altitude`, `Result.Azimuth` | local horizon, geometric (no refraction) | degrees, azimuth from true north through east |
| `Result.Diameter` | apparent angular diameter from the observer | degrees |
| times | `time.Time` in, UTC; the theory runs on Terrestrial Time internally | |

## 🚑 Errors

```go
var ErrInvalidLatitude, ErrInvalidLongitude, ErrInvalidBody error
```

Every error is go-apperr coded. An out-of-range observer carries `CodeInvalidLatitude` or `CodeInvalidLongitude`. A value that is not a planet, or `Earth` as a target, carries `CodeInvalidBody` (7013). `Heliocentric` accepts `Earth`.

## 📖 Sources

- P. Bretagnon and G. Francou 1988, VSOP87 (IMCCE files: https://ftp.imcce.fr/pub/ephem/planets/vsop87/).
- J. Meeus, *Astronomical Algorithms*, 2nd ed., chapters 32, 33, 40, 41, 45, and 48.
- A. Mallama and J. L. Hilton 2018, planetary magnitudes.
- B. A. Archinal et al. 2018, IAU planetary radii and poles.
- JPL Horizons, DE441, for the accuracy checks.
