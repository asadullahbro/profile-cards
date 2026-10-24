package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func readAll(resp *http.Response) ([]byte, error) {
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

func esc(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

// Colours follow the viewer's light/dark setting, which an SVG shown in an
// <img> can read itself. The amber matches the portfolio.
const style = `<style>
:root{--bg:#0d1117;--fg:#e6edf3;--muted:#8b949e;--line:#30363d;--accent:#f2a33d}
@media (prefers-color-scheme: light){:root{--bg:#ffffff;--fg:#1f2328;--muted:#656d76;--line:#d0d7de;--accent:#b86e00}}
text{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,'Liberation Mono',monospace}
</style>`

func langColor(name string) string {
	switch name {
	case "Go":
		return "#00ADD8"
	case "JavaScript":
		return "#F7DF1E"
	case "TypeScript":
		return "#3178C6"
	case "Python":
		return "#3776AB"
	case "Kotlin":
		return "#7F52FF"
	case "Shell":
		return "#89E051"
	case "HTML":
		return "#E34C26"
	}
	return "#8b949e"
}

type row struct {
	label string
	pct   float64
	color string
}

func card(w, h int, title, body string) []byte {
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">%s
<rect x="0.5" y="0.5" width="%d" height="%d" rx="8" fill="var(--bg)" stroke="var(--line)"/>
<text x="20" y="32" font-size="15" font-weight="700" fill="var(--accent)">%s</text>%s</svg>`,
		w, h, w, h, esc(title), style, w-1, h-1, esc(title), body))
}

func statsSVG(user string, items [][2]string) []byte {
	var b strings.Builder
	for i, it := range items {
		y := 62 + i*26
		fmt.Fprintf(&b, `<text x="20" y="%d" font-size="13" fill="var(--muted)">%s</text><text x="275" y="%d" font-size="13" font-weight="700" fill="var(--fg)" text-anchor="end">%s</text>`,
			y, esc(it[0]), y, esc(it[1]))
	}
	return card(295, 62+len(items)*26, user+" on GitHub", b.String())
}

func langsSVG(title string, rows []row) []byte {
	var b strings.Builder
	for i, r := range rows {
		y := 56 + i*30
		fmt.Fprintf(&b, `<text x="20" y="%d" font-size="12" fill="var(--fg)">%s</text><text x="275" y="%d" font-size="12" fill="var(--muted)" text-anchor="end">%.0f%%</text>`+
			`<rect x="20" y="%d" width="255" height="6" rx="3" fill="var(--line)"/><rect x="20" y="%d" width="%.1f" height="6" rx="3" fill="%s"/>`,
			y, esc(r.label), y, r.pct*100, y+6, y+6, 255*r.pct, r.color)
	}
	return card(295, 56+len(rows)*30, title, b.String())
}

// typingSVG types each line out, holds it, erases it, then moves to the next,
// all in SMIL, which GitHub shows in README images.
func typingSVG(lines []string, width int) []byte {
	const (
		charW  = 12.0 // 20px monospace
		typeS  = 0.07 // seconds per character
		holdS  = 1.6
		eraseS = 0.4
	)
	var slots []float64
	cycle := 0.0
	for _, l := range lines {
		d := float64(len([]rune(l)))*typeS + holdS + eraseS
		slots = append(slots, d)
		cycle += d
	}
	var b strings.Builder
	start := 0.0
	for i, l := range lines {
		n := float64(len([]rune(l)))
		full := n * charW
		x0 := (float64(width) - full) / 2
		if x0 < 4 {
			x0 = 4
		}
		tType := n * typeS
		t0 := start / cycle
		t1 := (start + tType) / cycle
		t2 := (start + tType + holdS) / cycle
		t3 := (start + slots[i]) / cycle
		kt := fmt.Sprintf("0;%.4f;%.4f;%.4f;%.4f;1", t0, t1, t2, t3)
		wv := fmt.Sprintf("0;0;%.1f;%.1f;0;0", full, full)
		xv := fmt.Sprintf("%.1f;%.1f;%.1f;%.1f;%.1f;%.1f", x0, x0, x0+full, x0+full, x0, x0)
		fmt.Fprintf(&b, `<clipPath id="c%d"><rect x="%.1f" y="0" height="50" width="0"><animate attributeName="width" values="%s" keyTimes="%s" dur="%.2fs" repeatCount="indefinite"/></rect></clipPath>`,
			i, x0, wv, kt, cycle)
		fmt.Fprintf(&b, `<text x="%.1f" y="32" font-size="20" fill="var(--accent)" clip-path="url(#c%d)">%s</text>`, x0, i, esc(l))
		fmt.Fprintf(&b, `<rect y="12" width="2" height="24" fill="var(--accent)" opacity="0"><animate attributeName="x" values="%s" keyTimes="%s" dur="%.2fs" repeatCount="indefinite"/><animate attributeName="opacity" values="0;0;1;1;0;0" keyTimes="%s" dur="%.2fs" repeatCount="indefinite"/></rect>`,
			xv, kt, cycle, kt, cycle)
		start += slots[i]
	}
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="50" viewBox="0 0 %d 50" role="img" aria-label="%s">%s%s</svg>`,
		width, width, esc(strings.Join(lines, ", ")), style, b.String()))
}
