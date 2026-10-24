package main

import (
	"fmt"
	"math"
	"strings"
)

// The three cards below reproduce the look of github-readme-stats (stats and
// top languages) and github-readme-streak-stats, which the README used before:
// same sizes, fonts, colours and animation, in the dark theme with the amber
// accent. github-readme-stats is MIT licensed.

const (
	cardBG = "#0d1117"
	accent = "#F7DF1E"
)

func statsCard(p *profile) []byte {
	level, pct := rank(p.Commits, p.PRs, p.Issues, p.Reviews, p.Stars, p.Followers)
	const circ = 251.32741228718345 // 2π · 40
	offset := (100 - (100 - pct)) / 100 * circ
	rows := []struct {
		label string
		value int
	}{
		{"Total Commits (last year):", p.Commits},
		{"Total PRs:", p.PRs},
		{"Total Issues:", p.Issues},
		{"Contributed to (last year):", p.ContributedTo},
	}
	height := 45 + (len(rows)+1)*25 // as github-readme-stats sizes it
	var b strings.Builder
	for i, r := range rows {
		fmt.Fprintf(&b, `<g transform="translate(0, %d)"><g class="stagger" style="animation-delay: %dms" transform="translate(25, 0)"><svg data-testid="icon" class="icon" viewBox="0 0 16 16" version="1.1" width="16" height="16"><path fill-rule="evenodd" d="%s"/></svg><text class="stat bold" x="25" y="12.5">%s</text><text class="stat bold" x="219.01" y="12.5">%d</text></g></g>`,
			i*25, 450+i*150, statIcons[i+1], esc(r.label), r.value)
	}
	return []byte(fmt.Sprintf(`<svg width="467" height="%d" viewBox="0 0 467 %d" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="%s">
<title>%s</title>
<style>
.header{font:600 18px 'Segoe UI',Ubuntu,Sans-Serif;fill:%s;animation:fadeInAnimation .8s ease-in-out forwards}
@supports(-moz-appearance:auto){.header{font-size:15.5px}}
.stat{font:600 14px 'Segoe UI',Ubuntu,"Helvetica Neue",Sans-Serif;fill:#9f9f9f}
@supports(-moz-appearance:auto){.stat{font-size:12px}}
.stagger{opacity:0;animation:fadeInAnimation .3s ease-in-out forwards}
.rank-text{font:800 24px 'Segoe UI',Ubuntu,Sans-Serif;fill:#9f9f9f;animation:scaleInAnimation .3s ease-in-out forwards}
.bold{font-weight:700}
.icon{fill:%s;display:block}
.rank-circle-rim{stroke:%s;fill:none;stroke-width:6;opacity:.2}
.rank-circle{stroke:%s;stroke-dasharray:250;fill:none;stroke-width:6;stroke-linecap:round;opacity:.8;transform-origin:-10px 8px;transform:rotate(-90deg);animation:rankAnimation 1s forwards ease-in-out}
@keyframes rankAnimation{from{stroke-dashoffset:%.4f}to{stroke-dashoffset:%.4f}}
@keyframes scaleInAnimation{from{transform:translate(-5px,5px) scale(0)}to{transform:translate(-5px,5px) scale(1)}}
@keyframes fadeInAnimation{from{opacity:0}to{opacity:1}}
</style>
<rect x="0.5" y="0.5" rx="4.5" height="99%%" width="466" fill="%s" stroke-opacity="0"/>
<g transform="translate(25, 35)"><text x="0" y="0" class="header">%s</text></g>
<g transform="translate(0, 55)">
<g transform="translate(390.5, %.1f)"><circle class="rank-circle-rim" cx="-10" cy="8" r="40"/><circle class="rank-circle" cx="-10" cy="8" r="40"/><g class="rank-text"><text x="-5" y="3" alignment-baseline="central" dominant-baseline="central" text-anchor="middle">%s</text></g></g>
<svg x="0" y="0">%s</svg>
</g></svg>`,
		height, height, esc(p.Name+"'s GitHub Stats, Rank: "+level), esc(p.Name+"'s GitHub Stats, Rank: "+level),
		accent, accent, accent, accent, circ, offset, cardBG, esc(p.Name+"'s GitHub Stats"), float64(height)/2-50, esc(level), b.String()))
}

