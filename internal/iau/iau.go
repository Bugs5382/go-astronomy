// Package iau holds the IAU Working Group on Cartographic Coordinates and
// Rotational Elements (WGCCRE) models of the Sun, the planets, and the Moon:
// the direction of each body's north pole and the angle of its prime meridian
// at an instant, and its reference ellipsoid. It is the 2015 report,
// Archinal et al., "Report of the IAU Working Group on Cartographic
// Coordinates and Rotational Elements: 2015", Celestial Mechanics and
// Dynamical Astronomy 130, 22 (2018), whose Moon model is unchanged from the
// 2009 report.
//
// The pole is (Alpha0, Delta0) in the ICRF (the J2000 mean equator, to the
// accuracy here), and W is the prime meridian's angle east along the body's
// equator from the node of that equator on the ICRF equator. Angles are in
// degrees; d is days and T Julian centuries of TDB from J2000.0.
package iau

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

import "math"

// j2000 is the Julian date of J2000.0.
const j2000 = 2451545.0

// Orientation is a body's pole and prime meridian at one instant, in degrees.
type Orientation struct {
	Alpha0, Delta0, W float64
}

// Model is one body's rotational elements and reference ellipsoid.
type Model struct {
	// Name is the body's lowercase name.
	Name string
	// EquatorialKm and PolarKm are the reference ellipsoid's radii.
	EquatorialKm, PolarKm float64
	// RotationDays is the sidereal rotation period, negative for a
	// retrograde spin (Venus, Uranus), from the rate of W.
	RotationDays float64
	// OrbitDays is the sidereal period of the body's orbit around the Sun;
	// for the Moon it is the Earth's.
	OrbitDays float64
	// orient evaluates the elements at d days and T centuries from J2000.
	orient func(d, t float64) Orientation
}

// At returns the body's orientation at the Julian ephemeris day jde.
func (m *Model) At(jde float64) Orientation {
	d := jde - j2000
	o := m.orient(d, d/36525)
	o.W = math.Mod(o.W, 360)
	if o.W < 0 {
		o.W += 360
	}
	return o
}

// SolarDays returns the length of the body's mean solar day in days: the
// period of the Sun's return to the same meridian, from the rotation and the
// orbit. It is the synodic month for the Moon, 116.75 days for Venus, and
// 176 days for Mercury.
func (m *Model) SolarDays() float64 {
	return math.Abs(1 / (1/m.RotationDays - 1/m.OrbitDays))
}

func sin(deg float64) float64 { return math.Sin(deg * math.Pi / 180) }
func cos(deg float64) float64 { return math.Cos(deg * math.Pi / 180) }

// Sun is the IAU model of the Sun.
var Sun = &Model{
	Name: "sun", EquatorialKm: 695700, PolarKm: 695700, RotationDays: 360 / 14.1844, OrbitDays: math.Inf(1),
	orient: func(d, _ float64) Orientation {
		return Orientation{Alpha0: 286.13, Delta0: 63.87, W: 84.176 + 14.1844000*d}
	},
}

// Mercury is the IAU model of Mercury, with its libration in W.
var Mercury = &Model{
	Name: "mercury", EquatorialKm: 2440.53, PolarKm: 2440.53, RotationDays: 360 / 6.1385108, OrbitDays: 87.9691,
	orient: func(d, t float64) Orientation {
		m1 := 174.7910857 + 4.092335*d
		m2 := 349.5821714 + 8.184670*d
		m3 := 164.3732571 + 12.277005*d
		m4 := 339.1643429 + 16.369340*d
		m5 := 153.9554286 + 20.461675*d
		return Orientation{
			Alpha0: 281.0103 - 0.0328*t,
			Delta0: 61.4155 - 0.0049*t,
			W: 329.5988 + 6.1385108*d + 0.01067257*sin(m1) - 0.00112309*sin(m2) - 0.00011040*sin(m3) -
				0.00002539*sin(m4) - 0.00000571*sin(m5),
		}
	},
}

// Venus is the IAU model of Venus. Its spin is retrograde, so W decreases.
var Venus = &Model{
	Name: "venus", EquatorialKm: 6051.8, PolarKm: 6051.8, RotationDays: -360 / 1.4813688, OrbitDays: 224.701,
	orient: func(d, _ float64) Orientation {
		return Orientation{Alpha0: 272.76, Delta0: 67.16, W: 160.20 - 1.4813688*d}
	},
}

// Earth is the IAU model of the Earth. It carries no nutation, so it is good
// to about 20 arc seconds; the earth package uses the full sidereal time.
var Earth = &Model{
	Name: "earth", EquatorialKm: 6378.1366, PolarKm: 6356.7519, RotationDays: 360 / 360.9856235, OrbitDays: 365.256363,
	orient: func(d, t float64) Orientation {
		return Orientation{Alpha0: 0.00 - 0.641*t, Delta0: 90.00 - 0.557*t, W: 190.147 + 360.9856235*d}
	},
}

