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

// CCSDS Orbit Mean-Elements Messages (OMM), the successor to the two-line
// format now that five-digit catalogue numbers are running out. CelesTrak and
// Space-Track publish OMM as JSON and as XML; both are read here. Only
// messages whose mean element theory is SGP4 (or unstated, as in CelesTrak's
// JSON) are accepted, since the elements mean nothing to any other propagator.

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// ErrMalformedOMM is the cause when an OMM message cannot be read or lacks a
// required field; the coded error carries astronomy.CodeInvalidElements.
var ErrMalformedOMM = errors.New("satellite: malformed OMM")

// maxOMM bounds how much of an OMM message is read: CelesTrak's full active
// catalogue is a few megabytes of JSON.
const maxOMM = 64 << 20

// ommFields is one OMM record, with every value kept as text so JSON numbers
// and strings and XML elements read the same way.
type ommFields map[string]string

// ParseOMM reads one or more element sets from a CCSDS OMM message in JSON (an
// object or an array of objects, as CelesTrak serves) or XML (an ndm or omm
// document). Every record must carry the SGP4 mean elements: EPOCH,
// MEAN_MOTION, ECCENTRICITY, INCLINATION, RA_OF_ASC_NODE, ARG_OF_PERICENTER,
// MEAN_ANOMALY, and BSTAR, plus NORAD_CAT_ID. Errors are go-apperr coded with
// astronomy.CodeInvalidElements and wrap ErrMalformedOMM.
func ParseOMM(r io.Reader) ([]Elements, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxOMM))
	if err != nil {
		return nil, ommErr("read: %v", err)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, ommErr("empty message")
	}
	var records []ommFields
	switch trimmed[0] {
	case '[', '{':
		records, err = ommJSON(trimmed)
	case '<':
		records, err = ommXML(trimmed)
	default:
		return nil, ommErr("not JSON or XML")
	}
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, ommErr("no element sets")
	}
	out := make([]Elements, 0, len(records))
	for i, f := range records {
		e, err := fromOMM(f)
		if err != nil {
			return nil, ommErr("record %d: %v", i+1, err)
		}
		out = append(out, e)
	}
	return out, nil
}

func ommErr(format string, args ...any) error {
	return apperr.Coded(astronomy.CodeInvalidElements, fmt.Errorf("%w: "+format, append([]any{ErrMalformedOMM}, args...)...))
}

func ommJSON(raw []byte) ([]ommFields, error) {
	var list []map[string]json.RawMessage
	if raw[0] == '{' {
		var one map[string]json.RawMessage
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil, ommErr("json: %v", err)
		}
		list = append(list, one)
	} else if err := json.Unmarshal(raw, &list); err != nil {
		return nil, ommErr("json: %v", err)
	}
	out := make([]ommFields, 0, len(list))
	for _, m := range list {
		f := ommFields{}
		for k, v := range m {
			var s string
			if err := json.Unmarshal(v, &s); err == nil {
				f[strings.ToUpper(k)] = s
				continue
			}
			f[strings.ToUpper(k)] = string(bytes.TrimSpace(v))
		}
		out = append(out, f)
	}
	return out, nil
}

// ommXML collects the leaf elements of every segment: the metadata, the mean
// elements, and the TLE parameters sit side by side in one record.
func ommXML(raw []byte) ([]ommFields, error) {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var out []ommFields
	var cur ommFields
	var text strings.Builder
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ommErr("xml: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "segment" {
				cur = ommFields{}
				continue
			}
			text.Reset()
		case xml.CharData:
			if cur != nil {
				text.Write(t)
			}
		case xml.EndElement:
			if cur == nil {
				continue
			}
			if t.Name.Local == "segment" {
				out = append(out, cur)
				cur = nil
				continue
			}
			if v := strings.TrimSpace(text.String()); v != "" {
				cur[strings.ToUpper(t.Name.Local)] = v
			}
			text.Reset()
		}
	}
	return out, nil
}

// ommEpochLayouts are the EPOCH forms seen in practice: ISO 8601 with or
// without fractional seconds and a trailing Z.
var ommEpochLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
}

