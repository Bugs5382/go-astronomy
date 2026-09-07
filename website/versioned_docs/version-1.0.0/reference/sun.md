---
id: sun
title: sun
sidebar_position: 1
---

# ☀️ Package `sun`

Import path: `github.com/Bugs5382/go-astronomy/sun`

Universal Sun physics: quantities that depend only on the Sun itself, not on any observer, atmosphere, or vantage body. Its centerpiece is the Sun's apparent angular size as a function of distance. Anything that depends on where the Sun is seen from — altitude, azimuth, solar noon, twilight — is a per-body concern and lives in a body package such as [`earth`](./earth.md).

Angles are in degrees. The package is stateless and concurrency-safe.

## 📏 Apparent size

```go
const SemidiameterArcsecAt1AU = 959.63

func ApparentSemidiameter(distanceAU float64) float64
func ApparentDiameter(distanceAU float64) astronomy.AngularDiameter
```

- `SemidiameterArcsecAt1AU` is the Sun's apparent angular semidiameter, in arc seconds, seen from one astronomical unit (Meeus, *Astronomical Algorithms*, chapter 55). It is a physical property of the Sun.
- `ApparentSemidiameter` returns the semidiameter in **degrees** seen from `distanceAU` astronomical units. It scales inversely with distance: at one AU it equals `SemidiameterArcsecAt1AU` converted to degrees.
- `ApparentDiameter` returns the full apparent diameter as an `astronomy.AngularDiameter` (degrees) — twice the semidiameter.

Because this is universal physics, any vantage body obtains the Sun's apparent size by supplying its own distance to the Sun. On Earth, [`earth.SunPosition`](./earth.md) already pairs the Sun's disc center with its `ApparentDiameter` for the Earth-Sun distance at the instant.

```go
// Apparent diameter of the Sun seen from ~1 AU.
d := sun.ApparentDiameter(1.0) // ~0.533 degrees
```

Full godoc: [pkg.go.dev/github.com/Bugs5382/go-astronomy/sun](https://pkg.go.dev/github.com/Bugs5382/go-astronomy/sun).
