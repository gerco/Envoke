package main

//go:generate go run logo_gen.go

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// colorMode is the color depth used to draw the help logo.
type colorMode int

const (
	colorNone colorMode = iota
	color256
	colorTrue
)

// detectColorMode decides whether, and how, the logo can be drawn. It returns
// colorNone when output is not a terminal or color has been disabled.
func detectColorMode(isTerminal bool, getenv func(string) string, goos string) colorMode {
	if !isTerminal || getenv("NO_COLOR") != "" || getenv("TERM") == "dumb" {
		return colorNone
	}
	switch strings.ToLower(getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return colorTrue
	}
	if goos == "windows" {
		// Windows Terminal supports true color; the legacy console may not
		// process escape sequences at all.
		if getenv("WT_SESSION") != "" {
			return colorTrue
		}
		return colorNone
	}
	return color256
}

// logoLines renders logoPixels with half-block characters, two pixel rows per
// line. Transparent pixels show the terminal's own background.
func logoLines(mode colorMode) []string {
	if mode == colorNone {
		return nil
	}
	const opaque = 128
	lines := make([]string, 0, len(logoPixels)/2)
	for y := 0; y+1 < len(logoPixels); y += 2 {
		var sb strings.Builder
		for x := range logoPixels[y] {
			top, bot := logoPixels[y][x], logoPixels[y+1][x]
			switch {
			case top[3] >= opaque && bot[3] >= opaque:
				sb.WriteString(fgColor(mode, top) + bgColor(mode, bot) + "▀")
			case top[3] >= opaque:
				sb.WriteString("\x1b[49m" + fgColor(mode, top) + "▀")
			case bot[3] >= opaque:
				sb.WriteString("\x1b[49m" + fgColor(mode, bot) + "▄")
			default:
				sb.WriteString("\x1b[0m ")
			}
		}
		sb.WriteString("\x1b[0m")
		lines = append(lines, sb.String())
	}
	return lines
}

func fgColor(mode colorMode, p [4]uint8) string { return sgrColor(mode, 38, p) }
func bgColor(mode colorMode, p [4]uint8) string { return sgrColor(mode, 48, p) }

func sgrColor(mode colorMode, base int, p [4]uint8) string {
	if mode == colorTrue {
		return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", base, p[0], p[1], p[2])
	}
	return fmt.Sprintf("\x1b[%d;5;%dm", base, logo256(p[0], p[1], p[2]))
}

// logo256 maps a logo pixel to the 256-color palette. The palette's coarse
// color cube turns the icon's gradient into uneven bands, so the background
// is drawn as one flat indigo and only the white key keeps its shading.
func logo256(r, g, b uint8) int {
	lum := (299*int(r) + 587*int(g) + 114*int(b)) / 1000
	switch {
	case lum >= 215:
		return 231 // white
	case lum >= 160:
		return 189 // light lavender, anti-aliased key edges
	default:
		return 62 // indigo
	}
}

// writeLogoHeader writes the logo with the title and tagline beside it.
func writeLogoHeader(w io.Writer, mode colorMode, title, tagline string) {
	lines := logoLines(mode)
	if lines == nil {
		return
	}
	mid := len(lines)/2 - 1
	for i, l := range lines {
		text := ""
		switch i {
		case mid:
			text = "\x1b[1m" + title + "\x1b[0m"
		case mid + 1:
			text = tagline
		}
		fmt.Fprintf(w, "%s  %s\n", l, text)
	}
	fmt.Fprintln(w)
}

// installLogoHelp prefixes the root command's help with the logo when it is
// written to a color-capable terminal. Subcommand help is unchanged.
func installLogoHelp(root *cobra.Command, bin string) {
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == root {
			out := cmd.OutOrStdout()
			isTerm := false
			if f, ok := out.(*os.File); ok {
				isTerm = term.IsTerminal(int(f.Fd()))
			}
			mode := detectColorMode(isTerm, os.Getenv, runtime.GOOS)
			writeLogoHeader(out, mode, "Envoke ("+bin+")", root.Short)
		}
		defaultHelp(cmd, args)
	})
}
