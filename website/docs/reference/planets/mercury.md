---
id: mercury
title: mercury
sidebar_position: 1
---

# 🪐 Package `mercury`

Import path: `github.com/Bugs5382/go-astronomy/planet/mercury`

Mercury in an observer's sky, and Mercury's heliocentric position. The package holds only Mercury's own VSOP87 table, so a program that imports it links no other planet's table; the results are the shared types of [`planet`](../planet.md).

## 📦 API

```go
const (
	Name              = "mercury"
	RadiusKm          = 2440.53   // IAU mean equatorial radius, km
	NearSunElongation = 10        // degrees
)

var Planet planet.Body // Mercury, for code that handles every planet alike

func Position(obs astronomy.Observer, t time.Time) (planet.Result, error)
func Heliocentric(t time.Time) planet.HeliocentricPosition
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
```

- `Position` returns a [`planet.Result`](../planet.md#result): topocentric geometric altitude and azimuth with the apparent diameter, the apparent RA and Dec of date, distance and light-time, magnitude, phase angle, illuminated fraction, elongation, and the `NearSun` flag. A negative altitude is below the horizon and is a valid answer.
- `Heliocentric` is Mercury seen from the Sun's centre, on the ecliptic and equinox of date; it belongs to no observer. Difference it with `planet.EarthHeliocentric` for the geometric view from Earth.
- `NextRise` and `NextSet` are the centre crossing `planet.HorizonAltitude` (−0.5667°, standard refraction), lowered by the horizon dip for the observer's `Height` (see [Observer and height](../observer.md)). `NextTransit` is the upper culmination, whether or not Mercury is up. Each reports `false` when nothing happens within 30 days.
- `NearSun` is set when the elongation is under 10°. That is a heuristic, so the elongation and the magnitude are always reported too.
- The observer's `Height` also enters the parallax, where it moves a planet by far less than 0.1″.
- Errors are go-apperr coded: `planet.ErrInvalidLatitude` and `planet.ErrInvalidLongitude` for an out-of-range observer, and `astronomy.ErrInvalidHeight` for a NaN or infinite height.

## 🚀 Examples

Each example is an `Example` test in `planet/mercury/example_test.go`; the output shown is what it prints.

### Position

Where Mercury is from Greenwich, how big and bright it looks, how much of it is lit, and how long its light took.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
r, err := mercury.Position(greenwich, when)
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
altitude -44.71, azimuth 319.33 degrees
RA 322.7443, Dec -12.4001 degrees (true equator and equinox of date)
distance 0.6998 au, light-time 5m49s
diameter 9.62 arcsec, magnitude 1.05
phase angle 124.15, 21.9% lit, elongation 20.1, near Sun false
```

### Heliocentric

Mercury seen from the Sun, which belongs to no observer.

```go
h := mercury.Heliocentric(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
fmt.Printf("L %.4f B %.4f R %.6f au\n", h.Lon, h.Lat, h.DistanceAU)
```

```text
L 279.0275 B -5.4069 R 0.458543 au
```

### NextRise

Mercury's rise, transit, and set at Greenwich from 1 September 2027.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
rise, ok, err := mercury.NextRise(greenwich, time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC))
if err != nil || !ok {
	panic(err)
}
transit, _, _ := mercury.NextTransit(greenwich, rise)
set, _, _ := mercury.NextSet(greenwich, transit)
fmt.Println("rise   ", rise.Format(time.DateTime))
fmt.Println("transit", transit.Format(time.DateTime))
fmt.Println("set    ", set.Format(time.DateTime))
```

```text
rise    2027-09-01 06:54:56
transit 2027-09-01 13:07:25
set     2027-09-01 19:17:56
```

## 🎯 Accuracy

- **Theory:** VSOP87D (Bretagnon and Francou 1988), generated from the IMCCE file `VSOP87D.mer` and truncated at `|A| × 0.3^α ≥ 3 × 10⁻⁸`. The truncation error is at most 0.17″ in Mercury's closest geocentric geometry, 1700 to 2300.
- **Against JPL Horizons DE441** (Greenwich, 2020 to 2030, 13 epochs, plus the conjunctions in the fixture): the apparent place is within 0.29″, altitude and azimuth within 2.50″, and the magnitude within 0.002.
- **Magnitude** follows Mallama and Hilton (2018): a sixth-order polynomial in the phase angle.
- **Frames and units:** see the [planets overview](../planet.md).
