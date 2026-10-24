package main

import (
	"fmt"
	"time"
)

type streak struct {
	Count      int
	Start, End time.Time
}

type streaks struct {
	Total            int
	Since            time.Time
	Current, Longest streak
}

// streaksOf finds the current and longest runs of days with a contribution.
// A quiet today doesn't end the current streak: it is still going until the
// day is over.
func streaksOf(days []day, today time.Time) streaks {
	var s streaks
	if len(days) == 0 {
		return s
	}
	s.Since = days[0].Date
	var run streak
	for _, d := range days {
		s.Total += d.Count
		if d.Count > 0 {
			if run.Count > 0 && d.Date.Sub(run.End) == 24*time.Hour {
				run.Count++
				run.End = d.Date
			} else {
				run = streak{1, d.Date, d.Date}
			}
			if run.Count > s.Longest.Count {
				s.Longest = run
			}
		}
	}
	// the run that ends today or yesterday is the current one
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if run.Count > 0 && today.Sub(run.End) <= 24*time.Hour {
		s.Current = run
	}
	return s
}

// span is a date range the way the old card wrote it: the year only when it
// differs between the two ends.
func span(a, b time.Time) string {
	if a.Equal(b) {
		return a.Format("Jan 2")
	}
	if a.Year() == b.Year() {
		return a.Format("Jan 2") + " - " + b.Format("Jan 2")
	}
	return a.Format("Jan 2, 2006") + " - " + b.Format("Jan 2, 2006")
}

func (r streak) String() string {
	if r.Count == 0 {
		return "No streak yet"
	}
	return span(r.Start, r.End)
}

func comma(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
