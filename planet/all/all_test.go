package all_test

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
	"github.com/Bugs5382/go-astronomy/planet/all"
)

// TestPlanets checks every planet is listed, in order from the Sun.
func TestPlanets(t *testing.T) {
	t.Parallel()
	got := all.Planets()
	if len(got) != len(planettest.Planets) {
		t.Fatalf("%d planets, want %d", len(got), len(planettest.Planets))
	}
	for i, b := range got {
		if b.Name() != planettest.Planets[i] {
			t.Errorf("planet %d is %q, want %q", i, b.Name(), planettest.Planets[i])
		}
	}
	got[0] = nil
	if all.Planets()[0] == nil {
		t.Error("Planets returned the shared slice")
	}
}

// TestByName checks lookup by name, and a miss.
func TestByName(t *testing.T) {
	t.Parallel()
	for _, n := range planettest.Planets {
		b, ok := all.ByName(n)
		if !ok || b.Name() != n {
			t.Errorf("ByName(%q) = %v, %v", n, b, ok)
		}
	}
	if _, ok := all.ByName("pluto"); ok {
		t.Error("ByName(pluto) found something")
	}
	if _, ok := all.ByName("earth"); ok {
		t.Error("Earth is not an observable planet")
	}
}
