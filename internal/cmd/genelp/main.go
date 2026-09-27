// Command genelp generates the truncated ELP 2000-82B lunar series used by
// internal/elp from the 36 files IMCCE publishes. It is a development tool,
// run by hand (go generate in internal/elp); nothing at build or run time
// depends on it.
//
// The theory is M. Chapront-Touze and J. Chapront, "The lunar ephemeris ELP
// 2000", Astronomy and Astrophysics 124, 50 (1983), with the constants fitted
// to the JPL DE200/LE200 integration and the arguments of ELP 2000-85
// (Astronomy and Astrophysics 190, 342, 1988), as distributed by IMCCE with
// the Fortran subroutine ELP82B_2. This generator does at generation time
// what ELP82B_2 does when it reads the files: it applies the corrections of
// the constants to the main problem, folds each term's arguments into a
// polynomial in time, and drops the terms smaller than the truncation level.
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

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://ftp.imcce.fr/pub/ephem/moon/elp82b/"

// digests pins each ELPn file, as downloaded from IMCCE in September 2026.
var digests = [37]string{
	1:  "ae30cbffb83a7bd4582a83a32a322d08a48ba057a4df7bf9dd5df9f06b1688fa",
	2:  "c91e5585b0a9e7bd091304b164ce89a6461acd0e439d47957c890aec1e031e08",
	3:  "862a8e4c8e70ce8b28383be4c9f2e2c025a8f633d7b7a811eb3afdab4ed9f354",
	4:  "f27ea439bf8f4fd35bed31c0a42de5414db07f9fcc3587237f6908891d43d773",
	5:  "6803422481e4decae4a59f89d4f94c7b33af21d293d9bc807d565f51c29b9915",
	6:  "2a6be4d33dfce4cf2351d295b4aade8f34cc476a97747b3eccb12763658e1fd1",
	7:  "35491a0c73ff6bcb136d8f54db89d8df2fb741af43aed9707618e3a925df474d",
	8:  "f3e7f4c851e7f9ac1a0556fe7e685e44612f3dca317ab605eb35f565251e020e",
	9:  "574347346363df52c7602f56747b790e9cbe60152127d24451c6a1633bc79f0e",
	10: "dbd82ddc6064e4cc7b4f08fa27b2fcb48f82456ad36a850a0d3ddae098c3e2e6",
	11: "0ad7a914c9f98008a648881783c9dd4a14692ec14e2e1bfce4709c68d17bd659",
	12: "8ed7be0ab70f4ffae6b1f711cc4e915257ae5e269fbbfc5a7060f7e952728ba8",
	13: "643295b3894023b4b1bd6ee2b0ecf5d3ff23d703baccc6302caa05fa8b84f76c",
	14: "b59d8b9bbef282f2bead538d6906781257a7fb5b8699bb6adcacb070a76f1e89",
	15: "17ab0d521c178187a5de4847b6696fbcb7a55d77d776568a0c16f33fd3be342a",
	16: "2bef867d8aad4075bc2711559cf1bc42757501bb10c307ff121152bddd344a66",
	17: "6cf0746d034ac75ed60d4d16ed0de790fe5b7aad9df7b462091c38020e1b1bfc",
	18: "b1d93931f6016023c83354cd54a2614978de3b6bc5b7234537d461200f7f4753",
	19: "dd0b0bd5f5c354683f035ee8a09d82e9c138baaf27758ca311d07c76983bdd2e",
	20: "0f1d571879dc9b1a6f7698b403bef26ab151c3ec2ed42cbdec5b297cb0464a8e",
	21: "1546d0e8af01f759f9dfb7a3bf4334a3e579f59e5c22edab29d660dede2ed4b4",
	22: "44263bb254c2b6c0df963bddfd1ecfd01950668fd913f9e6ba6da6ff729f3e41",
	23: "38917cf2cbe0f9afcd271444a47066b098f9a8792836ec85dac359f4c11b9464",
	24: "ca67e5db5933887130709767c1eb3ac009e4bce596978bad7995c416ee71c59f",
	25: "fd0cb03d496cbf23bf7bebef40aa09019c6a50072fc7fd2573645f26a56ab635",
	26: "d5b2a33974b099448a35536987b2f08aff5b11d5801ffa55c87ae8a1d26e9f4f",
	27: "648379d85e1753cc37bc3852b63899d414a05d3aea1c8621dc44e9fdfff234f1",
	28: "0785b8e002887799bc303e8be1abd71c37ae9de671bffbdeb8f50998f892f18e",
	29: "816fd1a94b1cb4e2e5e6cec72971f27ad0f2bf987735d184586fdadd033c3d02",
	30: "cff5ab4c84a6a36855e5b2e2f47e1e0e1d605e789ff2954755dc64d188067a13",
	31: "c2fc53c2442c1b61404991f31c859cc5eb300eb66bb764e1f16c70fe8d199dc3",
	32: "7a07397b63d1ade0909c12be9024632de6ff27fd9e8e410e5f97f821d2390a60",
	33: "459ea9eff9a9d7b5c224245f5edb113060174b4991d3adf7d27079719bcf2339",
	34: "b83178e98bd33e8f26ef6662e03455761ffac7dae399ad1ccb4bf028c5f0e774",
	35: "692d1752a7ea28c7157dbf694750af6ea2cbf74c744c4b792a7e8bf1bb5ad7d7",
	36: "1f8eec292def5ceb4fff9a09ca678bd81e9cdc23ff7bdc802f2281c837357bda",
}