func langsCard(langs []lang) []byte {
	if len(langs) > 6 {
		langs = langs[:6]
	}
	total := 0
	for _, l := range langs {
		total += l.Size
	}
	var bars, legend strings.Builder
	x := 0.0
	for _, l := range langs {
		w := float64(l.Size) / float64(total) * 250
		fmt.Fprintf(&bars, `<rect mask="url(#rect-mask)" x="%.2f" y="0" width="%.2f" height="8" fill="%s"/>`, x, w, langColor(l))
		x += w
	}
	rows := (len(langs) + 1) / 2
	for col := 0; col < 2; col++ {
		fmt.Fprintf(&legend, `<g transform="translate(%d, 0)">`, col*150)
		for r := 0; r < rows; r++ {
			i := col*rows + r
			if i >= len(langs) {
				break
			}
			l := langs[i]
			fmt.Fprintf(&legend, `<g transform="translate(0, %d)"><g class="stagger" style="animation-delay: %dms"><circle cx="5" cy="6" r="5" fill="%s"/><text x="15" y="10" class="lang-name">%s %.2f%%</text></g></g>`,
				r*25, 450+r*150, langColor(l), esc(l.Name), float64(l.Size)/float64(total)*100)
		}
		legend.WriteString(`</g>`)
	}
	h := 90 + rows*25
	return []byte(fmt.Sprintf(`<svg width="300" height="%d" viewBox="0 0 300 %d" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Most Used Languages">
<title>Most Used Languages</title>
<style>
.header{font:600 18px 'Segoe UI',Ubuntu,Sans-Serif;fill:%s;animation:fadeInAnimation .8s ease-in-out forwards}
@supports(-moz-appearance:auto){.header{font-size:15.5px}}
.lang-name{font:400 11px "Segoe UI",Ubuntu,Sans-Serif;fill:#FFFFFF}
.stagger{opacity:0;animation:fadeInAnimation .3s ease-in-out forwards}
@keyframes fadeInAnimation{from{opacity:0}to{opacity:1}}
</style>
<rect x="0.5" y="0.5" rx="4.5" height="99%%" width="299" fill="%s" stroke-opacity="0"/>
<g transform="translate(25, 35)"><text x="0" y="0" class="header">Most Used Languages</text></g>
<g transform="translate(0, 55)"><svg x="25"><mask id="rect-mask"><rect x="0" y="0" width="250" height="8" fill="white" rx="5"/></mask>%s<g transform="translate(0, 25)">%s</g></svg></g>
</svg>`, h, h, accent, cardBG, bars.String(), legend.String()))
}

func langColor(l lang) string {
	if l.Color != "" {
		return l.Color
	}
	return "#858585"
}

