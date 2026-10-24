package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var errNoToken = errors.New("GITHUB_TOKEN is not set")

// gql runs a GraphQL query as the token's owner. Answers are cached for an
// hour, and the last good answer is served if GitHub can't be reached, so a
// hiccup never breaks the README.
func (s *server) gql(query string, vars map[string]any, out any) error {
	if s.token == "" {
		return errNoToken
	}
	body, _ := json.Marshal(map[string]any{"query": query, "variables": vars})
	key := string(body)

	s.mu.Lock()
	c, ok := s.cache[key]
	s.mu.Unlock()
	if ok && time.Since(c.at) < cacheLife {
		return json.Unmarshal(c.body, out)
	}

	req, _ := http.NewRequest("POST", s.apiBase+"/graphql", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "profile-cards")
	var buf []byte
	resp, err := s.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = fmt.Errorf("github graphql: %s", resp.Status)
		} else if buf, err = readAll(resp); err == nil {
			var probe struct {
				Errors []struct{ Message string } `json:"errors"`
			}
			if json.Unmarshal(buf, &probe) == nil && len(probe.Errors) > 0 {
				err = fmt.Errorf("github graphql: %s", probe.Errors[0].Message)
			}
		}
	}
	if err != nil {
		if ok {
			return json.Unmarshal(c.body, out)
		}
		return err
	}
	s.mu.Lock()
	s.cache[key] = cached{at: time.Now(), body: buf}
	s.mu.Unlock()
	return json.Unmarshal(buf, out)
}

const profileQuery = `query($login:String!,$from:DateTime!,$to:DateTime!){user(login:$login){
 name login createdAt
 followers{totalCount}
 pullRequests{totalCount}
 issues{totalCount}
 repositoriesContributedTo(first:1,contributionTypes:[COMMIT,ISSUE,PULL_REQUEST,REPOSITORY]){totalCount}
 contributionsCollection(from:$from,to:$to){totalCommitContributions totalPullRequestReviewContributions}
 repositories(first:100,ownerAffiliations:OWNER,isFork:false,orderBy:{field:STARGAZERS,direction:DESC}){nodes{
  stargazerCount
  languages(first:10,orderBy:{field:SIZE,direction:DESC}){edges{size node{name color}}}}}}}`

type profile struct {
	Name          string
	Login         string
	CreatedAt     time.Time
	Followers     int
	PRs           int
	Issues        int
	ContributedTo int
	Commits       int // last year
	Reviews       int // last year
	Stars         int
	Langs         []lang // by bytes of code, biggest first
}

type lang struct {
	Name  string
	Color string
	Size  int
}

func (s *server) profile() (*profile, error) {
	now := time.Now().UTC()
	var r struct {
		Data struct {
			User struct {
				Name                      string
				Login                     string
				CreatedAt                 time.Time
				Followers                 struct{ TotalCount int }
				PullRequests              struct{ TotalCount int }
				Issues                    struct{ TotalCount int }
				RepositoriesContributedTo struct{ TotalCount int }
				ContributionsCollection   struct {
					TotalCommitContributions            int
					TotalPullRequestReviewContributions int
				}
				Repositories struct {
					Nodes []struct {
						StargazerCount int
						Languages      struct {
							Edges []struct {
								Size int
								Node struct{ Name, Color string }
							}
						}
					}
				}
			}
		}
	}
	vars := map[string]any{"login": s.user, "from": now.AddDate(-1, 0, 0).Format("2006-01-02") + "T00:00:00Z", "to": now.Format("2006-01-02") + "T00:00:00Z"}
	if err := s.gql(profileQuery, vars, &r); err != nil {
		return nil, err
	}
	u := r.Data.User
	if u.Login == "" {
		return nil, fmt.Errorf("user %q not found", s.user)
	}
	p := &profile{
		Name: u.Name, Login: u.Login, CreatedAt: u.CreatedAt,
		Followers: u.Followers.TotalCount, PRs: u.PullRequests.TotalCount, Issues: u.Issues.TotalCount,
		ContributedTo: u.RepositoriesContributedTo.TotalCount,
		Commits:       u.ContributionsCollection.TotalCommitContributions,
		Reviews:       u.ContributionsCollection.TotalPullRequestReviewContributions,
	}
	if p.Name == "" {
		p.Name = p.Login
	}
	sizes := map[string]*lang{}
	for _, repo := range u.Repositories.Nodes {
		p.Stars += repo.StargazerCount
		for _, e := range repo.Languages.Edges {
			l := sizes[e.Node.Name]
			if l == nil {
				l = &lang{Name: e.Node.Name, Color: e.Node.Color}
				sizes[e.Node.Name] = l
			}
			l.Size += e.Size
		}
	}
	for _, l := range sizes {
		p.Langs = append(p.Langs, *l)
	}
	sort.Slice(p.Langs, func(i, j int) bool {
		if p.Langs[i].Size != p.Langs[j].Size {
			return p.Langs[i].Size > p.Langs[j].Size
		}
		return p.Langs[i].Name < p.Langs[j].Name
	})
	return p, nil
}

type day struct {
	Date  time.Time
	Count int
}

var (
	dayCell    = regexp.MustCompile(`<td[^>]*?data-date="(\d{4}-\d{2}-\d{2})"[^>]*?id="([^"]+)"`)
	dayTooltip = regexp.MustCompile(`<tool-tip[^>]*?for="([^"]+)"[^>]*>\s*(No|\d+) contributions?\b`)
)

// days is every day's contribution count since the account was created,
// oldest first, read from the public contributions page. That page, unlike the
// GraphQL calendar, draws its days in the account owner's time zone, which is
// how github-readme-streak-stats counts streaks, so the numbers match the old
// card. It needs no token.
func (s *server) days(created time.Time) ([]day, error) {
	now := time.Now().UTC()
	seen := map[string]bool{}
	var out []day
	for y := created.Year(); y <= now.Year(); y++ {
		page, err := s.page(fmt.Sprintf("%s/users/%s/contributions?from=%d-01-01&to=%d-12-31", s.webBase, s.user, y, y))
		if err != nil {
			return nil, err
		}
		counts := map[string]int{}
		for _, m := range dayTooltip.FindAllStringSubmatch(page, -1) {
			n := 0
			if m[2] != "No" {
				n, _ = strconv.Atoi(m[2])
			}
			counts[m[1]] = n
		}
		for _, m := range dayCell.FindAllStringSubmatch(page, -1) {
			t, err := time.Parse("2006-01-02", m[1])
			if err != nil || t.Year() != y || seen[m[1]] || t.After(now) || t.Before(created.Truncate(24*time.Hour)) {
				continue
			}
			seen[m[1]] = true
			out = append(out, day{t, counts[m[2]]})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no contribution days found for %s", s.user)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}

// page fetches a web page with the same hour of caching as the API answers.
func (s *server) page(url string) (string, error) {
	s.mu.Lock()
	c, ok := s.cache[url]
	s.mu.Unlock()
	if ok && time.Since(c.at) < cacheLife {
		return string(c.body), nil
	}
	resp, err := s.client.Get(url)
	var buf []byte
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = fmt.Errorf("%s: %s", url, resp.Status)
		} else {
			buf, err = io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		}
	}
	if err != nil {
		if ok {
			return string(c.body), nil
		}
		return "", err
	}
	s.mu.Lock()
	s.cache[url] = cached{at: time.Now(), body: buf}
	s.mu.Unlock()
	return string(buf), nil
}