// Constants of ELP82B_2.
const (
	rad = 648000 / math.Pi // arc seconds per radian
	deg = math.Pi / 180
	ath = 384747.9806743165 // the distance scale of the series, km

	am    = 0.074801329518
	alpha = 0.002571881335
)

// term is one term of the series for one coordinate: amplitude times the
// power tpow of T, times the sine of a polynomial in T. Amplitudes are in arc
// seconds for longitude and latitude and in km for distance; T is in Julian
// centuries of TDB from J2000.
type term struct {
	coord int // 0 longitude, 1 latitude, 2 distance
	tpow  int // the power of T the amplitude carries
	a     float64
	phase []float64 // phase coefficients of T^0, T^1, ...
}

// args holds the Delaunay, lunar, and planetary arguments as polynomials in
// T, in radians, with the corrections of the DE200/LE200 fit.
type args struct {
	del                      [5][5]float64 // del[1..4][power]
	zeta                     [2]float64
	p                        [9][2]float64
	delnu, dele, delg, delnp float64
	delep, dtasm             float64
	w1                       [5]float64
}

func newArgs() args {
	var w [4][5]float64
	var eart, peri [5]float64
	w[1][0] = (218 + 18.0/60 + 59.95571/3600) * deg
	w[2][0] = (83 + 21.0/60 + 11.67475/3600) * deg
	w[3][0] = (125 + 2.0/60 + 40.39816/3600) * deg
	eart[0] = (100 + 27.0/60 + 59.22059/3600) * deg
	peri[0] = (102 + 56.0/60 + 14.42753/3600) * deg
	w[1][1] = 1732559343.73604 / rad
	w[2][1] = 14643420.2632 / rad
	w[3][1] = -6967919.3622 / rad
	eart[1] = 129597742.2758 / rad
	peri[1] = 1161.2283 / rad
	w[1][2] = -5.8883 / rad
	w[2][2] = -38.2776 / rad
	w[3][2] = 6.3622 / rad
	eart[2] = -0.0202 / rad
	peri[2] = 0.5327 / rad
	w[1][3] = 0.6604e-2 / rad
	w[2][3] = -0.45047e-1 / rad
	w[3][3] = 0.7625e-2 / rad
	eart[3] = 0.9e-5 / rad
	peri[3] = -0.138e-3 / rad
	w[1][4] = -0.3169e-4 / rad
	w[2][4] = 0.21301e-3 / rad
	w[3][4] = -0.3586e-4 / rad
	eart[4] = 0.15e-6 / rad

	var a args
	a.w1 = w[1]
	a.p[1] = [2]float64{(252 + 15.0/60 + 3.25986/3600) * deg, 538101628.68898 / rad}
	a.p[2] = [2]float64{(181 + 58.0/60 + 47.28305/3600) * deg, 210664136.43355 / rad}
	a.p[3] = [2]float64{eart[0], eart[1]}
	a.p[4] = [2]float64{(355 + 25.0/60 + 59.78866/3600) * deg, 68905077.59284 / rad}
	a.p[5] = [2]float64{(34 + 21.0/60 + 5.34212/3600) * deg, 10925660.42861 / rad}
	a.p[6] = [2]float64{(50 + 4.0/60 + 38.89694/3600) * deg, 4399609.65932 / rad}
	a.p[7] = [2]float64{(314 + 3.0/60 + 18.01841/3600) * deg, 1542481.19393 / rad}
	a.p[8] = [2]float64{(304 + 20.0/60 + 55.19575/3600) * deg, 786550.32074 / rad}

	a.delnu = 0.55604 / rad / w[1][1]
	a.dele = 0.01789 / rad
	a.delg = -0.08066 / rad
	a.delnp = -0.06424 / rad / w[1][1]
	a.delep = -0.12879 / rad
	a.dtasm = 2 * alpha / (3 * am)

	for i := range 5 {
		a.del[1][i] = w[1][i] - eart[i]
		a.del[4][i] = w[1][i] - w[3][i]
		a.del[3][i] = w[1][i] - w[2][i]
		a.del[2][i] = eart[i] - peri[i]
	}
	a.del[1][0] += math.Pi
	precess := 5029.0966 / rad
	a.zeta = [2]float64{w[1][0], w[1][1] + precess}
	return a
}

