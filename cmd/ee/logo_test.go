package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDetectColorMode(t *testing.T) {
	tests := []struct {
		name   string
		isTerm bool
		env    map[string]string
		goos   string
		want   colorMode
	}{
		{"not a terminal", false, map[string]string{"COLORTERM": "truecolor"}, "linux", colorNone},
		{"NO_COLOR set", true, map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor"}, "linux", colorNone},
		{"dumb terminal", true, map[string]string{"TERM": "dumb"}, "linux", colorNone},
		{"truecolor", true, map[string]string{"COLORTERM": "truecolor"}, "linux", colorTrue},
		{"24bit", true, map[string]string{"COLORTERM": "24bit"}, "darwin", colorTrue},
		{"plain unix terminal", true, map[string]string{"TERM": "xterm-256color"}, "darwin", color256},
		{"windows terminal", true, map[string]string{"WT_SESSION": "abc"}, "windows", colorTrue},
		{"legacy windows console", true, map[string]string{}, "windows", colorNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(k string) string { return tc.env[k] }
			if got := detectColorMode(tc.isTerm, getenv, tc.goos); got != tc.want {
				t.Errorf("detectColorMode() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLogoLines(t *testing.T) {
	if got := logoLines(colorNone); got != nil {
		t.Errorf("logoLines(colorNone) = %d lines, want nil", len(got))
	}
	tests := []struct {
		name   string
		mode   colorMode
		marker string
	}{
		{"truecolor", colorTrue, "38;2;"},
		{"256 color", color256, "38;5;"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines := logoLines(tc.mode)
			if len(lines) != len(logoPixels)/2 {
				t.Fatalf("got %d lines, want %d", len(lines), len(logoPixels)/2)
			}
			for i, l := range lines {
				if !strings.Contains(l, tc.marker) {
					t.Errorf("line %d has no %q color sequence", i, tc.marker)
				}
				if !strings.HasSuffix(l, "\x1b[0m") {
					t.Errorf("line %d does not reset attributes", i)
				}
			}
		})
	}
}

func TestLogo256(t *testing.T) {
	tests := []struct {
		name    string
		r, g, b uint8
		want    int
	}{
		{"white key", 255, 255, 255, 231},
		{"key edge", 190, 190, 235, 189},
		{"gradient light end", 99, 102, 241, 62},
		{"gradient dark end", 49, 46, 129, 62},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := logo256(tc.r, tc.g, tc.b); got != tc.want {
				t.Errorf("logo256(%d,%d,%d) = %d, want %d", tc.r, tc.g, tc.b, got, tc.want)
			}
		})
	}
}

func TestWriteLogoHeader(t *testing.T) {
	var buf bytes.Buffer
	writeLogoHeader(&buf, colorNone, "Envoke (ee)", "tagline")
	if buf.Len() != 0 {
		t.Errorf("colorNone wrote %q, want nothing", buf.String())
	}

	buf.Reset()
	writeLogoHeader(&buf, colorTrue, "Envoke (ee)", "tagline")
	out := buf.String()
	for _, want := range []string{"Envoke (ee)", "tagline"} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q", want)
		}
	}
}