func streakCard(s streaks) []byte {
	since := s.Since.Format("Jan 2, 2006") + " - Present"
	text := func(x, y float64, weight, size int, fill, delay, body string) string {
		return fmt.Sprintf(`<g transform="translate(%.1f, %.0f)"><text x="0" y="32" text-anchor="middle" fill="%s" font-family='"Segoe UI", Ubuntu, sans-serif' font-weight="%d" font-size="%dpx" style="opacity:0;animation:fadein .5s linear forwards %s">%s</text></g>`,
			x, y, fill, weight, size, delay, esc(body))
	}
	var b strings.Builder
	b.WriteString(text(82.5, 48, 700, 28, "#FEFEFE", "0.6s", comma(s.Total)))
	b.WriteString(text(82.5, 84, 400, 14, "#FEFEFE", "0.7s", "Total Contributions"))
	b.WriteString(text(82.5, 114, 400, 12, "#9E9E9E", "0.8s", since))
	b.WriteString(text(247.5, 108, 700, 14, "#FB8C00", "0.9s", "Current Streak"))
	b.WriteString(fmt.Sprintf(`<g transform="translate(247.5, 145)"><text x="0" y="21" text-anchor="middle" fill="#9E9E9E" font-family='"Segoe UI", Ubuntu, sans-serif' font-size="12px" style="opacity:0;animation:fadein .5s linear forwards 0.9s">%s</text></g>`, esc(s.Current.String())))
	b.WriteString(`<g mask="url(#mask_out_ring_behind_fire)"><circle cx="247.5" cy="71" r="40" fill="none" stroke="` + accent + `" stroke-width="5" style="opacity:0;animation:fadein .5s linear forwards 0.4s"/></g>`)
	b.WriteString(`<g transform="translate(247.5, 19.5)" stroke-opacity="0" style="opacity:0;animation:fadein .5s linear forwards 0.6s"><path d="M 1.5 0.67 C 1.5 0.67 2.24 3.32 2.24 5.47 C 2.24 7.53 0.89 9.2 -1.17 9.2 C -3.23 9.2 -4.79 7.53 -4.79 5.47 L -4.76 5.11 C -6.78 7.51 -8 10.62 -8 13.99 C -8 18.41 -4.42 22 0 22 C 4.42 22 8 18.41 8 13.99 C 8 8.6 5.41 3.79 1.5 0.67 Z M -0.29 19 C -2.07 19 -3.51 17.6 -3.51 15.86 C -3.51 14.24 -2.46 13.1 -0.7 12.74 C 1.07 12.38 2.9 11.53 3.92 10.16 C 4.31 11.45 4.51 12.81 4.51 14.2 C 4.51 16.85 2.36 19 -0.29 19 Z" fill="` + accent + `" stroke-opacity="0"/></g>`)
	b.WriteString(fmt.Sprintf(`<g transform="translate(247.5, 48)"><text x="0" y="32" text-anchor="middle" fill="#FEFEFE" font-family='"Segoe UI", Ubuntu, sans-serif' font-weight="700" font-size="28px" style="animation:currstreak .6s linear forwards">%d</text></g>`, s.Current.Count))
	b.WriteString(text(412.5, 48, 700, 28, "#FEFEFE", "1.2s", comma(s.Longest.Count)))
	b.WriteString(text(412.5, 84, 400, 14, "#FEFEFE", "1.3s", "Longest Streak"))
	b.WriteString(text(412.5, 114, 400, 12, "#9E9E9E", "1.4s", s.Longest.String()))
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" style="isolation: isolate" viewBox="0 0 495 195" width="495px" height="195px" direction="ltr" role="img" aria-label="Contribution streaks">
<style>
@keyframes currstreak{0%%{font-size:3px;opacity:.2}80%%{font-size:34px;opacity:1}100%%{font-size:28px;opacity:1}}
@keyframes fadein{0%%{opacity:0}100%%{opacity:1}}
</style>
<defs><clipPath id="outer_rectangle"><rect width="495" height="195" rx="4.5"/></clipPath><mask id="mask_out_ring_behind_fire"><rect width="495" height="195" fill="white"/><ellipse cx="247.5" cy="32" rx="13" ry="18" fill="black"/></mask></defs>
<g clip-path="url(#outer_rectangle)"><rect fill="%s" rx="4.5" x="0.5" y="0.5" width="494" height="194"/>
<line x1="165" y1="28" x2="165" y2="170" stroke-width="1" stroke="#E4E2E2"/><line x1="330" y1="28" x2="330" y2="170" stroke-width="1" stroke="#E4E2E2"/>%s</g></svg>`, cardBG, b.String()))
}

var _ = math.Pi
