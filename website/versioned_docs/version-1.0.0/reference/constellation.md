---
id: constellation
title: constellation
sidebar_position: 5
---

# 🌌 Package `constellation`

Import path: `github.com/Bugs5382/go-astronomy/constellation`

Identifies which of the 88 IAU constellations a celestial position falls in, and lists the constellations. Identification is a universal, observer-independent geometry problem: it depends only on a point's equatorial coordinates, not on where or when it is seen, so it lives outside any vantage-body package.

The boundaries are the official IAU boundaries fixed by Delporte (1930) and rearranged for position lookup by Roman (1987), distributed as VizieR catalog VI/42 and embedded here. Because those boundaries are defined at the B1875.0 equinox, `FindAt` precesses the query position — given at the J2000.0 equinox, matching the [`star`](./star.md) package — back to B1875.0 before testing it. Constellations are a convention of the Earth's sky; this package ships the IAU convention as its data. It is stateless and concurrency-safe; the boundary table is parsed once on first use.

## 🗺️ The Constellation type

```go
type Constellation struct {
	Abbrev   string // three-letter IAU abbreviation, e.g. "Ori"
	Name     string // Latin nominative name, e.g. "Orion"
	Genitive string // Latin genitive name, e.g. "Orionis"
}
```

## 🔎 Lookup

```go
func List() []Constellation
func Lookup(abbrev string) (Constellation, bool)
func FindAt(raDeg, decDeg float64) (Constellation, bool)
```

- `List` returns the constellation metadata.
- `Lookup` resolves a three-letter abbreviation to its `Constellation`; the boolean is false when the abbreviation is unknown.
- `FindAt` returns the constellation containing the J2000 equatorial point `(raDeg, decDeg)`. The boolean is false when the point cannot be resolved.

```go
c, ok := constellation.FindAt(101.287, -16.716) // near Sirius
if ok {
	fmt.Println(c.Name) // Canis Major
}
```

Pair `FindAt` with a [`star.Star`](./star.md)'s `RA`/`Dec` to name the figure a star lies in, or feed it any equatorial coordinate.

Full godoc: [pkg.go.dev/github.com/Bugs5382/go-astronomy/constellation](https://pkg.go.dev/github.com/Bugs5382/go-astronomy/constellation).
