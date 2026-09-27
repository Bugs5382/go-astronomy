---
id: planet
title: planet
sidebar_position: 4
---

# 🪐 Package `planet`

Import path: `github.com/Bugs5382/go-astronomy/planet`

Mercury, Venus, Mars, Jupiter, Saturn, Uranus, and Neptune in an observer's sky, in the same shape and with the same conventions as `earth/moon`, plus each planet's heliocentric position, which belongs to no observer. The package carries its own generated VSOP87 tables, so a program that never imports it never links them. It is stateless and concurrency-safe.

## 🔭 Position

```go
type Body int // Mercury, Venus, Earth, Mars, Jupiter, Saturn, Uranus, Neptune
var Bodies []Body // every planet but Earth

type Result struct {
	astronomy.Position         // topocentric altitude/azimuth (geometric, no refraction) and apparent diameter
	RA, Dec     float64        // topocentric apparent, true equator and equinox of date
	DistanceAU  float64        // from the observer
	LightTime   time.Duration
	Magnitude   float64        // apparent visual magnitude
	PhaseAngle  float64        // Sun-planet-observer, degrees
	Illuminated float64        // lit fraction of the disc, [0, 1]
	Elongation  float64        // from the Sun, degrees
	NearSun     bool           // Elongation < NearSunElongation(body)
}

func Position(obs astronomy.Observer, b Body, t time.Time) (Result, error)
func NearSunElongation(b Body) float64
```

The reduction follows Meeus chapter 33: light-time and aberration (the planet and the Earth are both evaluated at the instant the light left the planet), the FK5 correction, nutation, and the true obliquity, then the rigorous topocentric correction. The diameter is sized from the same observer-to-planet distance as the position, using the IAU radii. Magnitudes follow Mallama and Hilton (2018), the formulas the Astronomical Almanac uses, including Saturn's rings from their tilt toward the Earth.

A negative altitude is below the horizon and is a valid answer. `NearSun` flags a planet close enough to the Sun to be lost in its glare, using the classical arcus visionis (Mercury 10°, Venus 5°, Mars 11.5°, Jupiter 9°, Saturn 11°, and 15° for Uranus and Neptune). That is a heuristic; the elongation and magnitude are always reported so a consumer can decide for itself.

## 🌅 Rise, set, and transit

```go
const HorizonAltitude = -0.5667
func NextRise(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error)
func NextTransit(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error)
```

Rise and set are the planet's centre crossing the refracted horizon. Transit is the upper culmination on the meridian, whether or not the planet is up. Each returns false when nothing happens within 30 days, which can happen at high latitudes.

## ☀️ Heliocentric positions

```go
type HeliocentricPosition struct {
	Lon, Lat   float64 // ecliptic of date, degrees
	DistanceAU float64
}
func (h HeliocentricPosition) Vector() [3]float64 // rectangular ecliptic, au
func Heliocentric(b Body, t time.Time) (HeliocentricPosition, error)
```

`Heliocentric` works for Earth too, so the geometric view of one planet from another is the difference of two vectors at the same instant.

## 🎯 Accuracy

Against JPL Horizons DE441 at 13 epochs from 2020 to 2030 and two conjunctions, the topocentric apparent place is within 0.3″ for Mercury, Venus, and Mars, 0.6″ for Jupiter and Saturn, and 1.7″ for Uranus and Neptune, where VSOP87 itself departs from DE441 by that much. Magnitudes agree to 0.08 or better (Mars is the worst, from its orbital-longitude term, which is left out), diameters to 0.001″, and rise, set, and transit to within Horizons' one-minute step.

Errors are go-apperr coded: `ErrInvalidLatitude`, `ErrInvalidLongitude`, and `ErrInvalidBody` (code 7013) for a value that is not a planet, or for Earth as a target.
