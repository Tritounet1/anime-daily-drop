package scrape

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var DaysFR = map[string]string{
	"Monday": "Lundi", "Tuesday": "Mardi", "Wednesday": "Mercredi",
	"Thursday": "Jeudi", "Friday": "Vendredi", "Saturday": "Samedi", "Sunday": "Dimanche",
}

var weekdayMap = map[string]time.Weekday{
	"Sundays": time.Sunday, "Mondays": time.Monday, "Tuesdays": time.Tuesday,
	"Wednesdays": time.Wednesday, "Thursdays": time.Thursday,
	"Fridays": time.Friday, "Saturdays": time.Saturday,
}

func formatTitle(original, english string) string {
	if english == "" || english == original {
		return original
	}
	return english + " | " + original
}

func FetchSeason() ([]Entry, error) {
	var entries []Entry
	seen := make(map[int]bool)

	for page := 1; ; page++ {
		result, err := fetchSeasonPage(page)
		if err != nil {
			// Jikan's seasons/now endpoint is prone to sustained upstream
			// timeouts on deeper pages. Fail hard only if we have nothing
			// at all; otherwise return what was fetched so far rather than
			// discarding already-successful pages.
			if page == 1 {
				return nil, err
			}
			fmt.Fprintf(os.Stderr, "Attention: liste possiblement incomplète (%v)\n", err)
			break
		}

		for _, a := range result.Data {
			if a.Broadcast.Day == "" || seen[a.MalID] {
				continue
			}
			seen[a.MalID] = true
			date, day, t := GetNextAiring(a.Broadcast.Day, a.Broadcast.Time)
			if !date.IsZero() {
				entries = append(entries, Entry{
					Title:  formatTitle(a.Title, a.TitleEnglish),
					Date:   date,
					Day:    day,
					Time:   t,
					ImgURL: a.Images.JPG.LargeURL,
				})
			}
		}

		if !result.Pagination.HasNextPage {
			break
		}
		// Jikan rate-limits to 3 req/s.
		time.Sleep(400 * time.Millisecond)
	}

	return entries, nil
}

// fetchSeasonPage fetches one page, retrying on transient errors: Jikan's
// seasons/now endpoint frequently returns 429 (rate limit) or 5xx (upstream
// MAL timeout) on deeper pages even at a moderate request rate.
func fetchSeasonPage(page int) (Response, error) {
	const maxAttempts = 4
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, retryable, err := doFetchSeasonPage(page)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable || attempt == maxAttempts {
			break
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return Response{}, fmt.Errorf("page %d: %w", page, lastErr)
}

func doFetchSeasonPage(page int) (result Response, retryable bool, err error) {
	resp, err := http.Get(fmt.Sprintf("https://api.jikan.moe/v4/seasons/now?page=%d", page))
	if err != nil {
		return Response{}, true, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, true, err
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return Response{}, true, fmt.Errorf("http %s", resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return Response{}, false, fmt.Errorf("http %s", resp.Status)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return Response{}, false, err
	}
	return result, false, nil
}

// GetNextAiring converts a JST broadcast slot to the next upcoming Paris time.
func GetNextAiring(day, timeStr string) (time.Time, string, string) {
	if day == "" || timeStr == "" {
		return time.Time{}, "", ""
	}
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, day, timeStr
	}

	targetDay, ok := weekdayMap[day]
	if !ok {
		return time.Time{}, "", ""
	}

	jst, _ := time.LoadLocation("Asia/Tokyo")
	paris, _ := time.LoadLocation("Europe/Paris")

	nowJST := time.Now().In(jst)
	diff := int(targetDay - nowJST.Weekday())
	if diff < 0 {
		diff += 7
	}

	jstDate := nowJST.AddDate(0, 0, diff)
	jstTime := time.Date(jstDate.Year(), jstDate.Month(), jstDate.Day(),
		t.Hour(), t.Minute(), 0, 0, jst)

	if jstTime.Before(time.Now()) {
		jstTime = jstTime.AddDate(0, 0, 7)
	}

	parisTime := jstTime.In(paris)
	return parisTime, DaysFR[parisTime.Weekday().String()], parisTime.Format("15:04")
}
