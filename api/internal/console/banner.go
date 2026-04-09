// Package console renders styled attention-grabbing output to the
// terminal, such as the first-time setup banner printed at startup.
//
// Styling is emitted as raw ANSI SGR escape sequences written through
// go-colorable so that legacy Windows consoles (cmd.exe / PowerShell 5.1)
// receive Win32 console API calls instead of literal escape bytes.
// When stdout is piped to a file the colours are dropped automatically.
package console

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

// ANSI Select Graphic Rendition sequences used by the banner. Brand
// colours use 24-bit truecolor so they exactly match the Svelte theme;
// modern terminals (Windows Terminal, iTerm2, WezTerm, kitty, alacritty,
// gnome-terminal, PowerShell 7) all support this, and go-colorable maps
// the sequence to the nearest 16-colour equivalent on legacy consoles.
const (
	ansiReset     = "\x1b[0m"
	ansiBold      = "\x1b[1m"
	ansiDim       = "\x1b[2m"
	ansiUnderline = "\x1b[4m"
	ansiFgBright  = "\x1b[97m"

	// brandYellow matches rgb(255, 208, 0) — the "five" fill in the
	// website logo and the --primary token in the Svelte theme.
	brandYellow = "\x1b[38;2;255;208;0m"
)

// styledWriter returns a writer that safely emits ANSI escape sequences on
// every supported OS together with a flag indicating whether styling
// should be applied at all. Styling is disabled when stdout is not a TTY
// (piped / redirected) so log files never receive escape junk.
func styledWriter() (io.Writer, bool) {
	w := colorable.NewColorableStdout()
	useColor := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	return w, useColor
}

// SetupBanner prints a branded first-time-setup announcement framed by a
// left-edge accent bar in the runfive brand yellow. The wordmark is
// colour-split to mirror the website's logo branding: "run" stays neutral
// while "five" (and the title / URL) pick up the signature yellow.
func SetupBanner(url string) {
	w, useColor := styledWriter()

	// paint wraps s in the given SGR escape sequence when styling is on,
	// returning s verbatim when output is piped to a non-TTY sink.
	paint := func(s, sgr string) string {
		if !useColor || sgr == "" {
			return s
		}
		return sgr + s + ansiReset
	}

	// bar is the left accent column rendered on every banner row. Using
	// a bold half-block glyph gives the bar enough weight to read as an
	// intentional branding element rather than a stray character.
	bar := paint("▌", ansiBold+brandYellow)

	// row prefixes content with a three-space indent and the accent bar.
	row := func(content string) string {
		return "   " + bar + "   " + content
	}
	// blank is an empty row that still carries the accent bar, providing
	// vertical breathing room inside the branded column.
	blank := func() string {
		return "   " + bar
	}

	wordmark := paint("run", ansiBold+ansiFgBright) +
		paint("five", ansiBold+brandYellow) +
		paint("  ·  open source  ·  done right", ansiDim)

	title := paint("FIRST TIME SETUP", ansiBold+brandYellow)
	intro := "Open this URL in your browser to create your owner account:"
	arrow := paint("→", ansiBold+brandYellow)
	link := paint(url, ansiBold+ansiUnderline+brandYellow)
	footnote := paint("This code is only valid until setup is complete.", ansiDim)

	fmt.Fprintln(w)
	fmt.Fprintln(w, blank())
	fmt.Fprintln(w, row(wordmark))
	fmt.Fprintln(w, blank())
	fmt.Fprintln(w, row(title))
	fmt.Fprintln(w, blank())
	fmt.Fprintln(w, row(intro))
	fmt.Fprintln(w, blank())
	fmt.Fprintln(w, row(arrow+"   "+link))
	fmt.Fprintln(w, blank())
	fmt.Fprintln(w, row(footnote))
	fmt.Fprintln(w, blank())
	fmt.Fprintln(w)
}