func main() {
	dir := flag.String("in", "", "read ELP1 to ELP36 from this directory instead of downloading them")
	out := flag.String("out", "", "output Go file")
	prec := flag.Float64("prec", 5e-8, "truncation level in radians (for distance, times the mean distance), as ELP82B_2's prec")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("genelp: ")
	if *out == "" {
		log.Fatal("-out is required")
	}

	a := newArgs()
	var terms []term
	total := 0
	for n := 1; n <= 36; n++ {
		raw, origin, err := load(n, *dir)
		if err != nil {
			log.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if got := hex.EncodeToString(sum[:]); got != digests[n] {
			log.Fatalf("%s: SHA-256 %s, pinned %s", origin, got, digests[n])
		}
		ts, err := parseFile(n, raw, a)
		if err != nil {
			log.Fatalf("%s: %v", origin, err)
		}
		total += len(ts)
		for _, t := range ts {
			if keep(t, *prec) {
				terms = append(terms, t)
			}
		}
	}
	log.Printf("kept %d of %d terms at prec %g rad", len(terms), total, *prec)

	code, err := render(terms, *prec, len(terms), total, a.w1)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, code, 0o600); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", *out)
}

// keep applies ELP82B_2's truncation: an amplitude below prec (in arc seconds
// for the angles, in units of the mean distance for the distance) is dropped.
func keep(t term, prec float64) bool {
	limit := prec*rad - 1e-12
	if t.coord == 2 {
		limit = prec * ath
	}
	return math.Abs(t.a) >= limit
}

