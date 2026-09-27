package sky_test

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
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// row is one line of a Horizons fixture in testdata: the instant and the
// remaining fields, split on spaces.
type row struct {
	when   time.Time
	fields []string
}

// num parses field i of the row as a number.
func (r row) num(t *testing.T, i int) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(r.fields[i], 64)
	if err != nil {
		t.Fatalf("field %d of %s: %v", i, r.when, err)
	}
	return v
}

// load reads a Horizons fixture from testdata, skipping the header comments.
func load(t *testing.T, name string) []row {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []row
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		when, err := time.Parse(time.RFC3339, p[0])
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, row{when: when, fields: p[1:]})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatalf("%s has no rows", name)
	}
	return out
}

// separation returns the angle between two horizontal directions, in arc
// seconds.
func separation(alt1, az1, alt2, az2 float64) float64 {
	const r = math.Pi / 180
	c := math.Sin(alt1*r)*math.Sin(alt2*r) + math.Cos(alt1*r)*math.Cos(alt2*r)*math.Cos((az1-az2)*r)
	return math.Acos(math.Max(-1, math.Min(1, c))) / r * 3600
}
