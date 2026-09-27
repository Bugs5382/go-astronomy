package earth

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

import (
	"errors"
	"math"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// HorizonRefraction is the standard refraction at the horizon, 34 arc
// minutes, in degrees. It is the refraction part of HorizonAltitude (the rest
// is the Sun's semidiameter) and of the Moon's and planets' rise and set
// altitudes, and it is the part the air at the observer's height scales.
const HorizonRefraction = 34.0 / 60

// Reference conditions of the refraction model: Bennett's formula, and so
// Refraction and the 34 arc minutes at the horizon, hold for air at this
// pressure and temperature. MeasuredAtmosphere scales from them.
const (
	// RefractionReferencePressure is the reference pressure, in hectopascals
	// (millibars).
	RefractionReferencePressure = 1010.0
	// RefractionReferenceTemperature is the reference temperature, in degrees
	// Celsius.
	RefractionReferenceTemperature = 10.0
)

// ErrInvalidAtmosphere is the cause when a measured Atmosphere has a pressure
// that is not a positive finite number or a temperature that is not finite or
// at or below -273 C; the coded error carries astronomy.CodeInvalidAtmosphere.
var ErrInvalidAtmosphere = errors.New("earth: atmosphere pressure or temperature is not physical")

// Atmosphere is the air the refraction passes through. The zero value is
// StandardAtmosphere, which takes the air at the observer's height from the
// ISA standard atmosphere. MeasuredAtmosphere uses a caller's reading of
// pressure and temperature instead.
//
// Refraction bends light in proportion to the density of the air, so an
// Atmosphere reduces to one number, Factor: the refraction it gives relative
// to the sea-level refraction the library's thresholds are built on.
type Atmosphere struct {
	measured     bool
	pressureHPa  float64
	temperatureC float64
}

// StandardAtmosphere is the default air: the ISA standard atmosphere at the
// observer's height. Its factor is exactly 1 at sea level, so sea-level
// answers are those of the unscaled refraction.
var StandardAtmosphere = Atmosphere{}

// MeasuredAtmosphere returns the air of a measured station pressure, in
// hectopascals (millibars), and temperature, in degrees Celsius. Its factor is
// the absolute P/1010 * 283/(273+T) of Meeus (Astronomical Algorithms, ch. 16)
// and does not depend on the observer's height: the reading already is the air
// where the observer stands. Use a station pressure, not one reduced to sea
// level, which is what most weather reports give.
func MeasuredAtmosphere(pressureHPa, temperatureC float64) Atmosphere {
	return Atmosphere{measured: true, pressureHPa: pressureHPa, temperatureC: temperatureC}
}

// Measured reports the pressure (hPa) and temperature (C) of a measured
// atmosphere and true, or zeros and false for StandardAtmosphere.
func (a Atmosphere) Measured() (pressureHPa, temperatureC float64, ok bool) {
	return a.pressureHPa, a.temperatureC, a.measured
}

// Err returns the coded ErrInvalidAtmosphere for measured air that cannot
// exist: a pressure that is not a positive finite number, or a temperature
// that is not finite or at or below -273 C. It is nil for StandardAtmosphere
// and for every physical reading.
func (a Atmosphere) Err() error {
	if !a.measured {
		return nil
	}
	p, c := a.pressureHPa, a.temperatureC
	if !(p > 0) || math.IsInf(p, 0) || math.IsNaN(c) || math.IsInf(c, 0) || !(273+c > 0) {
		return apperr.Coded(astronomy.CodeInvalidAtmosphere, ErrInvalidAtmosphere)
	}
	return nil
}

// Factor returns the refraction of this air relative to the sea-level
// refraction the library's thresholds assume.
//
// For StandardAtmosphere it is the ISA density ratio at height h,
// P(h)/P0 * T0/T(h): exactly 1 at sea level, 0.86 at 1524 m (5000 ft), 0.31 at
// 10668 m (35000 ft), 0.16 at 15 km, and above 1 below sea level. The model
// follows the ISA layers to 84.9 km and thins isothermally above. For a
// measured atmosphere it is P/1010 * 283/(273+T), whatever h.
//
// Invalid input (a NaN or infinite height, or measured air Err rejects) gives
// 1, no correction; the functions that validate reject it instead.
func (a Atmosphere) Factor(h astronomy.Height) float64 {
	if a.measured {
		if a.Err() != nil {
			return 1
		}
		return a.pressureHPa / RefractionReferencePressure * (273 + RefractionReferenceTemperature) / (273 + a.temperatureC)
	}
	if h.Err() != nil {
		return 1
	}
	return isaDensityRatio(h.Meters())
}

// Refraction returns the atmospheric refraction, in degrees, at the given
// apparent altitude for an observer at height h in this air: Refraction
// scaled by Factor.
func (a Atmosphere) Refraction(apparentAltDeg float64, h astronomy.Height) float64 {
	f := a.Factor(h)
	if f == 1 {
		return Refraction(apparentAltDeg)
	}
	return Refraction(apparentAltDeg) * f
}

// HorizonAltitudeAt returns the geometric altitude of the Sun's centre at
// sunrise and sunset for an observer at height h in the standard atmosphere:
// HorizonAltitude, raised by the part of the 34 arc minutes of horizon
// refraction the thinner air does not give, and lowered by the dip of the sea
// horizon. It is HorizonAltitude exactly at sea level.
//
// The two corrections pull against each other. The dip, 1.76 * sqrt(h) arc minutes, already
// carries the refraction of the line of sight down to the sea horizon; the
// scaled refraction is the lift of the Sun itself. At Denver the dip lowers
// the horizon by 1.18 degrees and the thinner air takes back 0.08.
func HorizonAltitudeAt(h astronomy.Height) float64 {
	return HorizonAltitude + refractionLoss(StandardAtmosphere.Factor(h)) - HorizonDip(h)
}

// refractionLoss is the part of the horizon refraction, in degrees, that air
// with factor f does not give: how far a refracted horizon threshold rises.
// It is negative for denser air, which lowers the threshold.
func refractionLoss(f float64) float64 {
	if f == 1 {
		return 0
	}
	return (1 - f) * HorizonRefraction
}

// ISA constants (ICAO Doc 7488, US Standard Atmosphere 1976).
const (
	isaSeaLevelTemperature = 288.15    // K
	isaEarthRadius         = 6356766.0 // m, for geopotential height
	// isaHydrostatic is g0 * M / R*, in kelvin per metre.
	isaHydrostatic = 9.80665 * 0.0289644 / 8.31432
	// isaFloor bounds the height below: the standard tables start at -5 km,
	// and far below that the geopotential conversion loses meaning.
	isaFloor = -5000.0
)

// isaLayer is one layer of the standard atmosphere: its base geopotential
// height (m), its base temperature (K), its lapse rate (K/m), and its base
// pressure relative to sea level, filled in by init.
type isaLayer struct {
	base, temperature, lapse, pressure float64
}

// isaLayers are the seven ISA layers to 84852 m geopotential (86 km
// geometric). Above the last the air is held isothermal.
var isaLayers = []isaLayer{
	{base: 0, temperature: 288.15, lapse: -0.0065},
	{base: 11000, temperature: 216.65, lapse: 0},
	{base: 20000, temperature: 216.65, lapse: 0.001},
	{base: 32000, temperature: 228.65, lapse: 0.0028},
	{base: 47000, temperature: 270.65, lapse: 0},
	{base: 51000, temperature: 270.65, lapse: -0.0028},
	{base: 71000, temperature: 214.65, lapse: -0.002},
	{base: 84852, temperature: 186.946, lapse: 0},
}

func init() {
	isaLayers[0].pressure = 1
	for i := 1; i < len(isaLayers); i++ {
		below := isaLayers[i-1]
		isaLayers[i].pressure = below.pressure * layerPressure(below, isaLayers[i].base)
	}
}

// layerPressure returns the pressure at geopotential height hp inside layer l,
// relative to the layer's base pressure: the barometric formula for a layer
// with a lapse rate, and the exponential for an isothermal one.
func layerPressure(l isaLayer, hp float64) float64 {
	if l.lapse == 0 {
		return math.Exp(-isaHydrostatic * (hp - l.base) / l.temperature)
	}
	t := l.temperature + l.lapse*(hp-l.base)
	return math.Pow(t/l.temperature, -isaHydrostatic/l.lapse)
}

// isaDensityRatio returns the ISA density at geometric height m metres
// relative to sea level, P/P0 * T0/T. It is exactly 1 at sea level.
func isaDensityRatio(m float64) float64 {
	if m == 0 {
		return 1
	}
	m = math.Max(m, isaFloor)
	hp := isaEarthRadius * m / (isaEarthRadius + m) // geopotential height
	l := isaLayers[0]
	for _, next := range isaLayers[1:] {
		if hp < next.base {
			break
		}
		l = next
	}
	t := l.temperature + l.lapse*(hp-l.base)
	p := l.pressure * layerPressure(l, hp)
	return p * isaSeaLevelTemperature / t
}
