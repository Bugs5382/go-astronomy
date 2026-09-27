---
id: saturn
title: saturn
sidebar_position: 5
---

# 🪐 Package `saturn`

Import path: `github.com/Bugs5382/go-astronomy/planet/saturn`

Saturn in an observer's sky, and Saturn's heliocentric position. The package holds only Saturn's own VSOP87 table, so a program that imports it links no other planet's table; the results are the shared types of [`planet`](../planet.md).

## 📦 API

```go
const (
	Name              = "saturn"
	RadiusKm          = 60268    // IAU mean equatorial radius, km
	NearSunElongation = 11       // degrees
)

var Planet planet.Body // Saturn, for code that handles every planet alike

func Position(obs astronomy.Observer, t time.Time) (planet.Result, error)
func Heliocentric(t time.Time) planet.HeliocentricPosition
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
```

- `Position` returns a [`planet.Result`](../planet.md#result): topocentric geometric altitude and azimuth with the apparent diameter, the apparent RA and Dec of date, distance and light-time, magnitude, phase angle, illuminated fraction, elongation, and the `NearSun` flag. A negative altitude is below the horizon and is a valid answer.
- `Heliocentric` is Saturn seen from the Sun's centre, on the ecliptic and equinox of date; it belongs to no observer. Difference it with `planet.EarthHeliocentric` for the geometric view from Earth.
- `NextRise` and `NextSet` are the centre crossing `planet.HorizonAltitude` (−0.5667°, standard refraction), lowered by the horizon dip for the observer's `Height` (see [Observer and height](../observer.md)). `NextTransit` is the upper culmination, whether or not Saturn is up. Each reports `false` when nothing happens within 30 days.
- `NearSun` is set when the elongation is under 11°. That is a heuristic, so the elongation and the magnitude are always reported too.
- The observer's `Height` also enters the parallax, where it moves a planet by far less than 0.1″.
- Errors are go-apperr coded: `planet.ErrInvalidLatitude` and `planet.ErrInvalidLongitude` for an out-of-range observer, and `astronomy.ErrInvalidHeight` for a NaN or infinite height.

## 🚀 Examples

Each example is an `Example` test in `planet/saturn/example_test.go`; the output shown is what it prints.

### Position

Where Saturn is from Greenwich, how big and bright it looks, how much of it is lit, and how long its light took.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
r, err := saturn.Position(greenwich, when)
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
altitude -4.64, azimuth 280.92 degrees
RA 13.0027, Dec 3.1127 degrees (true equator and equinox of date)
distance 10.2124 au, light-time 1h24m56s
diameter 16.27 arcsec, magnitude 0.80
phase angle 3.22, 99.9% lit, elongation 32.2, near Sun false
```

### Heliocentric

Saturn seen from the Sun, which belongs to no observer.

```go
h := saturn.Heliocentric(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
fmt.Printf("L %.4f B %.4f R %.6f au\n", h.Lon, h.Lat, h.DistanceAU)
```

```text
L 14.3336 B -2.4514 R 9.406765 au
```

### NextRise

Saturn's rise, transit, and set at Greenwich from 1 September 2027.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
rise, ok, err := saturn.NextRise(greenwich, time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC))
if err != nil || !ok {
	panic(err)
}
transit, _, _ := saturn.NextTransit(greenwich, rise)
set, _, _ := saturn.NextSet(greenwich, transit)
fmt.Println("rise   ", rise.Format(time.DateTime))
fmt.Println("transit", transit.Format(time.DateTime))
fmt.Println("set    ", set.Format(time.DateTime))
```

```text
rise    2027-09-01 20:17:59
transit 2027-09-02 03:01:30
set     2027-09-02 09:44:57
```

### NextTransit

Saturn's culmination, and its altitude there.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
t, ok, err := saturn.NextTransit(greenwich, time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))
if err != nil || !ok {
	panic(err)
}
r, _ := saturn.Position(greenwich, t)
fmt.Printf("%s at altitude %.1f\n", t.Format(time.TimeOnly), r.Altitude)
```

```text
14:15:14 at altitude 41.6
```

## 🎯 Accuracy

- **Theory:** VSOP87D (Bretagnon and Francou 1988), generated from the IMCCE file `VSOP87D.sat` and truncated at `|A| × 0.3^α ≥ 3 × 10⁻⁸`. The truncation error is at most 0.15″ in Saturn's closest geocentric geometry, 1700 to 2300.
- **Against JPL Horizons DE441** (Greenwich, 2020 to 2030, 13 epochs, plus the conjunctions in the fixture): the apparent place is within 0.54″, altitude and azimuth within 2.08″, and the magnitude within 0.046.
- **Magnitude** follows Mallama and Hilton (2018): the globe and the rings together, from the tilt of the rings toward the Earth (Meeus 45.3).
- **Frames and units:** see the [planets overview](../planet.md).
