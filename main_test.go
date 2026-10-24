package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fakeGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/me/contributions" {
			now := time.Now().UTC()
			var b strings.Builder
			// today quiet, then 3 busy days, a quiet day, then a 2-day run
			for i, n := range []int{4, 1, 0, 2, 5, 7, 0} {
				d := now.AddDate(0, 0, i-6)
				if d.Year() != now.Year() {
					continue
				}
				label := "No contributions"
				if n == 1 {
					label = "1 contribution"
				} else if n > 1 {
					label = fmt.Sprintf("%d contributions", n)
				}
				fmt.Fprintf(&b, `<td tabindex="0" style="width: 11px" data-date="%s" id="cell-%d" data-level="1" class="ContributionCalendar-day"></td><tool-tip style="x" for="cell-%d" class="sr-only">%s on a day.</tool-tip>`,
					d.Format("2006-01-02"), i, i, label)
			}
			fmt.Fprint(w, b.String())
			return
		}
		if r.URL.Path != "/graphql" || r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		var q struct{ Query string }
		json.NewDecoder(r.Body).Decode(&q)
		fmt.Fprint(w, `{"data":{"user":{"name":"Test <User>","login":"me","createdAt":"2025-02-07T12:09:46Z",
"followers":{"totalCount":0},"pullRequests":{"totalCount":8},"issues":{"totalCount":1},
"repositoriesContributedTo":{"totalCount":4},
"contributionsCollection":{"totalCommitContributions":426,"totalPullRequestReviewContributions":0},
"repositories":{"nodes":[
 {"stargazerCount":0,"languages":{"edges":[{"size":3000,"node":{"name":"Go","color":"#00ADD8"}},{"size":100,"node":{"name":"Shell","color":"#89e051"}}]}},
 {"stargazerCount":0,"languages":{"edges":[{"size":1000,"node":{"name":"Go","color":"#00ADD8"}},{"size":900,"node":{"name":"Python","color":"#3572A5"}}]}}]}}}}`)
	}))
}

func newTestServer(gh *httptest.Server) *server {
	return &server{user: "me", apiBase: gh.URL, webBase: gh.URL, token: "tok", client: gh.Client(), cache: map[string]cached{}}
}

func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w.Code, w.Body.String()
}

func wellFormed(t *testing.T, svg string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(svg))
	for {
		if _, err := d.Token(); err != nil {
			if err.Error() == "EOF" {
				return
			}
			t.Fatalf("not valid XML: %v\n%s", err, svg)
		}
	}
}

func TestCards(t *testing.T) {
	gh := fakeGitHub(t)
	defer gh.Close()
	h := newTestServer(gh).routes()

	code, body := get(t, h, "/stats.svg")
	for _, want := range []string{"Test &lt;User&gt;&#39;s GitHub Stats", ">426<", ">8<", ">1<", ">4<", "C+"} {
		if code != 200 || !strings.Contains(body, want) {
			t.Fatalf("stats should contain %q: %d\n%s", want, code, body)
		}
	}
	wellFormed(t, body)

	// Go is 4000 of 5000 bytes
	code, body = get(t, h, "/langs.svg")
	if code != 200 || !strings.Contains(body, "Go 80.00%") || !strings.Contains(body, "Python 18.00%") {
		t.Fatalf("langs: %d\n%s", code, body)
	}
	wellFormed(t, body)

	// today is quiet; the 3 days before it (2, 5, 7 contributions) are the
	// current streak, and also the longest
	code, body = get(t, h, "/streak.svg")
	if code != 200 || !strings.Contains(body, ">3<") || !strings.Contains(body, "Current Streak") {
		t.Fatalf("streak: %d\n%s", code, body)
	}
	wellFormed(t, body)

	code, body = get(t, h, "/typing.svg?lines=hello+%26+%3Cworld%3E%7Csecond")
	if code != 200 || strings.Contains(body, "<world>") || !strings.Contains(body, "&lt;world&gt;") {
		t.Fatalf("typing must escape text: %d %s", code, body)
	}
	wellFormed(t, body)
	if code, _ := get(t, h, "/typing.svg"); code != 400 {
		t.Errorf("no lines = %d, want 400", code)
	}
}

func TestNoTokenIsAnError(t *testing.T) {
	s := &server{user: "me", apiBase: "http://127.0.0.1:1", client: http.DefaultClient, cache: map[string]cached{}}
	if code, _ := get(t, s.routes(), "/stats.svg"); code != http.StatusBadGateway {
		t.Fatalf("stats without a token = %d, want 502", code)
	}
}

func TestStaleAnswerWhenGitHubFails(t *testing.T) {
	gh := fakeGitHub(t)
	s := newTestServer(gh)
	if code, _ := get(t, s.routes(), "/stats.svg"); code != 200 {
		t.Fatal("first call should work")
	}
	gh.Close()
	for k, c := range s.cache { // expire everything
		c.at = c.at.Add(-2 * cacheLife)
		s.cache[k] = c
	}
	if code, body := get(t, s.routes(), "/stats.svg"); code != 200 || !strings.Contains(body, "426") {
		t.Fatalf("should serve the last good answer: %d", code)
	}
}

// The grade must match what github-readme-stats gave this profile: 426 commits,
// 8 PRs, 1 issue, nothing else is C+, about 85.6th percentile.
func TestRankMatchesTheOldCard(t *testing.T) {
	level, pct := rank(426, 8, 1, 0, 0, 0)
	if level != "C+" || pct < 85.5 || pct > 85.7 {
		t.Fatalf("rank = %s %.2f, want C+ ~85.6", level, pct)
	}
	if l, _ := rank(5000, 300, 200, 100, 500, 200); l != "S" && l != "A+" {
		t.Errorf("a busy profile graded %s", l)
	}
}

func TestStreaks(t *testing.T) {
	d := func(s string, n int) day {
		tm, _ := time.Parse("2006-01-02", s)
		return day{tm, n}
	}
	days := []day{d("2026-03-01", 1), d("2026-03-02", 2), d("2026-03-03", 1), d("2026-03-04", 0), d("2026-03-05", 3), d("2026-03-06", 0)}
	today, _ := time.Parse("2006-01-02", "2026-03-06") // quiet today: the run that ended yesterday still counts
	s := streaksOf(days, today)
	if s.Total != 7 || s.Longest.Count != 3 || s.Current.Count != 1 || s.Current.String() != "Mar 5" || s.Longest.String() != "Mar 1 - Mar 3" {
		t.Fatalf("%+v / %s / %s", s, s.Current, s.Longest)
	}
	if later, _ := time.Parse("2006-01-02", "2026-03-08"); streaksOf(days, later).Current.Count != 0 {
		t.Error("a streak that ended two days ago is over")
	}
}
