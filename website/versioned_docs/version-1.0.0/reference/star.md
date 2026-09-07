---
id: star
title: star
sidebar_position: 4
---

# ⭐ Package `star`

Import path: `github.com/Bugs5382/go-astronomy/star`

A universal, observer-independent catalog of stars: the quantities that belong to a star itself, not to any vantage. There is deliberately no altitude, azimuth, or observer here — turning a star's equatorial position into a local alt/az direction needs a rotating body and an observer, which is a vantage concern handled by [`earth.StarPosition`](./earth.md).

The catalog is a curated subset of the [HYG database](https://codeberg.org/astronexus/hyg) (Hipparcos-Yale-Gliese), embedded at build time. To keep the binary small the subset is the naked-eye sky plus every named star: all stars with apparent magnitude at or below 6.5, together with every star with a proper name regardless of magnitude — a few thousand stars rather than the full ~120,000-row catalog. Distance is retained in parsecs so a future vantage other than Earth can reproject a star from its true three-dimensional position. The HYG data is used under CC BY-SA 4.0; see the repository `NOTICE`. The package is stateless and concurrency-safe; the catalog is parsed once on first use.

## 🌟 The Star type

```go
type Star struct {
	ProperName    string  // common or IAU name, e.g. "Sirius", or ""
	Designation   string  // Bayer/Flamsteed, e.g. "Alp CMa", or ""
	Constellation string  // three-letter IAU abbreviation, e.g. "CMa"
	RA            float64 // right ascension, degrees, [0, 360), J2000
	Dec           float64 // declination, degrees, [-90, 90], J2000
	Distance      float64 // distance from the Sun, parsecs, or 0 if unknown
	Magnitude     float64 // apparent visual magnitude; smaller is brighter
	HIP           string  // Hipparcos number, or ""
	HD            string  // Henry Draper number, or ""
	HR            string  // Harvard Revised (Bright Star) number, or ""
	Gliese        string  // Gliese-Jahreiss identifier, or ""
}
```

## 🔎 Accessors

```go
func All() []Star
func Count() int
func GetNamedStar(name string) (Star, bool)
func ListNamedStars() []Star
func Brighter(maxMagnitude float64) []Star
func InMagnitudeRange(brightest, faintest float64) ([]Star, error)
```

- `All` returns the whole catalog, sorted by ascending magnitude (brightest first) with proper name and designation as tie-breakers, so every accessor returns a stable, deterministic order. `Count` is its length.
- `GetNamedStar` looks a star up by proper name (case-insensitive); the boolean is false when there is no match.
- `ListNamedStars` returns only the stars that carry a proper name.
- `Brighter` returns the stars at or brighter than `maxMagnitude` (magnitude at or below the value).
- `InMagnitudeRange` returns the stars with magnitude in `[brightest, faintest]`. It returns `ErrEmptyMagnitudeRange` (a coded error) when the bright bound is fainter than the faint bound, which could select no star.

```go
sirius, ok := star.GetNamedStar("Sirius")
if ok {
	fmt.Println(sirius.Constellation, sirius.Magnitude) // CMa -1.44
}
naked := star.Brighter(6.5) // the naked-eye sky
```

To place a star in the sky for an observer, pass it to [`earth.StarPosition`](./earth.md).

Full godoc: [pkg.go.dev/github.com/Bugs5382/go-astronomy/star](https://pkg.go.dev/github.com/Bugs5382/go-astronomy/star).
