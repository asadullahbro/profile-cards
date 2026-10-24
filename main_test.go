package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/repos") || r.URL.Path == "/users/me/repos":
			fmt.Fprint(w, `[{"name":"a","stargazers_count":2},{"name":"b"},{"name":"c"},{"name":"forked","fork":true}]`)
		case r.URL.Path == "/repos/me/a/languages":
			fmt.Fprint(w, `{"Go":2000,"Shell":100}`)
		case r.URL.Path == "/repos/me/b/languages":
			fmt.Fprint(w, `{"Go":1000}`)
		case r.URL.Path == "/repos/me/c/languages":
			fmt.Fprint(w, `{"Python":900}`)
		case r.URL.Path == "/repos/me/forked/languages":
			fmt.Fprint(w, `{"Kotlin":99999}`)
		case r.URL.Path == "/users/me":
			fmt.Fprint(w, `{"followers":3,"created_at":"2025-02-07T12:09:46Z"}`)
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
		default:
			http.NotFound(w, r)
		}
	}))
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
	s := &server{user: "me", apiBase: gh.URL, hubRepo: "me/hub", client: gh.Client(), cache: map[string]cached{}}
	h := s.routes()

	code, body := get(t, h, "/stats.svg")
	if code != 200 || !strings.Contains(body, "v1.2.3") || !strings.Contains(body, ">3<") {
		t.Fatalf("stats: %d %s", code, body)
	}
	if !strings.Contains(body, ">3<") || strings.Contains(body, "Kotlin") {
		t.Errorf("stats counted wrong")
	}
	wellFormed(t, body)

	code, body = get(t, h, "/langs.svg")
	if code != 200 || !strings.Contains(body, "Go") || !strings.Contains(body, "75%") || strings.Contains(body, "Kotlin") {
		t.Fatalf("langs should count own repos only (Go 3000 of 4000 bytes): %d %s", code, body)
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

func TestStaleAnswerWhenGitHubFails(t *testing.T) {
	gh := fakeGitHub(t)
	s := &server{user: "me", apiBase: gh.URL, hubRepo: "me/hub", client: gh.Client(), cache: map[string]cached{}}
	if code, _ := get(t, s.routes(), "/langs.svg"); code != 200 {
		t.Fatal("first call should work")
	}
	gh.Close()
	for k, c := range s.cache { // expire everything
		c.at = c.at.Add(-2 * cacheLife)
		s.cache[k] = c
	}
	if code, body := get(t, s.routes(), "/langs.svg"); code != 200 || !strings.Contains(body, "Go") {
		t.Fatalf("should serve the last good answer: %d", code)
	}
}
