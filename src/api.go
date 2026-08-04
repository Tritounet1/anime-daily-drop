package scrape

import (
	"encoding/json"
	"io"
	"net/http"
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
	resp, err := http.Get("https://api.jikan.moe/v4/seasons/now")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var entries []Entry
	seen := make(map[int]bool)
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
	return entries, nil
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
