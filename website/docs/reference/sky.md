---
id: sky
title: sky
sidebar_position: 5
---

# 🔭 sky: standing on another world

`github.com/Bugs5382/go-astronomy/sky` puts the observer on any body in the solar system and resolves any other body from there: Earth from Mars, the Sun from the Moon, Jupiter from Venus. It is the general form of what `earth`, `earth/moon`, and the planet packages do for an observer on Earth, and it sits beside them. Nothing in the v1 surface changes, and a site on Earth gives exactly the v1 answers.

## 📦 API

```go
type Body struct{ /* unexported */ }

var Sun, Earth, Moon *Body
func Planet(p planet.Body) (*Body, error) // for example Planet(mars.Planet)

func (b *Body) Name() string
func (b *Body) RadiusKm() float64
func (b *Body) PolarRadiusKm() float64
func (b *Body) HorizonRefraction() float64          // degrees; 0 when airless or not modelled
func (b *Body) Twilight() (earth.Segmentation, bool) // false when the air makes no twilight
func (b *Body) RotationPeriod() time.Duration        // sidereal; negative when retrograde
func (b *Body) SolarDay() time.Duration              // noon to noon

type Site struct {
	Body   *Body
	Lat    float64          // planetodetic latitude, degrees
	Lon    float64          // east longitude, degrees, [-180, 180]
	Height astronomy.Height // above the reference ellipsoid, optional
}

type Result struct {
	astronomy.Position         // apparent alt/az of the disc centre, and its diameter
	DistanceAU  float64
	LightTime   time.Duration
	PhaseAngle  float64 // Sun-target-site, degrees
	Illuminated float64 // lit fraction
	Elongation  float64 // from the Sun, degrees
}

func Position(site Site, target *Body, t time.Time) (Result, error)

type State int // Crossed, AlwaysAbove, AlwaysBelow, NoEvent
type Event struct {
	Time  time.Time // UTC, when State is Crossed
	State State
}
func (e Event) Found() bool

func NextRise(site Site, target *Body, t time.Time) (Event, error)
func NextSet(site Site, target *Body, t time.Time) (Event, error)
func NextTransit(site Site, target *Body, t time.Time) (Event, error)
func Window(b *Body) time.Duration

func Segments(site Site, from, to time.Time) ([]earth.Segment, error)
```

## 🪨 What a body carries

Everything that made the old sky Earth-only is now a property of the body:

| property | Earth | Moon | planets |
| --- | --- | --- | --- |
| rotation | precession, nutation, and apparent sidereal time, as in `earth` | IAU WGCCRE 2015 pole and prime meridian, with the 13 periodic terms | IAU WGCCRE 2015 (Mars with the Kuchynka periodic terms, Jupiter System III) |
| shape | the reference ellipsoid | a 1737.4 km sphere | the IAU reference ellipsoid (the 1 bar level on the giants) |
| refraction at the horizon | 34′ | 0 (no air) | 0 (not modelled) |
| twilight | `earth.DefaultSegmentation` | none | none |
| search window for rise and set | 30 days, as the v1 packages | 65 days | two solar days or two rotations |

Mars does have a thin atmosphere. It refracts a body on the horizon by well under an arc minute and makes a long, dusty twilight with no conventional levels, so neither is modelled here, and JPL Horizons treats Mars as airless too. Longitude is east-positive on every body; Mars's west-positive planetographic longitude converts as `360 - west` (Jezero, 282.55° W, is 77.45° E).

## 🚀 Examples

### Earth from the Moon

From the middle of the near side, Earth hangs high in the sky, goes through phases, and neither rises nor sets.

```go
site := sky.Site{Body: sky.Moon, Lat: 0, Lon: 0}
when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

earth, _ := sky.Position(site, sky.Earth, when)
fmt.Printf("altitude %.1f, azimuth %.1f, %.1f degrees across, %.0f%% lit\n",
	earth.Altitude, earth.Azimuth, float64(earth.Diameter), earth.Illuminated*100)
// altitude 80.6, azimuth 50.0, 1.9 degrees across, 62% lit

rise, _ := sky.NextRise(site, sky.Earth, when)
fmt.Println(rise.State) // always_above
```

Earth's phase from the Moon is the complement of the Moon's from Earth: the two lit fractions add up to one, to within half a percent.

### Sunrise on Mars

