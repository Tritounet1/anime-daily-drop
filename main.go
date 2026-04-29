package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	scrape "tidy/src"
)

func main() {
	mode := "week"
	if len(os.Args) > 1 {
		mode = strings.ToLower(os.Args[1])
	}

	// Mode schedule
	if mode == "schedule" {
		webhookURL := scrape.GetEnv("DISCORD_WEBHOOK", ".env")
		if webhookURL == "" {
			log.Fatal("DISCORD_WEBHOOK environment variable is missing.")
		}

		paris, _ := time.LoadLocation("Europe/Paris")
		log.Println("Scheduler start, next sendind is tomorrow at 00:01 (Paris)")

		for {
			now := time.Now().In(paris)
			next := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, 0, 0, paris)
			if !next.After(now) {
				next = next.AddDate(0, 0, 1)
			}
			log.Printf("Next message sending : %s", next.Format("2006-01-02 15:04 MST"))
			time.Sleep(time.Until(next))

			if err := sendTodayAnimesToDiscord(webhookURL); err != nil {
				log.Printf("Error while sending to : %v", err)
			}
		}
	}

	entries, err := scrape.FetchSeason()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur API: %v\n", err)
		os.Exit(1)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})

	paris, _ := time.LoadLocation("Europe/Paris")
	todayStr := scrape.DaysFR[time.Now().In(paris).Weekday().String()]

	// Mode Discord (sans scheduler)
	if mode == "discord" {
		webhookURL := scrape.GetEnv("DISCORD_WEBHOOK", ".env")
		if webhookURL == "" {
			fmt.Fprintln(os.Stderr, "DISCORD_WEBHOOK manquant (variable d'environnement ou .env)")
			os.Exit(1)
		}

		filter := "week"
		if len(os.Args) > 2 {
			filter = strings.ToLower(os.Args[2])
		}

		var toSend []scrape.Entry
		for _, e := range entries {
			if filter == "today" && e.Day != todayStr {
				continue
			}
			toSend = append(toSend, e)
		}

		if err := scrape.SendDiscord(webhookURL, toSend, todayStr); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur Discord: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ %d anime(s) envoyé(s) sur Discord\n", len(toSend))
		return
	}

	// Mode terminal
	var toShow []scrape.Entry
	for _, e := range entries {
		if mode == "today" && e.Day != todayStr {
			continue
		}
		toShow = append(toShow, e)
	}

	scrape.PrintTable(toShow, todayStr)

	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run main.go               -> toute la semaine")
	fmt.Println("  go run main.go today         -> seulement aujourd'hui")
	fmt.Println("  go run main.go discord       -> envoyer la semaine sur Discord")
	fmt.Println("  go run main.go discord today -> envoyer aujourd'hui sur Discord")
	fmt.Println("  go run main.go schedule      -> boucle quotidienne (Docker/VPS)")
}

// sendToday récupère les animes du jour et les envoie sur Discord.
func sendTodayAnimesToDiscord(webhookURL string) error {
	entries, err := scrape.FetchSeason()
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})

	paris, _ := time.LoadLocation("Europe/Paris")
	todayStr := scrape.DaysFR[time.Now().In(paris).Weekday().String()]

	var toSend []scrape.Entry
	for _, e := range entries {
		if e.Day == todayStr {
			toSend = append(toSend, e)
		}
	}

	if err := scrape.SendDiscord(webhookURL, toSend, todayStr); err != nil {
		return err
	}
	log.Printf("✓ %d anime(s) send to Discord", len(toSend))
	return nil
}
