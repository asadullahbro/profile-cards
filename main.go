// profile-cards serves the small SVG widgets on the GitHub profile README
// (typing line, stats, languages), so the README depends on nothing but this.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const cacheLife = time.Hour

type server struct {
	user    string
	apiBase string
	webBase string
	token   string
	client  *http.Client

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	at   time.Time
	body []byte
}

func main() {
	s := &server{
		user:    env("GITHUB_USER", "asadullahbro"),
		apiBase: env("GITHUB_API", "https://api.github.com"),
		webBase: env("GITHUB_WEB", "https://github.com"),
		token:   os.Getenv("GITHUB_TOKEN"), // optional: only raises the rate limit
		client:  &http.Client{Timeout: 10 * time.Second},
		cache:   map[string]cached{},
	}
	addr := ":" + env("PORT", "8000")
	log.Printf("profile-cards for %s on %s", s.user, addr)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.HandleFunc("/typing.svg", s.typing)
	mux.HandleFunc("/stats.svg", s.stats)
	mux.HandleFunc("/langs.svg", s.langs)
	mux.HandleFunc("/streak.svg", s.streak)
	return mux
}

func writeSVG(w http.ResponseWriter, svg []byte) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(svg)
}

func (s *server) langs(w http.ResponseWriter, r *http.Request) {
	p, err := s.profile()
	if err != nil || len(p.Langs) == 0 {
		s.fail(w, err)
		return
	}
	writeSVG(w, langsCard(p.Langs))
}

func (s *server) stats(w http.ResponseWriter, r *http.Request) {
	p, err := s.profile()
	if err != nil {
		s.fail(w, err)
		return
	}
	writeSVG(w, statsCard(p))
}

func (s *server) streak(w http.ResponseWriter, r *http.Request) {
	created := time.Date(2025, 2, 7, 0, 0, 0, 0, time.UTC) // account creation date, if the API is unavailable
	if p, err := s.profile(); err == nil {
		created = p.CreatedAt
	}
	days, err := s.days(created)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeSVG(w, streakCard(streaksOf(days, time.Now().UTC())))
}

func (s *server) fail(w http.ResponseWriter, err error) {
	if err != nil {
		log.Printf("card: %v", err)
	}
	http.Error(w, "github unavailable", http.StatusBadGateway)
}

func (s *server) typing(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var lines []string
	for _, l := range strings.Split(q.Get("lines"), "|") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 || len(lines) > 8 {
		http.Error(w, "give 1-8 lines, separated by |", http.StatusBadRequest)
		return
	}
	width := clamp(atoi(q.Get("width"), 435), 100, 1000)
	writeSVG(w, typingSVG(lines, width))
}

func atoi(s string, d int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return d
}

func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}
