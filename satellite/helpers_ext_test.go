package satellite_test

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

// Fixtures and helpers shared by the external tests.

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Bugs5382/go-astronomy/satellite"
)

// The synthetic ISS element set the Python and Skyfield fixtures were built
// from (see testdata/iss_crosscheck.py).
const (
	iss1 = "1 25544U 98067A   26269.51782528  .00016717  00000-0  30306-3 0  9990"
	iss2 = "2 25544  51.6416 247.4627 0006703 130.5360 325.0288 15.49450470 12348"
)

func iss(t *testing.T) satellite.Elements {
	t.Helper()
	e, err := satellite.ParseTLE(iss1, iss2)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func fields(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out [][]string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.Fields(line))
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func num(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