```go
m, _ := sky.Planet(mars.Planet)
jezero := sky.Site{Body: m, Lat: 18.44, Lon: 77.45}

rise, _ := sky.NextRise(jezero, sky.Sun, time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC))
set, _ := sky.NextSet(jezero, sky.Sun, rise.Time)
// sunrise 2027-01-01T18:38:12Z, sunset 2027-01-02T07:47:56Z
fmt.Printf("%.2f Mars hours\n", mars.LocalMeanSolarTime(rise.Time, jezero.Lon)) // 5.75
fmt.Println(m.SolarDay().Round(time.Second))                                     // 24h39m35s
```

### A lunar day

An airless body's day divides at the Sun's horizon crossings and nowhere else: no twilight bands.

```go
segs, _ := sky.Segments(sky.Site{Body: sky.Moon}, from, from.AddDate(0, 0, 45))
// night Jan 01 00:00 to Jan 15 04:36 (14.2 days)
// day   Jan 15 04:36 to Jan 30 01:32 (14.9 days)
// night Jan 30 01:32 to Feb 13 19:02 (14.7 days)
// day   Feb 13 19:02 to Feb 15 00:00 (1.2 days)
```

## 🌅 Rise, set, and transit

`NextRise` and `NextSet` find the target's upper limb crossing the horizon, lowered by the body's horizontal refraction and by the dip of the site's height (the geometric dip off Earth). The search steps at 1/96 of the shorter of the body's rotation and solar day and refines each crossing to under a second. A target that stays on one side of the horizon for the whole `Window` comes back as `AlwaysAbove` or `AlwaysBelow`, never as a failure: Earth from the lunar near side is `AlwaysAbove`, from the far side `AlwaysBelow`, and the midnight Sun and the polar night on Earth come back the same way. `NextTransit` finds the upper culmination on the site's meridian, or `NoEvent` when the target does not cross it in the window.

On Earth the Sun rises where `earth.NewSunTimes` starts its sunrise band and transits at its solar noon, and the Moon and the planets rise and set where their own packages say.

## 🌗 Segments

`Segments(site, from, to)` covers the window with labelled spans. On Earth they are the `earth.NewSunTimes` bands of the UTC civil days the window touches, cut to the window, with a band that runs through midnight as one span. On a body with no twilight they are `earth.LabelDay` while the Sun's upper limb is above the horizon and `earth.LabelNight` while it is below.

## 🎯 Accuracy

Measured against JPL Horizons (DE441, airless, apparent), with the fixtures in `sky/testdata`:

| from | target | direction | rise and set |
| --- | --- | --- | --- |
| Jezero, Mars | the Sun, Earth, Jupiter | under 0.4″ | inside Horizons' one-minute step |
| Venus | the Sun | 0.1″ | |
| the Moon | the Sun, Earth, Mars | about 11″ | Sun within 1 minute, Earth within 4 minutes |

From the Moon the difference is the frame: Horizons orients a lunar site in the mean Earth frame of the DE441 libration, and the IAU rotation model approximates that frame to about 10″, which shows up the same for the Sun, Earth, and Mars. Earth rises slowly from the lunar limb, so those 10″ are a few minutes. The light-time and the aberration of the site's motion are applied; the gravitational bending of light is not (under 0.01″ away from the Sun's limb). An Earth site gives the v1 answers exactly. The same general transform, run on Earth, agrees with them to under 1″.

## 🚑 Errors

| sentinel | code | when |
| --- | --- | --- |
| `sky.ErrUnknownBody` | 7018 `CodeUnknownBody` | a nil site body or target, a site on the Sun, or a planet with no IAU model |
| `sky.ErrSameBody` | 7019 `CodeSameBody` | the target is the body the site stands on |
| `sky.ErrInvalidLatitude`, `sky.ErrInvalidLongitude` | 7001, 7002 | a site outside `[-90, 90]` or `[-180, 180]` |
| `astronomy.ErrInvalidHeight` | 7011 | a NaN or infinite height |

## 🕰️ Time

Every instant is a `time.Time`, returned in UTC: time zones are an Earth institution, and a `Site` has none. Mars keeps its own clock in `planet/mars`: `mars.SolDate(t)` is the Mars Sol Date of Allison and McEwen (2000), `mars.LocalMeanSolarTime(t, lonEast)` the local mean solar time in Mars hours, and `mars.Sol` the 24h 39m 35.244s mean solar day.
