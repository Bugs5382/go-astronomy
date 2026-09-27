---
id: concepts
title: Concepts
sidebar_position: 3
---

# 🧠 Concepts

## 🌌 Universal geometry versus per-body traits

The library separates what is universal to the solar system from what belongs to a specific vantage body.

- **Universal** packages describe the sky itself. `sun` holds the Sun's apparent angular size versus distance — the only observer-independent part of the Sun. `star` holds an embedded catalog of stars with their equatorial coordinates and distance. `constellation` identifies which IAU figure a point on the sky falls in. each `planet/<name>` package publishes its planet's heliocentric position, which belongs to no observer, alongside the Earth-observer view of it. None of these know about an observer's horizon until asked.
- **Per-body** packages add the traits of one vantage. `earth` turns the universal Sun into an Earth observer's alt/az, twilight bands, refraction, polar states, and seasons. `earth/moon` is Luna, which belongs to Earth (another body would own its own moons).

This is why the alt/az of the Sun lives in `earth.SunPosition` and not in `sun`: altitude, azimuth, sidereal time, and the Earth-Sun distance are all Earth-specific. Adding a future body (for example `mars/`) would reuse `sun`, `star`, and `constellation` unchanged.

## 🎨 The library owns math, the consumer owns meaning and appearance

The library emits geometry as **degrees** plus time-progress. It never emits pixels or colors.

- **Colors** are the consumer's. Even the ground-darkness ramp (`earth.NightDarkness`) is a `[0, 1]` geometry factor the consumer multiplies into its own palette.
- **Twilight band names and thresholds are data**, not a hardcoded rule. `earth.DefaultSegmentation` ships the Earth defaults (astronomical −18°, nautical −12°, civil −6°, the −0.833° horizon crossing, golden hour, day). A consumer can supply a different `Segmentation` wholesale — for example to describe the same Sun with a different vocabulary.

## 📐 Discs, not points

`astronomy.Position` embeds a `Horizontal` direction (altitude and azimuth in degrees) and pairs it with an `AngularDiameter`. A position is therefore the direction to the **center of the disc** plus the disc's apparent size. That pairing lets a consumer size and place the disc, and compute alignment and overlap between bodies (eclipses, occultations) purely from the data. There is no single "point" for the Sun or Moon.

## 🖥️ The projection contract

The `project` package maps already-computed positions to canvas pixels. The canvas origin is the top-left corner, with x increasing right and y increasing down. The horizon is a line **inside** the canvas, not the bottom edge: altitude 0° maps to `y = HorizonFraction × CanvasH` (for example 0.70 of the height), so negative altitudes dip into a ground band below it.

The mapping is selectable through three enums on a `View`:

- **`XMode`** — `TimeProgress` maps the 0..1 sunrise-to-sunset progress across the full width (no azimuth axis). `Azimuth` maps the object's compass azimuth.
- **`YMode`** — `NormalizedByPeak` (default) normalizes altitude by a supplied peak so the arch fills the sky band, preserving an arch shape. `Geometric` uses a fixed degrees-per-pixel elevation scale.
- **`ViewMode`** (used when `XMode` is `Azimuth`) — `FullArc` spans the whole sunrise-to-sunset azimuth sweep across the width. `Directional` shows a heading-centered slice: `ViewHeading` sits at the canvas middle and `FieldOfView` spans the width, so an object outside the field is marked not visible.

Two helpers support direction-aware coloring without the package owning a palette: `ColumnAzimuth` inverts the azimuth X-map (screen column to the azimuth being looked at), and `AngularSeparation` returns the shortest angular difference between two azimuths across the 0/360° seam. Per column, a consumer can compute the azimuth, its separation from the Sun, and color as a function of that separation and the Sun's altitude.

See [`project`](./reference/project.md) for the full API.

## 🕛 Seamless midnight rollover

`earth.SegmentAt` answers "which band, and how far through it, at `now`" for a single instant. A caller asks only for `now`, never for the previous or next day: the resolver stitches across midnight so a live clock at 23:59:59 and again at 00:00:01 has no gap. `NewSunTimes` resolves the whole civil day of bands at once when you need the entire day rather than one instant.

## 🕰️ Time, timezone, and DST

Pass a zone-aware `time.Time` on every call. The library computes internally in UTC — Julian dates are UTC-native — and applies the observer's timezone (`Observer.TZ`, defaulting to UTC when nil) only at the boundary: day-boundary resolution and phase timestamps convert to the target zone. Because it uses `time.Location` rather than a fixed offset, daylight-saving transitions are handled correctly, so a spring-forward or fall-back civil day spans 23 or 25 hours and `SunTrack` samples honor that.

## ❄️ Polar states

At high latitudes the Sun may not cross the horizon on a given day. That is an explicit state, never a nil panic or a `-1` sentinel. `SunTimes.Polar()` returns a `PolarState` of `MidnightSun` (Sun up all day) or `PolarNight` (Sun down all day) with a boolean that is false on an ordinary day. Absence in general is explicit throughout the API: `(value, bool)` or `(value, error)` where a result may not exist, for example no sunrise during a polar day.

## 🪐 Heliocentric first, then an observer

A planet's position is computed in two steps, and the first is published on its own. Each planet package's `Heliocentric` (for example `mars.Heliocentric`) gives the planet seen from the Sun's centre, a vector that does not depend on anyone; its `Position` then reduces it to one observer on Earth, with light-time, aberration, nutation, and the topocentric correction. Keeping the first step public is what lets an observer on another body be built later: Earth seen from Mars is the difference of two heliocentric vectors, just as Mars seen from Earth is.

## ⛰️ Height and the horizon

An observer's height moves the horizon, not the sky. From height the sea horizon sits below the astronomical horizon by the dip, so rise and set move while positions barely change (the height shifts the Sun's and Moon's parallax by under an arc second). The height is optional: leave it out for sea level and no network, set it by hand in feet or metres, or look it up with a resolver. The calculation functions never look anything up. A moving observer, such as a plane, passes its position and height for each instant. See [Observer and height](./reference/observer.md).

## 🎯 Accuracy

The target is amateur, arcminute-class accuracy, using an in-house implementation of the Meeus algorithms. The Sun and Moon are computed on Terrestrial Time, with ΔT taken from the IERS leap-second table, and include nutation and the observer's parallax; against JPL Horizons DE441 the Moon is good to about 10″ and the Sun, from the VSOP87 series, to about 2″. Star positions still omit nutation and aberration. Do not rely on the library for higher-precision ephemeris work.
