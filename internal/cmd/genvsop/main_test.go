package main

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

import "testing"

// sample is the head of two blocks of VSOP87D.ear, in the file's own layout.
const sample = ` VSOP87 VERSION D4    EARTH     VARIABLE 1 (LBR)       *T**0    2 TERMS    HELIOCENTRIC DYNAMICAL ECLIPTIC AND EQUINOX OF THE DATE
 4310    1  0  0  0  0  0  0  0  0  0  0  0  0  0.00000000000     1.75347045673     1.75347045673 0.00000000000       0.00000000000
 4310    2  0  0  1  0  0  0  0  0  0  0  0  0 -0.00748171065    -0.03256824823     0.03341656456 4.66925680417    6283.07584999140
 VSOP87 VERSION D4    EARTH     VARIABLE 3 (LBR)       *T**1    1 TERMS    HELIOCENTRIC DYNAMICAL ECLIPTIC AND EQUINOX OF THE DATE
 4331    1  0  0  1  0  0  0  0  0  0  0  0  0  0.00000000000     0.01030186000     0.01030186000 1.10748969588    6283.07584999140
`

func TestParse(t *testing.T) {
	t.Parallel()
	s, err := parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(s[0]) != 1 || len(s[0][0]) != 2 {
		t.Fatalf("L blocks = %v, want one block of two terms", s[0])
	}
	if got := s[0][0][1]; got != (term{0.03341656456, 4.66925680417, 6283.07584999140}) {
		t.Errorf("L0 term 2 = %+v", got)
	}
	if len(s[1]) != 0 {
		t.Errorf("B has %d blocks, want none", len(s[1]))
	}
	if len(s[2]) != 2 || len(s[2][0]) != 0 || len(s[2][1]) != 1 {
		t.Fatalf("R blocks = %v, want an empty R0 and one R1 term", s[2])
	}
}

func TestParseRejectsTermBeforeHeader(t *testing.T) {
	t.Parallel()
	if _, err := parse([]byte(" 4310 1 0 0 0.1 0.2 0.3\n")); err == nil {
		t.Error("want an error for a term line before any header")
	}
}