func fromOMM(f ommFields) (Elements, error) {
	if theory, ok := f["MEAN_ELEMENT_THEORY"]; ok && !strings.EqualFold(theory, "SGP4") {
		return Elements{}, fmt.Errorf("mean element theory %q is not SGP4", theory)
	}
	if frame, ok := f["REF_FRAME"]; ok && !strings.EqualFold(frame, "TEME") {
		return Elements{}, fmt.Errorf("reference frame %q is not TEME", frame)
	}
	if ts, ok := f["TIME_SYSTEM"]; ok && !strings.EqualFold(ts, "UTC") {
		return Elements{}, fmt.Errorf("time system %q is not UTC", ts)
	}
	var err error
	num := func(key string, required bool) float64 {
		v, ok := f[key]
		if !ok || v == "" {
			if required && err == nil {
				err = fmt.Errorf("missing %s", key)
			}
			return 0
		}
		x, perr := strconv.ParseFloat(v, 64)
		if (perr != nil || math.IsNaN(x) || math.IsInf(x, 0)) && err == nil {
			err = fmt.Errorf("%s %q is not a number", key, v)
		}
		return x
	}
	var e Elements
	e.MeanMotion = num("MEAN_MOTION", true)
	e.Eccentricity = num("ECCENTRICITY", true)
	e.Inclination = num("INCLINATION", true)
	e.RAAN = num("RA_OF_ASC_NODE", true)
	e.ArgPerigee = num("ARG_OF_PERICENTER", true)
	e.MeanAnomaly = num("MEAN_ANOMALY", true)
	e.BStar = num("BSTAR", true)
	e.NDot = num("MEAN_MOTION_DOT", false)
	e.NDDot = num("MEAN_MOTION_DDOT", false)
	e.SatNum = int(num("NORAD_CAT_ID", true))
	e.EphemerisType = int(num("EPHEMERIS_TYPE", false))
	e.ElementSetNumber = int(num("ELEMENT_SET_NO", false))
	e.RevNumber = int(num("REV_AT_EPOCH", false))
	if err != nil {
		return Elements{}, err
	}
	e.CatalogNumber = strconv.Itoa(e.SatNum)
	e.Classification = 'U'
	if c := f["CLASSIFICATION_TYPE"]; len(c) == 1 {
		e.Classification = c[0]
	}
	if id := f["OBJECT_ID"]; len(id) > 2 {
		// "1998-067A" is "98067A" in the two-line form.
		e.IntlDesignator = strings.ReplaceAll(id[2:], "-", "")
	}

	epochText, ok := f["EPOCH"]
	if !ok {
		return Elements{}, errors.New("missing EPOCH")
	}
	var epoch time.Time
	for _, layout := range ommEpochLayouts {
		if t, perr := time.Parse(layout, epochText); perr == nil {
			epoch = t.UTC()
			break
		}
	}
	if epoch.IsZero() {
		return Elements{}, fmt.Errorf("EPOCH %q is not an ISO 8601 time", epochText)
	}
	e.epoch = epoch
	e.EpochYear = epoch.Year()
	start := time.Date(epoch.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	e.EpochDay = 1 + epoch.Sub(start).Hours()/24

	rec := &satrec{
		bstar:   e.BStar,
		ecco:    e.Eccentricity,
		argpo:   e.ArgPerigee * deg2rd,
		inclo:   e.Inclination * deg2rd,
		mo:      e.MeanAnomaly * deg2rd,
		noKozai: e.MeanMotion / xpdotp,
		nodeo:   e.RAAN * deg2rd,
	}
	// Days since 1949 December 31 0h UTC, the SGP4 epoch origin, from whole
	// seconds and nanoseconds so no precision is lost.
	origin := time.Date(1949, 12, 31, 0, 0, 0, 0, time.UTC)
	sec := epoch.Unix() - origin.Unix()
	days := float64(sec)/86400 + float64(epoch.Nanosecond())/86400e9
	sgp4init(rec, days)
	e.rec = rec
	return e, nil
}