// load reads ELPn from dir, or downloads it.
func load(n int, dir string) ([]byte, string, error) {
	name := fmt.Sprintf("ELP%d", n)
	if dir != "" {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p) // #nosec G304 -- a directory the developer passes on the command line.
		return b, p, err
	}
	u := baseURL + name
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(u) // #nosec G107 -- the URL is built from pinned constants.
	if err != nil {
		return nil, u, fmt.Errorf("get %s: %w", u, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, u, fmt.Errorf("get %s: %s", u, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	return b, u, err
}

// field returns the text of a fixed-width field, 0-based [from, to), trimmed.
func field(line string, from, to int) string {
	if to > len(line) {
		to = len(line)
	}
	if from >= to {
		return ""
	}
	return strings.TrimSpace(line[from:to])
}

func ints(line string, from, n int) ([]int, error) {
	out := make([]int, n)
	for i := range out {
		v, err := strconv.Atoi(field(line, from+3*i, from+3*i+3))
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func float(line string, from, to int) (float64, error) {
	return strconv.ParseFloat(field(line, from, to), 64)
}

// parseFile reads one ELPn file, after its title line, into terms, doing what
// ELP82B_2 does on reading: the formats are (4i3,2x,f13.5,6(2x,f10.2)) for
// the main problem, (5i3,1x,f9.5,1x,f9.5) for the Earth and Moon figures,
// the tides, relativity, and solar eccentricity, and (11i3,1x,f9.5,1x,f9.5)
// for the planetary perturbations.
func parseFile(n int, raw []byte, a args) ([]term, error) {
	itab := (n + 2) / 3
	coord := (n - 1) % 3
	tpow := 0
	switch itab {
	case 3, 5, 7, 9:
		tpow = 1
	case 12:
		tpow = 2
	}
	var out []term
	sc := bufio.NewScanner(bytes.NewReader(raw))
	first := true
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		if first {
			first = false // the title line
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		t, err := parseLine(n, coord, tpow, text, a)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		out = append(out, t)
	}
	return out, sc.Err()
}

func parseLine(n, coord, tpow int, text string, a args) (term, error) {
	t := term{coord: coord, tpow: tpow}
	switch {
	case n <= 3:
		ilu, err := ints(text, 0, 4)
		if err != nil {
			return t, err
		}
		var coef [7]float64
		if coef[0], err = float(text, 14, 27); err != nil {
			return t, err
		}
		for k := 1; k < 7; k++ {
			if coef[k], err = float(text, 27+12*(k-1)+2, 27+12*(k-1)+12); err != nil {
				return t, err
			}
		}
		tgv := coef[1] + a.dtasm*coef[5]
		if n == 3 {
			coef[0] -= 2 * coef[0] * a.delnu / 3
		}
		t.a = coef[0] + tgv*(a.delnp-am*a.delnu) + coef[2]*a.delg + coef[3]*a.dele + coef[4]*a.delep
		t.phase = make([]float64, 5)
		for k := range 5 {
			for i := range 4 {
				t.phase[k] += float64(ilu[i]) * a.del[i+1][k]
			}
		}
		if coord == 2 {
			t.phase[0] += math.Pi / 2 // the distance series is in cosines
		}
	case n <= 9 || n >= 22:
		iz, err := strconv.Atoi(field(text, 0, 3))
		if err != nil {
			return t, err
		}
		ilu, err := ints(text, 3, 4)
		if err != nil {
			return t, err
		}
		pha, err := float(text, 16, 25)
		if err != nil {
			return t, err
		}
		if t.a, err = float(text, 26, 35); err != nil {
			return t, err
		}
		t.phase = make([]float64, 2)
		for k := range 2 {
			if k == 0 {
				t.phase[k] = pha * deg
			}
			t.phase[k] += float64(iz) * a.zeta[k]
			for i := range 4 {
				t.phase[k] += float64(ilu[i]) * a.del[i+1][k]
			}
		}
	default:
		ipla, err := ints(text, 0, 11)
		if err != nil {
			return t, err
		}
		pha, err := float(text, 34, 43)
		if err != nil {
			return t, err
		}
		if t.a, err = float(text, 44, 53); err != nil {
			return t, err
		}
		t.phase = make([]float64, 2)
		for k := range 2 {
			if k == 0 {
				t.phase[k] = pha * deg
			}
			if n < 16 {
				t.phase[k] += float64(ipla[8])*a.del[1][k] + float64(ipla[9])*a.del[3][k] + float64(ipla[10])*a.del[4][k]
				for i := range 8 {
					t.phase[k] += float64(ipla[i]) * a.p[i+1][k]
				}
			} else {
				for i := range 4 {
					t.phase[k] += float64(ipla[i+7]) * a.del[i+1][k]
				}
				for i := range 7 {
					t.phase[k] += float64(ipla[i]) * a.p[i+1][k]
				}
			}
		}
	}
	return t, nil
}

// render writes the generated table.
func render(terms []term, prec float64, kept, total int, w1 [5]float64) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("// Code generated by genelp. DO NOT EDIT.\n\npackage elp\n\n")
	b.WriteString(license)
	fmt.Fprintf(&b, "\n// The ELP 2000-82B series, truncated at %g radians (ELP82B_2's prec): %d of %d terms.\n", prec, kept, total)
	fmt.Fprintf(&b, "// Source: %sELP1 to ELP36, SHA-256 pinned in internal/cmd/genelp.\n", baseURL)
	b.WriteString("// The DE200/LE200 corrections of the constants are applied.\n\n")
	fmt.Fprintf(&b, "// meanLongitude is W1, the Moon's mean longitude, in radians: a polynomial in T.\n")
	fmt.Fprintf(&b, "var meanLongitude = [5]float64{%s, %s, %s, %s, %s}\n\n", num(w1[0]), num(w1[1]), num(w1[2]), num(w1[3]), num(w1[4]))
	b.WriteString("// series are the longitude and latitude (arc seconds) and distance (km).\n")
	b.WriteString("var series = [3]coordinate{\n")
	for c, name := range []string{"longitude", "latitude", "distance"} {
		fmt.Fprintf(&b, "// %s\n{\nmain: []mainTerm{\n", name)
		for _, t := range terms {
			if t.coord == c && len(t.phase) == 5 {
				fmt.Fprintf(&b, "{%s, [5]float64{%s, %s, %s, %s, %s}},\n", num(t.a), num(t.phase[0]), num(t.phase[1]), num(t.phase[2]), num(t.phase[3]), num(t.phase[4]))
			}
		}
		b.WriteString("},\nperturbations: [3][]term{\n")
		for p := range 3 {
			fmt.Fprintf(&b, "// T^%d\n{\n", p)
			for _, t := range terms {
				if t.coord == c && len(t.phase) == 2 && t.tpow == p {
					fmt.Fprintf(&b, "{%s, %s, %s},\n", num(t.a), num(t.phase[0]), num(t.phase[1]))
				}
			}
			b.WriteString("},\n")
		}
		b.WriteString("},\n},\n")
	}
	b.WriteString("}\n")
	return format.Source(b.Bytes())
}

func num(x float64) string { return strconv.FormatFloat(x, 'g', -1, 64) }

// license is the header every Go source file in this module carries.
const license = `/*
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

`
