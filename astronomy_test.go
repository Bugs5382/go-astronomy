package astronomy

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
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	log "github.com/Bugs5382/go-log"
)

func TestObserverLocationDefaultsToUTC(t *testing.T) {
	t.Parallel()
	o := Observer{Lat: 40.678, Lng: -73.944}
	if got := o.Location(); got != time.UTC {
		t.Errorf("Location() = %v, want UTC when TZ is nil", got)
	}
}

func TestObserverLocationUsesTZ(t *testing.T) {
	t.Parallel()
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tz database unavailable: %v", err)
	}
	o := Observer{Lat: 40.678, Lng: -73.944, TZ: tz}
	if got := o.Location(); got != tz {
		t.Errorf("Location() = %v, want %v", got, tz)
	}
}

func TestHorizontalAboveHorizon(t *testing.T) {
	t.Parallel()
	cases := []struct {
		alt  float64
		want bool
	}{
		{10, true},
		{0, false},
		{-0.5, false},
		{89.9, true},
	}
	for _, c := range cases {
		h := Horizontal{Altitude: c.alt, Azimuth: 123}
		if got := h.AboveHorizon(); got != c.want {
			t.Errorf("AboveHorizon(alt=%v) = %v, want %v", c.alt, got, c.want)
		}
	}
}

func TestAngularDiameterRadius(t *testing.T) {
	t.Parallel()
	d := AngularDiameter(0.5)
	if got := d.Radius(); got != 0.25 {
		t.Errorf("Radius() = %v, want 0.25", got)
	}
}

func TestErrorsRegistryDescribesEveryCode(t *testing.T) {
	t.Parallel()
	reg := Errors()
	if reg == nil {
		t.Fatal("Errors() = nil, want a registry")
	}
	for _, code := range []int{CodeInvalidLatitude, CodeInvalidLongitude, CodeInvalidSegmentation} {
		e, ok := reg.Describe(code)
		if !ok {
			t.Errorf("Describe(%d) not found", code)
			continue
		}
		if e.Code != code {
			t.Errorf("Describe(%d).Code = %d", code, e.Code)
		}
		if code/1000 != ErrorServiceDigit {
			t.Errorf("code %d does not carry service digit %d", code, ErrorServiceDigit)
		}
	}
}

func TestErrorsRegistryPresentsCode(t *testing.T) {
	t.Parallel()
	// A coded error resolves to its own code and a sanitized, quotable message.
	wrapped := apperr.Coded(CodeInvalidLatitude, errors.New("latitude 120 out of range"))
	msg, code := Errors().Present(wrapped, CodeInvalidSegmentation)
	if code != CodeInvalidLatitude {
		t.Errorf("Present code = %d, want %d", code, CodeInvalidLatitude)
	}
	if want := fmt.Sprintf("%d", CodeInvalidLatitude); !strings.Contains(msg, want) {
		t.Errorf("Present message %q does not mention code %s", msg, want)
	}
	// An uncoded error falls back to the default code the caller supplies.
	if _, code := Errors().Present(errors.New("plain"), CodeInvalidSegmentation); code != CodeInvalidSegmentation {
		t.Errorf("uncoded Present code = %d, want default %d", code, CodeInvalidSegmentation)
	}
}

// TestLogSinkAdapter exercises the go-apperr Logger adapter over go-log without
// any OpenTelemetry setup: with no span on the context it logs the plain coded
// line, confirming go-log is wired as the sink and stays dormant on tracing.
// LOG_LEVEL is disabled so the test emits nothing to stdout.
func TestLogSinkAdapter(t *testing.T) {
	t.Setenv("LOG_LEVEL", "disabled")
	sink := logSink{l: log.NewLogger("go-astronomy-test")}
	sink.LogCoded(context.Background(), CodeInvalidLatitude, errors.New("boom"))
}

func TestPositionEmbedsHorizontal(t *testing.T) {
	t.Parallel()
	p := Position{
		Horizontal: Horizontal{Altitude: 30, Azimuth: 200},
		Diameter:   AngularDiameter(0.53),
	}
	if !p.AboveHorizon() {
		t.Error("expected embedded AboveHorizon to report true")
	}
	if p.Azimuth != 200 {
		t.Errorf("embedded Azimuth = %v, want 200", p.Azimuth)
	}
}
