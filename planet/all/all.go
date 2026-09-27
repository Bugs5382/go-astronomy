// Package all lists every planet package, for a caller who wants to iterate
// over the planets or pick one by name. Importing it links every planet's
// VSOP87 table; a caller who wants one planet imports that planet's package
// instead and pays for nothing else.
package all

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
	"github.com/Bugs5382/go-astronomy/planet"
	"github.com/Bugs5382/go-astronomy/planet/jupiter"
	"github.com/Bugs5382/go-astronomy/planet/mars"
	"github.com/Bugs5382/go-astronomy/planet/mercury"
	"github.com/Bugs5382/go-astronomy/planet/neptune"
	"github.com/Bugs5382/go-astronomy/planet/saturn"
	"github.com/Bugs5382/go-astronomy/planet/uranus"
	"github.com/Bugs5382/go-astronomy/planet/venus"
)

// planets is every observable planet, in order from the Sun.
var planets = []planet.Body{
	mercury.Planet, venus.Planet, mars.Planet, jupiter.Planet,
	saturn.Planet, uranus.Planet, neptune.Planet,
}

// Planets returns every planet an observer on Earth can look at, Mercury to
// Neptune, in order from the Sun. The slice is a copy.
func Planets() []planet.Body {
	return append([]planet.Body(nil), planets...)
}

// ByName returns the planet with the given lowercase name, such as "mars",
// and true, or false when there is none. Earth is not an observable planet;
// use planet.EarthHeliocentric for its position.
func ByName(name string) (planet.Body, bool) {
	for _, p := range planets {
		if p.Name() == name {
			return p, true
		}
	}
	return nil, false
}
