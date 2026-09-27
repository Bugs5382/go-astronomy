---
id: mars
title: mars
sidebar_position: 3
---

# 🪐 Package `mars`

Import path: `github.com/Bugs5382/go-astronomy/planet/mars`

Mars in an observer's sky, and Mars's heliocentric position. The package holds only Mars's own VSOP87 table, so a program that imports it links no other planet's table; the results are the shared types of [`planet`](../planet.md).

## 📦 API

```go
const (
	Name              = "mars"
	RadiusKm          = 3396.19 // IAU mean equatorial radius, km
	NearSunElongation = 11.5    // degrees
)

var Planet planet.Body // Mars, for code that handles every planet alike

func Position(obs astronomy.Observer, t time.Time) (planet.Result, error)
func Heliocentric(t time.Time) planet.HeliocentricPosition
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error)

// The Mars clock (Allison and McEwen 2000, as in Mars24).
const Sol = 88775244147 * time.Microsecond // the mean solar day, 24h 39m 35.244s
func SolDate(t time.Time) float64                              // Mars Sol Date
func LocalMeanSolarTime(t time.Time, lonEastDeg float64) float64 // Mars hours, [0, 24)
```

- `Position` returns a [`planet.Result`](../planet.md#result): topocentric geometric altitude and azimuth with the apparent diameter, the apparent RA and Dec of date, distance and light-time, magnitude, phase angle, illuminated fraction, elongation, and the `NearSun` flag. A negative altitude is below the horizon and is a valid answer.
- `Heliocentric` is Mars seen from the Sun's centre, on the ecliptic and equinox of date; it belongs to no observer. Difference it with `planet.EarthHeliocentric` for the geometric view from Earth.
- `NextRise` and `NextSet` are the centre crossing `planet.HorizonAltitude` (−0.5667°, standard refraction), lowered by the horizon dip for the observer's `Height` (see [Observer and height](../observer.md)). `NextTransit` is the upper culmination, whether or not Mars is up. Each reports `false` when nothing happens within 30 days.
- `NearSun` is set when the elongation is under 11.5°. That is a heuristic, so the elongation and the magnitude are always reported too.
- The observer's `Height` also enters the parallax, where it moves a planet by far less than 0.1″.
- `SolDate` is the Mars Sol Date, the count of sols since 1873 December 29 that Mars24 and the mission clocks use (44795.9998 at 2000 January 6, 00:00 UTC). `LocalMeanSolarTime` is the local mean solar time at an east longitude, in Mars hours (a 24th of a sol). To stand on Mars and look out, see [sky](../sky.md).
- Errors are go-apperr coded: `planet.ErrInvalidLatitude` and `planet.ErrInvalidLongitude` for an out-of-range observer, and `astronomy.ErrInvalidHeight` for a NaN or infinite height.

## 🚀 Examples

Each example is an `Example` test in `planet/mars/example_test.go`; the output shown is what it prints.

### Position

Where Mars is from Greenwich, how big and bright it looks, how much of it is lit, and how long its light took.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
when := time.Date(2027, 2, 19, 22, 0, 0, 0, time.UTC) // near opposition
r, err := mars.Position(greenwich, when)
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
altitude 44.53, azimuth 129.60 degrees
RA 154.3613, Dec 15.4028 degrees (true equator and equinox of date)
distance 0.6779 au, light-time 5m38s
diameter 13.82 arcsec, magnitude -1.28
phase angle 2.66, 99.9% lit, elongation 175.5, near Sun false
```

### Heliocentric

Mars seen from the Sun, which belongs to no observer.

```go
h := mars.Heliocentric(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
fmt.Printf("L %.4f B %.4f R %.6f au\n", h.Lon, h.Lat, h.DistanceAU)
```

```text
L 128.8690 B 1.8161 R 1.646686 au
```

### NextRise

Mars's rise, transit, and set at Greenwich from 1 September 2027.

```go
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
rise, ok, err := mars.NextRise(greenwich, time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC))
if err != nil || !ok {
	panic(err)
}
transit, _, _ := mars.NextTransit(greenwich, rise)
set, _, _ := mars.NextSet(greenwich, transit)
fmt.Println("rise   ", rise.Format(time.DateTime))
fmt.Println("transit", transit.Format(time.DateTime))
fmt.Println("set    ", set.Format(time.DateTime))
```

```text
rise    2027-09-01 10:03:59
transit 2027-09-01 15:07:54
set     2027-09-01 20:11:15
```

## 🎯 Accuracy

- **Theory:** VSOP87D (Bretagnon and Francou 1988), generated from the IMCCE file `VSOP87D.mar` and truncated at `|A| × 0.3^α ≥ 3 × 10⁻⁸`. The truncation error is at most 0.58″ in Mars's closest geocentric geometry, 1700 to 2300.
- **Against JPL Horizons DE441** (Greenwich, 2020 to 2030, 13 epochs, plus the conjunctions in the fixture): the apparent place is within 0.30″, altitude and azimuth within 2.20″, and the magnitude within 0.075.
- **Magnitude** follows Mallama and Hilton (2018): two polynomials in the phase angle, split at 50°; the orbital-longitude terms (a few hundredths of a magnitude) are left out.
- **Frames and units:** see the [planets overview](../planet.md).
