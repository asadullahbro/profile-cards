// profile-cards serves the small SVG widgets on the GitHub profile README
// (typing line, stats, languages), so the README depends on nothing but this.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const cacheLife = time.Hour

type server struct {
	user    string
	apiBase string
	token   string
	hubRepo string
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
		token:   os.Getenv("GITHUB_TOKEN"), // optional: only raises the rate limit
		hubRepo: env("HUB_REPO", "asadullahbro/ASDL-Hub"),
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
	return mux
}

func writeSVG(w http.ResponseWriter, svg []byte) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(svg)
}

// get fetches a GitHub API path as JSON into v. A failed fetch falls back to
// the last good answer, so a GitHub hiccup never breaks the README.
func (s *server) get(path string, v any) error {
	s.mu.Lock()
	c, ok := s.cache[path]
	s.mu.Unlock()
	if ok && time.Since(c.at) < cacheLife {
		return json.Unmarshal(c.body, v)
	}
	req, _ := http.NewRequest("GET", s.apiBase+path, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "profile-cards")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = fmt.Errorf("github %s: %s", path, resp.Status)
		}
	}
	if err != nil {
		if ok {
			return json.Unmarshal(c.body, v)
		}
		return err
	}
	var buf []byte
	buf, err = readAll(resp)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cache[path] = cached{at: time.Now(), body: buf}
	s.mu.Unlock()
	return json.Unmarshal(buf, v)
}

type repo struct {
	Name     string `json:"name"`
	Fork     bool   `json:"fork"`
	Language string `json:"language"`
	Stars    int    `json:"stargazers_count"`
}

func (s *server) repos() ([]repo, error) {
	var all []repo
	if err := s.get("/users/"+s.user+"/repos?per_page=100&type=owner", &all); err != nil {
		return nil, err
	}
	var own []repo
	for _, r := range all {
		if !r.Fork {
			own = append(own, r)
		}
	}
	return own, nil
}

func (s *server) langs(w http.ResponseWriter, r *http.Request) {
	repos, err := s.repos()
	if err != nil {
		http.Error(w, "github unavailable", http.StatusBadGateway)
		return
	}
	// Bytes of code per language, over my own repos, so a big project counts
	// for more than a small script. A repo whose answer fails is skipped.
	perRepo := make([]map[string]int, len(repos))
	var wg sync.WaitGroup
	for i, rp := range repos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var bytes map[string]int
			if err := s.get("/repos/"+s.user+"/"+rp.Name+"/languages", &bytes); err == nil {
				perRepo[i] = bytes
			}
		}()
	}
	wg.Wait()
	count := map[string]int{}
	total := 0
	for _, bytes := range perRepo {
		for lang, n := range bytes {
			count[lang] += n
			total += n
		}
	}
	if total == 0 {
		http.Error(w, "no language data", http.StatusBadGateway)
		return
	}
	type kv struct {
		name string
		n    int
	}
	var list []kv
	for k, n := range count {
		list = append(list, kv{k, n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n != list[j].n {
			return list[i].n > list[j].n
		}
		return list[i].name < list[j].name
	})
	if len(list) > 6 {
		list = list[:6]
	}
	rows := make([]row, 0, len(list))
	for _, e := range list {
		rows = append(rows, row{label: e.name, pct: float64(e.n) / float64(total), color: langColor(e.name)})
	}
	writeSVG(w, langsSVG("Top languages", rows))
}

func (s *server) stats(w http.ResponseWriter, r *http.Request) {
	repos, err := s.repos()
	if err != nil {
		http.Error(w, "github unavailable", http.StatusBadGateway)
		return
	}
	var u struct {
		Followers int    `json:"followers"`
		Created   string `json:"created_at"`
	}
	if err := s.get("/users/"+s.user, &u); err != nil {
		http.Error(w, "github unavailable", http.StatusBadGateway)
		return
	}
	stars := 0
	for _, rp := range repos {
		stars += rp.Stars
	}
	since := u.Created
	if len(since) >= 4 {
		since = since[:4]
	}
	items := [][2]string{
		{"Public repos", strconv.Itoa(len(repos))},
		{"Stars earned", strconv.Itoa(stars)},
		{"Followers", strconv.Itoa(u.Followers)},
		{"On GitHub since", since},
	}
	var rel struct {
		Tag string `json:"tag_name"`
	}
	if err := s.get("/repos/"+s.hubRepo+"/releases/latest", &rel); err == nil && rel.Tag != "" {
		name := s.hubRepo[strings.Index(s.hubRepo, "/")+1:]
		items = append(items, [2]string{name + " release", rel.Tag})
	}
	writeSVG(w, statsSVG(s.user, items))
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
