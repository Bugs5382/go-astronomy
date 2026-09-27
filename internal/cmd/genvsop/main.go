// Command genvsop generates the truncated VSOP87 tables in internal/ephemeris.
//
// Run it from internal/ephemeris (see the go:generate line in vsop87.go):
//
//	go run ../cmd/genvsop -body earth -var earthVSOP87D -out vsop87_earth.go
//
// It downloads the VSOP87D file for the body from the IMCCE server (or reads
// -in), checks its SHA-256 against the pinned value, keeps every term whose
// amplitude, scaled by the largest power of time it is multiplied by across
// the range the library supports, reaches the cutoff, and writes the Go
// table. The raw file is never committed; the header of the generated file
// records where it came from, its digest, and the cutoff, so the table can be
// regenerated and checked.
//
// The theory is Bretagnon and Francou, "Planetary theories in rectangular and
// spherical variables. VSOP87 solutions", Astronomy and Astrophysics 202, 309
// (1988). VSOP87D gives heliocentric ecliptic longitude, latitude, and radius
// referred to the dynamical ecliptic and equinox of date, in radians and
// astronomical units, as series in Julian millennia of TDB from J2000.0.
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
	"strconv"
	"strings"
	"time"
)

// source describes one VSOP87D body file on the IMCCE server.
type source struct {
	file   string
	sha256 string
}

// sources pins each file this package generates from. The digests were taken
// from the files on the IMCCE server in September 2026.
var sources = map[string]source{
	"mercury": {"VSOP87D.mer", "f468481b5a05080a943ad4746ff7ea7e0ff6652b71a46d83c9c636cb69485e34"},
	"venus":   {"VSOP87D.ven", "cb2f3a738289ed45f69fec1845e480baf4b32d481eccc21b8629a2d0d10e8261"},
	"earth":   {"VSOP87D.ear", "8b160c859136d467f2be7fc29efa8a9652e95516dfbde00e4c739d7ddc90ca91"},
	"mars":    {"VSOP87D.mar", "b1184df9553d85ffcf904c16bd437ab668804fa98859f27fe2e7bf6cfa6bc07e"},
	"jupiter": {"VSOP87D.jup", "3f3dfbc7d117ecad2b2dadf2fc626b260a3cd5efa98e7d4c6b26cd682fc48090"},
	"saturn":  {"VSOP87D.sat", "2e49e19396f24c17298f0b667e7763ee5c28b60d549c89d72a17dfd5f8d46b05"},
	"uranus":  {"VSOP87D.ura", "80eb3a778d53f450066d9b13f17e15b979e253437d22f33ec5a4604cdb2872a7"},
	"neptune": {"VSOP87D.nep", "3ff65a5cabc04c411f975888f77268b89e336ece0b75d198a3935090fdde3ae6"},
}

const baseURL = "https://ftp.imcce.fr/pub/ephem/planets/vsop87/"

// term is one periodic term A cos(B + C tau).
type term struct{ a, b, c float64 }

func main() {
	body := flag.String("body", "earth", "body to generate (a key of the pinned sources)")
	in := flag.String("in", "", "read the VSOP87D file from this path instead of downloading it")
	out := flag.String("out", "", "output Go file")
	varName := flag.String("var", "", "name of the generated table variable")
	cutoff := flag.Float64("cutoff", 3e-8, "smallest amplitude kept, radians or astronomical units")
	tauMax := flag.Float64("taumax", 0.3, "largest |tau|, in Julian millennia from J2000, the truncation must hold over")
	pkg := flag.String("pkg", "ephemeris", "package of the generated file; outside ephemeris the types are imported from it")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("genvsop: ")

	src, ok := sources[*body]
	if !ok {
		log.Fatalf("unknown body %q", *body)
	}
	if *out == "" || *varName == "" {
		log.Fatal("-out and -var are required")
	}

	raw, origin, err := load(src, *in)
	if err != nil {
		log.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	if digest != src.sha256 {
		log.Fatalf("%s: SHA-256 %s, pinned %s", origin, digest, src.sha256)
	}
	log.Printf("read %s (%d bytes, sha256 %s)", origin, len(raw), digest)

	series, err := parse(raw)
	if err != nil {
		log.Fatalf("%s: %v", origin, err)
	}

	kept, total := 0, 0
	var trunc [3][][]term
	for v := range series {
		for alpha, ts := range series[v] {
			scale := math.Pow(*tauMax, float64(alpha))
			var keep []term
			for _, t := range ts {
				total++
				if math.Abs(t.a)*scale >= *cutoff {
					keep = append(keep, t)
				}
			}
			kept += len(keep)
			log.Printf("variable %d, tau^%d: kept %d of %d terms", v+1, alpha, len(keep), len(ts))
			trunc[v] = append(trunc[v], keep)
		}
	}
	log.Printf("kept %d of %d terms", kept, total)

	code, err := render(*pkg, *varName, *body, src, digest, *cutoff, *tauMax, kept, total, trunc)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, code, 0o600); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", *out)
}

