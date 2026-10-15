package scrape

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func schedule(airingAt int64, ep, id int, romaji, english, country string, adult bool) map[string]any {
	return map[string]any{
		"airingAt": airingAt,
		"episode":  ep,
		"media": map[string]any{
			"id":              id,
			"title":           map[string]any{"romaji": romaji, "english": english},
			"coverImage":      map[string]any{"extraLarge": "https://img/" + romaji, "large": ""},
			"siteUrl":         "https://anilist.co/anime/1",
			"isAdult":         adult,
			"countryOfOrigin": country,
			"format":          "TV",
		},
	}
}

func TestFetchAiringPaginatesAndFilters(t *testing.T) {
	base := time.Date(2026, 10, 15, 0, 0, 0, 0, Paris)
	at := base.Add(16*time.Hour + 30*time.Minute).Unix()

	pages := map[int][]map[string]any{
		1: {
			schedule(at, 3, 1, "Sousou no Frieren", "Frieren", "JP", false),
			schedule(at, 1, 2, "Some Donghua", "", "CN", false),
			schedule(at, 1, 3, "Adult Show", "", "JP", true),
		},
		2: {
			schedule(at+3600, 12, 4, "One Piece", "One Piece", "JP", false),
			schedule(at, 3, 1, "Sousou no Frieren", "Frieren", "JP", false), // doublon
		},
	}

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			// Première requête : rate limit, doit être réessayée.
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		var req struct {
			Variables struct {
				Page int `json:"page"`
			} `json:"variables"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		p := req.Variables.Page
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"Page": map[string]any{
				"pageInfo":        map[string]any{"hasNextPage": p == 1},
				"airingSchedules": pages[p],
			}},
		})
	}))
	defer srv.Close()
	AniListURL = srv.URL

	entries, err := FetchAiring(base, base.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}
	e := entries[0]
	if e.Title != "Frieren | Sousou no Frieren" || e.Episode != 3 {
		t.Errorf("unexpected entry: %+v", e)
	}
	if e.Day() != "Jeudi" || e.Time() != "16:30" || !e.IsSameDay(base) {
		t.Errorf("unexpected date: %s %s", e.Day(), e.Time())
	}
	if entries[1].Title != "One Piece" {
		t.Errorf("unexpected title: %q", entries[1].Title)
	}
}

func TestFetchAiringGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":null,"errors":[{"message":"boom"}]}`))
	}))
	defer srv.Close()
	AniListURL = srv.URL

	if _, err := FetchAiring(time.Now(), time.Now().Add(time.Hour)); err == nil {
		t.Fatal("expected error")
	}
}
