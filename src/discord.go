package scrape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
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

type discordThumbnail struct {
	URL string `json:"url"`
}

type discordEmbed struct {
	Title     string            `json:"title"`
	URL       string            `json:"url,omitempty"`
	Color     int               `json:"color"`
	Thumbnail *discordThumbnail `json:"thumbnail,omitempty"`
	Fields    []discordField    `json:"fields"`
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

// SendDiscord envoie un embed par épisode (par lots de 10, limite Discord).
// Les épisodes diffusés le jour de `now` sont en or, les autres en bleu.
func SendDiscord(webhookURL, header string, entries []Entry, now time.Time) error {
	if len(entries) == 0 {
		return postWebhook(webhookURL, discordPayload{Content: header + "\n_Aucun épisode prévu._"})
	}

	var embeds []discordEmbed
	for _, e := range entries {
		color := ColorFuture
		if e.IsSameDay(now) {
			color = ColorToday
		}
		em := discordEmbed{
			Title: truncate(e.Title, 250),
			URL:   e.URL,
			Color: color,
			Fields: []discordField{
				{Name: "🎬 Épisode", Value: strconv.Itoa(e.Episode), Inline: true},
				{Name: "📅 Jour", Value: e.Day(), Inline: true},
				{Name: "🕐 Heure (Paris)", Value: e.Time(), Inline: true},
			},
		}
		if e.ImgURL != "" {
			em.Thumbnail = &discordThumbnail{URL: e.ImgURL}
		}
		embeds = append(embeds, em)
	}

	for i := 0; i < len(embeds); i += 10 {
		end := min(i+10, len(embeds))
		payload := discordPayload{Embeds: embeds[i:end]}
		if i == 0 {
			payload.Content = header
		}
		if err := postWebhook(webhookURL, payload); err != nil {
			return err
		}
	}
	return nil
}

// postWebhook poste un message en respectant le rate limit Discord (429).
func postWebhook(webhookURL string, payload discordPayload) error {
	body, _ := json.Marshal(payload)

	for attempt := 1; attempt <= 5; attempt++ {
		resp, err := httpClient.Post(webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			var rl struct {
				RetryAfter float64 `json:"retry_after"`
			}
			_ = json.Unmarshal(data, &rl)
			time.Sleep(time.Duration((rl.RetryAfter + 0.5) * float64(time.Second)))
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("discord webhook: %s: %s", resp.Status, truncate(string(data), 200))
		}
		return nil
	}
	return fmt.Errorf("discord webhook: échec après plusieurs tentatives")
}