// load reads the file from disk when a path is given and downloads it
// otherwise.
func load(src source, path string) ([]byte, string, error) {
	if path != "" {
		b, err := os.ReadFile(path) // #nosec G304 -- a path the developer passes on the command line.
		return b, path, err
	}
	u := baseURL + src.file
	client := &http.Client{Timeout: 2 * time.Minute}
	start := time.Now()
	resp, err := client.Get(u) // #nosec G107 -- the URL is built from pinned constants.
	if err != nil {
		return nil, u, fmt.Errorf("get %s: %w", u, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, u, fmt.Errorf("get %s: %s", u, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	log.Printf("downloaded %s in %v", u, time.Since(start).Round(time.Millisecond))
	return b, u, err
}

// parse reads a VSOP87D file into series[variable][power of tau][term]. Each
// block starts with a header line naming the variable and the power of tau;
// each term line ends with the amplitude A, the phase B, and the frequency C.
func parse(raw []byte) ([3][][]term, error) {
	var series [3][][]term
	// Terms are collected per (variable, power) first and laid out once the
	// whole file is read, so a block may appear in any order.
	type key struct{ v, alpha int }
	blocks := map[key][]term{}
	var maxAlpha [3]int
	for i := range maxAlpha {
		maxAlpha[i] = -1
	}
	cur, inBlock := key{}, false
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		if strings.TrimSpace(text) == "" {
			continue
		}
		f := strings.Fields(text)
		if strings.Contains(text, "VSOP87") {
			v, alpha, err := header(f)
			if err != nil {
				return series, fmt.Errorf("line %d: %w", line, err)
			}
			cur, inBlock = key{v, alpha}, true
			if _, seen := blocks[cur]; !seen {
				blocks[cur] = []term{}
			}
			maxAlpha[v] = max(maxAlpha[v], alpha)
			continue
		}
		if !inBlock {
			return series, fmt.Errorf("line %d: term before any header", line)
		}
		if len(f) < 3 {
			return series, fmt.Errorf("line %d: short term line", line)
		}
		var t term
		var err error
		if t.a, err = strconv.ParseFloat(f[len(f)-3], 64); err != nil {
			return series, fmt.Errorf("line %d: %w", line, err)
		}
		if t.b, err = strconv.ParseFloat(f[len(f)-2], 64); err != nil {
			return series, fmt.Errorf("line %d: %w", line, err)
		}
		if t.c, err = strconv.ParseFloat(f[len(f)-1], 64); err != nil {
			return series, fmt.Errorf("line %d: %w", line, err)
		}
		blocks[cur] = append(blocks[cur], t)
	}
	if err := sc.Err(); err != nil {
		return series, err
	}
	for v := range series {
		out := make([][]term, maxAlpha[v]+1)
		for alpha := range out {
			out[alpha] = blocks[key{v, alpha}]
		}
		series[v] = out
	}
	return series, nil
}

// header returns the variable index (0 for L, 1 for B, 2 for R) and the power
// of tau from a block header such as
// "VSOP87 VERSION D4 EARTH VARIABLE 1 (LBR) *T**0 559 TERMS ...".
func header(f []string) (int, int, error) {
	v, alpha := -1, -1
	for i, w := range f {
		if w == "VARIABLE" && i+1 < len(f) {
			n, err := strconv.Atoi(f[i+1])
			if err != nil || n < 1 || n > 3 {
				return 0, 0, fmt.Errorf("bad variable %q", f[i+1])
			}
			v = n - 1
		}
		if strings.HasPrefix(w, "*T**") {
			n, err := strconv.Atoi(strings.TrimPrefix(w, "*T**"))
			if err != nil || n < 0 {
				return 0, 0, fmt.Errorf("bad power %q", w)
			}
			alpha = n
		}
	}
	if v < 0 || alpha < 0 {
		return 0, 0, fmt.Errorf("unrecognised header %q", strings.Join(f, " "))
	}
	return v, alpha, nil
}

// render writes the generated Go file.
func render(pkg, varName, body string, src source, digest string, cutoff, tauMax float64, kept, total int, s [3][][]term) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by genvsop. DO NOT EDIT.\n\npackage %s\n\n", pkg)
	b.WriteString(license)
	qual := ""
	if pkg != "ephemeris" {
		b.WriteString("\nimport \"github.com/Bugs5382/go-astronomy/internal/ephemeris\"\n")
		qual = "ephemeris."
	}
	fmt.Fprintf(&b, "\n// %s is the VSOP87D series for %s, truncated by amplitude.\n", varName, strings.ToUpper(body[:1])+body[1:])
	fmt.Fprintf(&b, "//\n// Source: %s%s\n", baseURL, src.file)
	fmt.Fprintf(&b, "// SHA-256: %s\n", digest)
	fmt.Fprintf(&b, "// Truncation: |A| * %g^alpha >= %g; %d of %d terms kept.\n", tauMax, cutoff, kept, total)
	fmt.Fprintf(&b, "var %s = %sVSOP87Body{\n", varName, qual)
	for v, name := range []string{"L", "B", "R"} {
		fmt.Fprintf(&b, "%s: [][]%sVSOP87Term{\n", name, qual)
		for alpha, ts := range s[v] {
			fmt.Fprintf(&b, "// %s%d\n{\n", name, alpha)
			for _, t := range ts {
				fmt.Fprintf(&b, "{A: %s, B: %s, C: %s},\n", num(t.a), num(t.b), num(t.c))
			}
			b.WriteString("},\n")
		}
		b.WriteString("},\n")
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
