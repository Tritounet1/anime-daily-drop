package scrape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// AniListURL est l'endpoint GraphQL d'AniList (variable pour les tests).
var AniListURL = "https://graphql.anilist.co"

var httpClient = &http.Client{Timeout: 30 * time.Second}

var Paris = mustLoadLocation("Europe/Paris")

var DaysFR = map[time.Weekday]string{
	time.Monday: "Lundi", time.Tuesday: "Mardi", time.Wednesday: "Mercredi",
	time.Thursday: "Jeudi", time.Friday: "Vendredi", time.Saturday: "Samedi", time.Sunday: "Dimanche",
}

type Entry struct {
	Title   string
	Episode int
	Date    time.Time // heure de diffusion, en heure de Paris
	ImgURL  string
	URL     string
}

func (e Entry) Day() string  { return DaysFR[e.Date.Weekday()] }
func (e Entry) Time() string { return e.Date.Format("15:04") }

// IsSameDay indique si e est diffusé le même jour (calendaire, Paris) que t.
func (e Entry) IsSameDay(t time.Time) bool {
	y1, m1, d1 := e.Date.In(Paris).Date()
	y2, m2, d2 := t.In(Paris).Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

const airingQuery = `query ($page: Int, $from: Int, $to: Int) {
  Page(page: $page, perPage: 50) {
    pageInfo { hasNextPage }
    airingSchedules(airingAt_greater: $from, airingAt_lesser: $to, sort: TIME) {
      airingAt
      episode
      media {
        id
        title { romaji english }
        coverImage { extraLarge large }
        siteUrl
        isAdult
        countryOfOrigin
        format
      }
    }
  }
}`

type airingResponse struct {
	Data struct {
		Page struct {
			PageInfo struct {
				HasNextPage bool `json:"hasNextPage"`
			} `json:"pageInfo"`
			AiringSchedules []struct {
				AiringAt int64 `json:"airingAt"`
				Episode  int   `json:"episode"`
				Media    struct {
					ID    int `json:"id"`
					Title struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
					} `json:"title"`
					CoverImage struct {
						ExtraLarge string `json:"extraLarge"`
						Large      string `json:"large"`
					} `json:"coverImage"`
					SiteURL         string `json:"siteUrl"`
					IsAdult         bool   `json:"isAdult"`
					CountryOfOrigin string `json:"countryOfOrigin"`
					Format          string `json:"format"`
				} `json:"media"`
			} `json:"airingSchedules"`
		} `json:"Page"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FetchAiring retourne les épisodes d'animes japonais diffusés dans [from, to),
// triés par heure de diffusion.
func FetchAiring(from, to time.Time) ([]Entry, error) {
	var entries []Entry
	seen := make(map[string]bool)

	for page := 1; ; page++ {
		res, err := fetchPage(page, from, to)
		if err != nil {
			return nil, fmt.Errorf("anilist page %d: %w", page, err)
		}

		for _, s := range res.Data.Page.AiringSchedules {
			m := s.Media
			if m.IsAdult || m.CountryOfOrigin != "JP" || m.Format == "MUSIC" {
				continue
			}
			key := fmt.Sprintf("%d-%d", m.ID, s.Episode)
			if seen[key] {
				continue
			}
			seen[key] = true

			img := m.CoverImage.ExtraLarge
			if img == "" {
				img = m.CoverImage.Large
			}
			entries = append(entries, Entry{
				Title:   formatTitle(m.Title.Romaji, m.Title.English),
				Episode: s.Episode,
				Date:    time.Unix(s.AiringAt, 0).In(Paris),
				ImgURL:  img,
				URL:     m.SiteURL,
			})
		}

		if !res.Data.Page.PageInfo.HasNextPage {
			break
		}
	}
	return entries, nil
}

// fetchPage interroge AniList en réessayant sur les erreurs transitoires
// (réseau, 429 rate limit, 5xx).
func fetchPage(page int, from, to time.Time) (airingResponse, error) {
	const maxAttempts = 5
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		res, wait, err := doFetchPage(page, from, to)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if wait < 0 || attempt == maxAttempts {
			break
		}
		if wait == 0 {
			wait = time.Duration(attempt*attempt) * time.Second
		}
		time.Sleep(wait)
	}
	return airingResponse{}, lastErr
}

// doFetchPage renvoie wait < 0 si l'erreur n'est pas réessayable,
// wait > 0 si le serveur a indiqué un délai (Retry-After).
func doFetchPage(page int, from, to time.Time) (res airingResponse, wait time.Duration, err error) {
	body, _ := json.Marshal(map[string]any{
		"query": airingQuery,
		"variables": map[string]any{
			"page": page,
			// airingAt_greater est strict : -1 pour inclure `from`.
			"from": from.Unix() - 1,
			"to":   to.Unix(),
		},
	})

	req, err := http.NewRequest(http.MethodPost, AniListURL, bytes.NewReader(body))
	if err != nil {
		return res, -1, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return res, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, 0, err
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		if secs, convErr := strconv.Atoi(resp.Header.Get("Retry-After")); convErr == nil {
			wait = time.Duration(secs) * time.Second
		}
		return res, wait, fmt.Errorf("http %s", resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return res, -1, fmt.Errorf("http %s: %s", resp.Status, truncate(string(data), 200))
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return res, -1, err
	}
	if len(res.Errors) > 0 {
		return res, -1, fmt.Errorf("graphql: %s", res.Errors[0].Message)
	}
	return res, 0, nil
}

func formatTitle(romaji, english string) string {
	if english == "" || english == romaji {
		return romaji
	}
	return english + " | " + romaji
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
