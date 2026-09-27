---
id: planet
title: planet
sidebar_position: 4
---

# 🪐 Planets

Each planet is its own package, with its own VSOP87 table and the same small API. A program that imports `planet/mars` links only the Mars table (plus the Earth table the Sun already needs). `planet` holds the shared types every planet returns, and `planet/all` imports every planet for a caller who wants them all.

| package | import path | page |
| --- | --- | --- |
| `planet` | `github.com/Bugs5382/go-astronomy/planet` | this page: shared types |
| `mercury` | `github.com/Bugs5382/go-astronomy/planet/mercury` | [mercury](./planets/mercury.md) |
| `venus` | `github.com/Bugs5382/go-astronomy/planet/venus` | [venus](./planets/venus.md) |
| `mars` | `github.com/Bugs5382/go-astronomy/planet/mars` | [mars](./planets/mars.md) |
| `jupiter` | `github.com/Bugs5382/go-astronomy/planet/jupiter` | [jupiter](./planets/jupiter.md) |
| `saturn` | `github.com/Bugs5382/go-astronomy/planet/saturn` | [saturn](./planets/saturn.md) |
| `uranus` | `github.com/Bugs5382/go-astronomy/planet/uranus` | [uranus](./planets/uranus.md) |
| `neptune` | `github.com/Bugs5382/go-astronomy/planet/neptune` | [neptune](./planets/neptune.md) |
| `all` | `github.com/Bugs5382/go-astronomy/planet/all` | [below](#-every-planet-planetall) |

Every planet package exports the same identifiers:

```go
const Name, RadiusKm, NearSunElongation
var Planet planet.Body

func Position(obs astronomy.Observer, t time.Time) (planet.Result, error)
func Heliocentric(t time.Time) planet.HeliocentricPosition
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
```

```go
r, err := mars.Position(greenwich, time.Date(2027, 2, 19, 22, 0, 0, 0, time.UTC))
// altitude 44.53, azimuth 129.60; 13.82 arcsec, magnitude -1.28; light-time 5m38s
```

## 🧩 Shared types (`planet`)

### Result

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
| `NearSun` | | the elongation is under the planet's `NearSunElongation` |

### HeliocentricPosition and EarthHeliocentric

```go
type HeliocentricPosition struct {
	Lon        float64 // ecliptic longitude, degrees, [0, 360)
	Lat        float64 // ecliptic latitude, degrees
	DistanceAU float64 // from the Sun, au
}

func (h HeliocentricPosition) Vector() [3]float64 // rectangular ecliptic, au
func EarthHeliocentric(t time.Time) HeliocentricPosition
```

A heliocentric position is the planet seen from the Sun's centre, on the ecliptic and equinox of date (the dynamical frame of VSOP87D). It is geometric: no light-time or aberration, because it belongs to no observer. `EarthHeliocentric` gives Earth's, from the table the Sun already uses, so the geometric view of a planet from Earth is the difference of two vectors:

```go
when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
ev, mv := planet.EarthHeliocentric(when).Vector(), mars.Heliocentric(when).Vector()
d := [3]float64{mv[0] - ev[0], mv[1] - ev[1], mv[2] - ev[2]}
// Mars from Earth: 0.9139 au, ecliptic longitude 159.87 degrees
```

### Body

```go
type Body interface {
	Name() string
	RadiusKm() float64
	NearSunElongation() float64
	Position(obs astronomy.Observer, t time.Time) (Result, error)
	Heliocentric(t time.Time) HeliocentricPosition
	NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
	NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
	NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
}
```

Each planet package's `Planet` value implements `Body`, with methods that are the package functions, for code that treats every planet alike.

### Horizon and errors

```go
const HorizonAltitude = -0.5667
var ErrInvalidLatitude, ErrInvalidLongitude error
```

`NextRise` and `NextSet` find the planet's centre crossing `HorizonAltitude`, the geometric altitude of the refracted horizon (34 arc minutes of standard refraction; the discs are arc seconds, so the centre stands in for the planet). Every error is go-apperr coded: an out-of-range observer carries `CodeInvalidLatitude` or `CodeInvalidLongitude`.

## 🌌 Every planet (`planet/all`)

```go
func Planets() []planet.Body              // Mercury to Neptune, in order from the Sun
func ByName(name string) (planet.Body, bool)
```

```go
london := astronomy.Observer{Lat: 51.5074, Lng: -0.1278, TZ: time.UTC}
when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
for _, p := range all.Planets() {
	r, err := p.Position(london, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%-8s alt %6.1f az %6.1f mag %5.1f elong %5.1f near Sun %v\n",
		p.Name(), r.Altitude, r.Azimuth, r.Magnitude, r.Elongation, r.NearSun)
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

`ByName("jupiter")` returns Jupiter's `Body`; Earth is not in the list, because it is not a target (use `planet.EarthHeliocentric`). Importing `planet/all` links every planet's table; that is its only cost, and a caller who never imports it pays nothing.

## 🔗 What a program links

Importing one planet links only that planet's table. Each planet package's `deps_test.go` checks with `go list -deps` that it depends on no other planet package and not on `planet/all`. A program that only calls `mars.Position` contains one table symbol, `planet/mars.table`, and builds to 4.57 MB; the same program over `planet/all` contains all seven and builds to 4.74 MB.

## 🧮 How a position is computed

The reduction is shared by every planet package and follows Meeus, *Astronomical Algorithms*, 2nd ed., chapter 33:

1. The planet's and the Earth's heliocentric places come from VSOP87D at the instant the light left the planet, found by iterating the light-time (0.0057755183 days per au, Meeus 33.3). Evaluating the Earth at that same earlier instant folds the annual aberration in with the light-time.
2. The geocentric ecliptic place is moved to the FK5 system (Meeus 32.3), the nutation in longitude is added, and the true obliquity turns it into right ascension and declination of date.
3. The rigorous topocentric correction of Meeus chapter 40 moves it to the observer on the WGS84 ellipsoid, and apparent sidereal time gives the altitude and azimuth.
4. The distance, phase angle, and elongation come from the triangle of the Sun, the Earth at the observation instant, and the planet where the light left it.

## ✨ Magnitude

Magnitudes follow A. Mallama and J. L. Hilton, "Computing apparent planetary magnitudes for The Astronomical Almanac" (*Astronomy and Computing* 25, 10, 2018), the formulas the Almanac and JPL Horizons use: `V(1, α) + 5 log10(r Δ)`, with `α` the phase angle and `r` and `Δ` the distances from the Sun and the observer. Saturn's figure includes the rings, from their tilt toward the Earth (Meeus 45.3). Uranus's includes the sub-Earth latitude term, from the IAU pole. Mars's small orbital-longitude term, a few hundredths of a magnitude, is left out.

Each planet's formula is in its own package's `magnitude.go`, and each planet's page says which.

## ☀️ Near the Sun

`NearSun` flags a planet close enough to the Sun to be lost in its glare, whatever its altitude, when the elongation is under the package's `NearSunElongation`: Mercury 10°, Venus 5°, Mars 11.5°, Jupiter 9°, Saturn 11° (the classical arcus visionis), and 15° for Uranus and Neptune. That is a heuristic: whether a planet is visible also depends on its magnitude, the sky, and the observer, so the elongation and magnitude are always reported for a consumer to decide.

## 📚 VSOP87 coverage

The positions come from VSOP87, version D (P. Bretagnon and G. Francou, "Planetary theories in rectangular and spherical variables. VSOP87 solutions", *Astronomy and Astrophysics* 202, 309, 1988): heliocentric ecliptic longitude, latitude, and radius on the dynamical ecliptic and equinox of date, as Poisson series in Julian millennia of TDB from J2000. Each table, `planet/<name>/vsop87.go`, is generated from the IMCCE files by `internal/cmd/genvsop` (`go generate ./planet/...`). The generator checks each file's SHA-256 against a pinned value and records the source, digest, and cutoff in the generated header; the raw files are not committed.

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
- Each planet package's tests (`planet/<name>/position_test.go`, `magnitude_test.go`, `riseset_test.go`) hold these numbers against committed Horizons fixtures, with the queries and fetch date in each header.

## 🧭 Frames and units

| quantity | frame | unit |
| --- | --- | --- |
| `Heliocentric` | ecliptic and equinox of date (VSOP87D dynamical frame), geometric | degrees, au |
| `Result.RA`, `Result.Dec` | true equator and equinox of date, topocentric, apparent (light-time, aberration, nutation) | degrees |
| `Result.Altitude`, `Result.Azimuth` | local horizon, geometric (no refraction) | degrees, azimuth from true north through east |
| `Result.Diameter` | apparent angular diameter from the observer | degrees |
| times | `time.Time` in, UTC; the theory runs on Terrestrial Time internally | |

## 📖 Sources

- P. Bretagnon and G. Francou 1988, VSOP87 (IMCCE files: https://ftp.imcce.fr/pub/ephem/planets/vsop87/).
- J. Meeus, *Astronomical Algorithms*, 2nd ed., chapters 32, 33, 40, 41, 45, and 48.
- A. Mallama and J. L. Hilton 2018, planetary magnitudes.
- B. A. Archinal et al. 2018, IAU planetary radii and poles.
- JPL Horizons, DE441, for the accuracy checks.