// Moon is the IAU model of the Moon, an approximation of the mean Earth and
// polar axis frame with thirteen periodic terms. Its orbit around the Sun is
// the Earth's.
var Moon = &Model{
	Name: "moon", EquatorialKm: 1737.4, PolarKm: 1737.4, RotationDays: 360 / 13.17635815, OrbitDays: 365.256363,
	orient: func(d, t float64) Orientation {
		e1 := 125.045 - 0.0529921*d
		e2 := 250.089 - 0.1059842*d
		e3 := 260.008 + 13.0120009*d
		e4 := 176.625 + 13.3407154*d
		e5 := 357.529 + 0.9856003*d
		e6 := 311.589 + 26.4057084*d
		e7 := 134.963 + 13.0649930*d
		e8 := 276.617 + 0.3287146*d
		e9 := 34.226 + 1.7484877*d
		e10 := 15.134 - 0.1589763*d
		e11 := 119.743 + 0.0036096*d
		e12 := 239.961 + 0.1643573*d
		e13 := 25.053 + 12.9590088*d
		return Orientation{
			Alpha0: 269.9949 + 0.0031*t - 3.8787*sin(e1) - 0.1204*sin(e2) + 0.0700*sin(e3) - 0.0172*sin(e4) +
				0.0072*sin(e6) - 0.0052*sin(e10) + 0.0043*sin(e13),
			Delta0: 66.5392 + 0.0130*t + 1.5419*cos(e1) + 0.0239*cos(e2) - 0.0278*cos(e3) + 0.0068*cos(e4) -
				0.0029*cos(e6) + 0.0009*cos(e7) + 0.0008*cos(e10) - 0.0009*cos(e13),
			W: 38.3213 + 13.17635815*d - 1.4e-12*d*d + 3.5610*sin(e1) + 0.1208*sin(e2) - 0.0642*sin(e3) +
				0.0158*sin(e4) + 0.0252*sin(e5) - 0.0066*sin(e6) - 0.0047*sin(e7) - 0.0046*sin(e8) +
				0.0028*sin(e9) + 0.0052*sin(e10) + 0.0040*sin(e11) + 0.0019*sin(e12) - 0.0044*sin(e13),
		}
	},
}

// Mars is the IAU 2015 model of Mars, with the periodic terms of Kuchynka et
// al. (2014).
var Mars = &Model{
	Name: "mars", EquatorialKm: 3396.19, PolarKm: 3376.20, RotationDays: 360 / 350.891982443297, OrbitDays: 686.980,
	orient: func(d, t float64) Orientation {
		return Orientation{
			Alpha0: 317.269202 - 0.10927547*t +
				0.000068*sin(198.991226+19139.4819985*t) + 0.000238*sin(226.292679+38280.8511281*t) +
				0.000052*sin(249.663391+57420.7251593*t) + 0.000009*sin(266.183510+76560.6367950*t) +
				0.419057*sin(79.398797+0.5042615*t),
			Delta0: 54.432516 - 0.05827105*t +
				0.000051*cos(122.433576+19139.9407476*t) + 0.000141*cos(43.058401+38280.8753272*t) +
				0.000031*cos(57.663379+57420.7517205*t) + 0.000005*cos(79.476401+76560.6495004*t) +
				1.591274*cos(166.325722+0.5042615*t),
			W: 176.049863 + 350.891982443297*d +
				0.000145*sin(129.071773+19140.0328244*t) + 0.000157*sin(36.352167+38281.0473591*t) +
				0.000040*sin(56.668646+57420.9295360*t) + 0.000001*sin(67.364003+76560.2552215*t) +
				0.000001*sin(104.792680+95700.4387578*t) + 0.584542*sin(95.391654+0.5042615*t),
		}
	},
}

// Jupiter is the IAU 2015 model of Jupiter; W is System III.
var Jupiter = &Model{
	Name: "jupiter", EquatorialKm: 71492, PolarKm: 66854, RotationDays: 360 / 870.5360000, OrbitDays: 4332.59,
	orient: func(d, t float64) Orientation {
		ja := 99.360714 + 4850.4046*t
		jb := 175.895369 + 1191.9605*t
		jc := 300.323162 + 262.5475*t
		jd := 114.012305 + 6070.2476*t
		je := 49.511251 + 64.3000*t
		return Orientation{
			Alpha0: 268.056595 - 0.006499*t + 0.000117*sin(ja) + 0.000938*sin(jb) + 0.001432*sin(jc) +
				0.000030*sin(jd) + 0.002150*sin(je),
			Delta0: 64.495303 + 0.002413*t + 0.000050*cos(ja) + 0.000404*cos(jb) + 0.000617*cos(jc) -
				0.000013*cos(jd) + 0.000926*cos(je),
			W: 284.95 + 870.5360000*d,
		}
	},
}

// Saturn is the IAU model of Saturn.
var Saturn = &Model{
	Name: "saturn", EquatorialKm: 60268, PolarKm: 54364, RotationDays: 360 / 810.7939024, OrbitDays: 10759.22,
	orient: func(d, t float64) Orientation {
		return Orientation{Alpha0: 40.589 - 0.036*t, Delta0: 83.537 - 0.004*t, W: 38.90 + 810.7939024*d}
	},
}

// Uranus is the IAU model of Uranus. Its spin is retrograde, so W decreases.
var Uranus = &Model{
	Name: "uranus", EquatorialKm: 25559, PolarKm: 24973, RotationDays: -360 / 501.1600928, OrbitDays: 30685.4,
	orient: func(d, _ float64) Orientation {
		return Orientation{Alpha0: 257.311, Delta0: -15.175, W: 203.81 - 501.1600928*d}
	},
}

// Neptune is the IAU model of Neptune.
var Neptune = &Model{
	Name: "neptune", EquatorialKm: 24764, PolarKm: 24341, RotationDays: 360 / 541.1397757, OrbitDays: 60189,
	orient: func(d, t float64) Orientation {
		n := 357.85 + 52.316*t
		return Orientation{Alpha0: 299.36 + 0.70*sin(n), Delta0: 43.46 - 0.51*cos(n), W: 249.978 + 541.1397757*d - 0.48*sin(n)}
	},
}

// ByName returns the model of the named body, or nil.
func ByName(name string) *Model {
	for _, m := range []*Model{Sun, Mercury, Venus, Earth, Moon, Mars, Jupiter, Saturn, Uranus, Neptune} {
		if m.Name == name {
			return m
		}
	}
	return nil
}
