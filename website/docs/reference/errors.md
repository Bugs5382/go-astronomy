---
id: errors
title: Errors
sidebar_position: 7
---

# 🚑 Errors and the code registry

Every returned `error` in go-astronomy is a [go-apperr](https://github.com/Bugs5382/go-apperr) coded error carrying a stable numeric code. A caller can match a condition two ways:

- with `errors.Is` against a package sentinel (for example `earth.ErrInvalidLatitude`), or
- by recovering the code with `apperr.Code(err)` and branching on the number.

The codes are part of the public contract: a code is never renumbered, only added. Every go-astronomy code shares the leading service digit `7`, exported as `astronomy.ErrorServiceDigit`, so a code is attributable to this module at a glance.

## 🔢 Code table

| Code | Constant | Meaning |
| --- | --- | --- |
| 7001 | `CodeInvalidLatitude` | Observer latitude outside `[-90, 90]`. |
| 7002 | `CodeInvalidLongitude` | Observer longitude outside `[-180, 180]`. |
| 7003 | `CodeInvalidSegmentation` | Segmentation levels empty or not strictly ascending. |
| 7004 | `CodeInvalidMagnitudeRange` | Magnitude range bright bound fainter than the faint bound. |
| 7005 | `CodeInvalidCanvas` | Projection canvas width or height not positive. |
| 7006 | `CodeInvalidHorizonFraction` | Viewport horizon fraction outside `[0, 1]`. |
| 7007 | `CodeInvalidPeakAltitude` | `NormalizedByPeak` peak altitude not positive. |
| 7008 | `CodeInvalidFieldOfView` | Azimuth field of view or arc span not positive. |
| 7009 | `CodeInvalidElevationScale` | `Geometric` elevation degrees-per-pixel not positive. |
| 7010 | `CodeXModeNotAzimuth` | Column-to-azimuth query on a non-azimuth horizontal mode. |
| 7013 | `CodeInvalidBody` | `planet.Body` that is not an observable planet (or is Earth where a target is needed). |

These constants are declared on the root `astronomy` package.

## 🏷️ Matching sentinels

Each error-returning package also exports named sentinels you can match with `errors.Is`, wrapped inside the coded error:

- `earth.ErrInvalidLatitude`, `earth.ErrInvalidLongitude`, `earth.ErrInvalidSegmentation`
- `star.ErrEmptyMagnitudeRange`
- `planet.ErrInvalidLatitude`, `planet.ErrInvalidLongitude`, `planet.ErrInvalidBody`
- `project.ErrInvalidCanvas`, `project.ErrInvalidHorizonFraction`, `project.ErrInvalidPeakAltitude`, `project.ErrInvalidFieldOfView`, `project.ErrInvalidElevationScale`, `project.ErrXModeNotAzimuth`

```go
day, err := earth.NewSunTimes(obs, t)
switch {
case errors.Is(err, earth.ErrInvalidLatitude):
	// out-of-range latitude
case err != nil:
	// some other coded error
	code := apperr.Code(err) // e.g. 7002
	_ = code
default:
	_ = day
}
```

## 📇 The registry

```go
func astronomy.Errors() *apperr.Registry
```

`astronomy.Errors()` returns the module's go-apperr registry: the stable code table plus the presentation and logging policy around it. A consumer renders a coded error at its edge with `Registry.Present` (or `PresentContext`, which also drives the wired go-log logger), looks a code up with `Registry.Describe`, or emits the whole table as Markdown with `Registry.Markdown`. The registry is built once and is safe for concurrent use.

The library is quiet by default — it never logs on its own. The registry has go-log wired as go-apperr's logger, so a service that presents a coded error with `Registry.PresentContext` gets a structured line correlated with its OpenTelemetry trace when one is active. OpenTelemetry is a transitive dependency only; this library never starts a tracer or exporter, so it stays dormant until the surrounding service turns it on.

Full godoc: [pkg.go.dev/github.com/Bugs5382/go-astronomy](https://pkg.go.dev/github.com/Bugs5382/go-astronomy).
