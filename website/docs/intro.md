---
id: intro
title: Overview
slug: /intro
sidebar_position: 1
---

# 🔭 go-astronomy

> Observer-aware Sun, Moon, star, and constellation positions for Go — altitude/azimuth, twilight bands, arc tracks, and moon phases for any `(latitude, longitude, timezone, time)`.

`go-astronomy` computes where the Sun, Moon, and stars are in the sky for a given observer and instant. It emits **degrees** (altitude, azimuth) and time-progress — never pixels — so any consumer can drive an animated sky, a rise/set table, a twilight timeline, or a moon-phase widget from the same data.

## ✨ What it is

- 🌍 **Universal Sun plus per-body traits.** Only the Sun is universal to the solar system; each body package (currently `earth`, including `earth/moon`) owns that body's observer, atmosphere, and naming traits. Stars and constellations are universal catalogs.
- 🧵 **Stateless and concurrency-safe.** `time.Time` is always a parameter, never captured at construction, so the same instance serves many callers at once. A service can compute a distinct sky per site visitor.
- 📐 **Discs, not points.** Sun and Moon positions are the center of the disc, always paired with an angular diameter, so a consumer can size the disc and reason about alignment and overlap (eclipses, occultations) from the data alone.
- 🎯 **Arcminute-class accuracy.** The math is built on [`soniakeys/meeus`](https://github.com/soniakeys/meeus), a mature implementation of Jean Meeus' *Astronomical Algorithms*. Nutation, ΔT, and leap seconds are deliberately omitted — they sit below that precision floor.

## 🧭 How the packages fit together

| Package | Kind | What it provides |
| --- | --- | --- |
| `github.com/Bugs5382/go-astronomy` | universal | Shared types: `Observer`, `Horizontal`, `Position`, `AngularDiameter`, and the error-code registry via `Errors()`. |
| `.../sun` | universal | The Sun's apparent angular size versus distance (observer-independent physics only). |
| `.../star` | universal | Embedded HYG catalog (RA/Dec plus distance). |
| `.../constellation` | universal | IAU boundary lookup by RA/Dec. |
| `.../earth` | Earth traits | Sun position and track, twilight bands, refraction, polar states, seasons, ground darkness, and star projection for an Earth observer. |
| `.../earth/moon` | Earth traits | Luna: position, apparent position, phases, rise/set, and an arc track. |

Adding a future body (for example `mars/`) would not change `sun`, `star`, or `constellation`.

## 🙏 Credits and licensing

- The algorithmic foundation is Jean Meeus' *Astronomical Algorithms*, via [`soniakeys/meeus`](https://github.com/soniakeys/meeus) (with `soniakeys/unit`). MIT-licensed.
- The embedded star catalog is a curated subset of the [HYG database](https://codeberg.org/astronexus/hyg) (Hipparcos-Yale-Gliese), used under [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/).
- The constellation boundaries are the IAU boundaries of Eugène Delporte (1930), digitized by Nancy Roman (1987) as [VizieR VI/42](https://vizier.cds.unistra.fr/viz-bin/VizieR?-source=VI/42).

The library itself is MIT-licensed. See the repository `NOTICE` for the full attribution of the embedded data.

## 📚 Where to go next

- 🚀 [Getting started](./getting-started.md) — install and run a first example.
- 🧠 [Concepts](./concepts.md) — the universal-vs-per-body split, the projection contract, midnight rollover, and time handling.
- 📦 [Package reference](./reference/sun.md) — a page per package, tied to the shipped API.
- 🌐 [pkg.go.dev](https://pkg.go.dev/github.com/Bugs5382/go-astronomy) — the full generated godoc.
