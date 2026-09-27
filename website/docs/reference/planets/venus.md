---
id: venus
title: venus
sidebar_position: 2
---

# 🪐 Package `venus`

Import path: `github.com/Bugs5382/go-astronomy/planet/venus`

Venus in an observer's sky, and Venus's heliocentric position. The package holds only Venus's own VSOP87 table, so a program that imports it links no other planet's table; the results are the shared types of [`planet`](../planet.md).

## 📦 API

```go
const (
	Name              = "venus"
	RadiusKm          = 6051.8  // IAU mean equatorial radius, km
	NearSunElongation = 5       // degrees
)

var Planet planet.Body // Venus, for code that handles every planet alike

func Position(obs astronomy.Observer, t time.Time) (planet.Result, error)
func Heliocentric(t time.Time) planet.HeliocentricPosition
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
```

- `Position` returns a [`planet.Result`](../planet.md#result): topocentric geometric altitude and azimuth with the apparent diameter, the apparent RA and Dec of date, distance and light-time, magnitude, phase angle, illuminated fraction, elongation, and the `NearSun` flag. A negative altitude is below the horizon and is a valid answer.
- `Heliocentric` is Venus seen from the Sun's centre, on the ecliptic and equinox of date; it belongs to no observer. Difference it with `planet.EarthHeliocentric` for the geometric view from Earth.
- `NextRise` and `NextSet` are the centre crossing `planet.HorizonAltitude` (−0.5667°, standard refraction). `NextTransit` is the upper culmination, whether or not Venus is up. Each reports `false` when nothing happens within 30 days.
- `NearSun` is set when the elongation is under 5°. That is a heuristic, so the elongation and the magnitude are always reported too.
- Errors are go-apperr coded: `planet.ErrInvalidLatitude` and `planet.ErrInvalidLongitude` for an out-of-range observer.

## 🚀 Examples

Each example is an `Example` test in `planet/venus/example_test.go`; the output shown is what it prints.

### Position

Where Venus is from Greenwich, how big and bright it looks, how much of it is lit, and how long its light took.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
r, err := venus.Position(greenwich, when)
if err != nil {
	panic(err)
}
fmt.Printf("altitude %.2f, azimuth %.2f degrees\n", r.Altitude, r.Azimuth)
fmt.Printf("RA %.4f, Dec %.4f degrees (true equator and equinox of date)\n", r.RA, r.Dec)
fmt.Printf("distance %.4f au, light-time %v\n", r.DistanceAU, r.LightTime.Round(time.Second))
fmt.Printf("diameter %.2f arcsec, magnitude %.2f\n", float64(r.Diameter)*3600, r.Magnitude)
fmt.Printf("phase angle %.2f, %.1f%% lit, elongation %.1f, near Sun %v\n", r.PhaseAngle, 100*r.Illuminated, r.Elongation, r.NearSun)
```

```text
altitude -57.16, azimuth 345.41 degrees
RA 302.7612, Dec -19.3002 degrees (true equator and equinox of date)
distance 1.0927 au, light-time 9m5s
diameter 15.27 arcsec, magnitude -4.07
phase angle 62.25, 73.3% lit, elongation 40.3, near Sun false
```

### Heliocentric

Venus seen from the Sun, which belongs to no observer.

```go
h := venus.Heliocentric(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
fmt.Printf("L %.4f B %.4f R %.6f au\n", h.Lon, h.Lat, h.DistanceAU)
```

```text
L 141.8581 B 3.0760 R 0.718531 au
```

### NextRise

Venus's rise, transit, and set at Greenwich from 1 September 2027.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
rise, ok, err := venus.NextRise(greenwich, time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC))
if err != nil || !ok {
	panic(err)
}
transit, _, _ := venus.NextTransit(greenwich, rise)
set, _, _ := venus.NextSet(greenwich, transit)
fmt.Println("rise   ", rise.Format(time.DateTime))
fmt.Println("transit", transit.Format(time.DateTime))
fmt.Println("set    ", set.Format(time.DateTime))
```

```text
rise    2027-09-01 05:41:02
transit 2027-09-01 12:23:16
set     2027-09-01 19:04:05
```

## 🎯 Accuracy

- **Theory:** VSOP87D (Bretagnon and Francou 1988), generated from the IMCCE file `VSOP87D.ven` and truncated at `|A| × 0.3^α ≥ 3 × 10⁻⁸`. The truncation error is at most 0.29″ in Venus's closest geocentric geometry, 1700 to 2300.
- **Against JPL Horizons DE441** (Greenwich, 2020 to 2030, 13 epochs, plus the conjunctions in the fixture): the apparent place is within 0.30″, altitude and azimuth within 2.50″, and the magnitude within 0.001.
- **Magnitude** follows Mallama and Hilton (2018): two polynomials in the phase angle, split at 163.7°.
- **Frames and units:** see the [planets overview](../planet.md).
