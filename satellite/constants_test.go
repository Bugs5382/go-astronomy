package satellite

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
	"math"
	"testing"
)

// TestConstants checks the derived constants against their definitions.
func TestConstants(t *testing.T) {
	t.Parallel()
	if math.Abs(xpdotp-229.1831180523293) > 1e-12 {
		t.Errorf("xpdotp = %v, want 1440 / 2 pi", xpdotp)
	}
	// xke is sqrt(mu / Re^3) in earth radii per minute: 0.0743669161 for WGS-72.
	if math.Abs(wgs72Xke-0.07436691613317342) > 1e-15 {
		t.Errorf("wgs72Xke = %v", wgs72Xke)
	}
	if wgs84A != 6378.137 || math.Abs(1/wgs84F-298.257223563) > 1e-6 {
		t.Errorf("WGS84 a = %v, 1/f = %v", wgs84A, 1/wgs84F)
	}
}
