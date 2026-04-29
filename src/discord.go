package scrape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	ColorToday  = 0xFFD700 // couleur or pour les animes du jour
	ColorFuture = 0x5865F2 // Discord blurple
)

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordEmbed struct {
	Title     string `json:"title"`
	Color     int    `json:"color"`
	Thumbnail struct {
		URL string `json:"url"`
	} `json:"thumbnail,omitempty"`
	Fields []discordField `json:"fields"`
}

type discordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds"`
}

// GetEnv retourne la valeur d'une variable : os.Getenv en priorité pour Docker et Dokploy,
// puis si on trouve pas on regarde le fichier .env (pour le dev local).
func GetEnv(key, envFile string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

func SendDiscord(webhookURL string, entries []Entry, todayStr string) error {
	var embeds []discordEmbed
	for _, e := range entries {
		color := ColorFuture
		if e.Day == todayStr {
			color = ColorToday
		}
		em := discordEmbed{
			Title: e.Title,
			Color: color,
			Fields: []discordField{
				{Name: "📅 Jour", Value: e.Day, Inline: true},
				{Name: "🕐 Heure (Paris)", Value: e.Time, Inline: true},
			},
		}
		if e.ImgURL != "" {
			em.Thumbnail.URL = e.ImgURL
		}
		embeds = append(embeds, em)
	}

	// Discord est limité à 10 embeds par message.
	for i := 0; i < len(embeds); i += 10 {
		end := i + 10
		if end > len(embeds) {
			end = len(embeds)
		}
		payload := discordPayload{Embeds: embeds[i:end]}
		if i == 0 {
			if len(entries) > 0 && entries[0].Day == todayStr && len(entries) <= 10 {
				payload.Content = "📺 **Animes du jour**"
			} else {
				payload.Content = "📺 **Programme anime de la semaine**"
			}
		}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("discord webhook error: %s", resp.Status)
		}
	}
	return nil
}
