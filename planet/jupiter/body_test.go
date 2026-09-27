package jupiter_test

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
	"testing"

	"github.com/Bugs5382/go-astronomy/internal/planettest"
	"github.com/Bugs5382/go-astronomy/planet/jupiter"
)

// TestBody checks the package constants and that Planet reports them.
func TestBody(t *testing.T) {
	t.Parallel()
	if jupiter.Name != "jupiter" || jupiter.Planet.Name() != jupiter.Name {
		t.Errorf("name %q, Planet %q", jupiter.Name, jupiter.Planet.Name())
	}
	if jupiter.RadiusKm != 71492 || jupiter.Planet.RadiusKm() != jupiter.RadiusKm {
		t.Errorf("radius %v, Planet %v", jupiter.RadiusKm, jupiter.Planet.RadiusKm())
	}
	if jupiter.NearSunElongation != 9 || jupiter.Planet.NearSunElongation() != jupiter.NearSunElongation {
		t.Errorf("near-Sun limit %v, Planet %v", jupiter.NearSunElongation, jupiter.Planet.NearSunElongation())
	}
}

// TestObserverErrors checks the coded errors for an out-of-range observer.
func TestObserverErrors(t *testing.T) {
	t.Parallel()
	planettest.CheckObserverErrors(t, jupiter.Planet)
}
