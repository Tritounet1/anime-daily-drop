package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // fuseaux horaires embarqués (pas besoin de tzdata dans l'image)

	scrape "tidy/src"
)

func main() {
	args := make([]string, 0, len(os.Args))
	for _, a := range os.Args[1:] {
		args = append(args, strings.ToLower(a))
	}

	switch {
	case len(args) > 0 && args[0] == "schedule":
		runScheduler()

	case len(args) > 0 && args[0] == "discord":
		today := len(args) > 1 && args[1] == "today"
		if err := sendToDiscord(today); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}

	case len(args) == 0 || args[0] == "week" || args[0] == "today":
		now := time.Now().In(scrape.Paris)
		from, to := period(now, len(args) > 0 && args[0] == "today")
		entries, err := scrape.FetchAiring(from, to)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur API: %v\n", err)
			os.Exit(1)
		}
		scrape.PrintTable(entries, now)

	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run .               -> les 7 prochains jours")
	fmt.Println("  go run . today         -> seulement aujourd'hui")
	fmt.Println("  go run . discord       -> envoyer les 7 prochains jours sur Discord")
	fmt.Println("  go run . discord today -> envoyer aujourd'hui sur Discord")
	fmt.Println("  go run . schedule      -> envoi quotidien à 00:01 (Docker/VPS)")
}

// period renvoie la plage [from, to) : la journée en cours (Paris) ou 7 jours
// à partir de minuit aujourd'hui.
func period(now time.Time, today bool) (time.Time, time.Time) {
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, scrape.Paris)
	if today {
		return from, from.AddDate(0, 0, 1)
	}
	return from, from.AddDate(0, 0, 7)
}

func sendToDiscord(today bool) error {
	webhookURL := scrape.GetEnv("DISCORD_WEBHOOK", ".env")
	if webhookURL == "" {
		return fmt.Errorf("DISCORD_WEBHOOK manquant (variable d'environnement ou .env)")
	}

	now := time.Now().In(scrape.Paris)
	from, to := period(now, today)
	entries, err := scrape.FetchAiring(from, to)
	if err != nil {
		return err
	}

	header := "📺 **Programme anime des 7 prochains jours**"
	if today {
		header = fmt.Sprintf("📺 **Animes du jour — %s %s**", scrape.DaysFR[now.Weekday()], now.Format("02/01"))
	}
	if err := scrape.SendDiscord(webhookURL, header, entries, now); err != nil {
		return err
	}
	log.Printf("✓ %d épisode(s) envoyé(s) sur Discord", len(entries))
	return nil
}

func runScheduler() {
	if scrape.GetEnv("DISCORD_WEBHOOK", ".env") == "" {
		log.Fatal("DISCORD_WEBHOOK manquant (variable d'environnement ou .env)")
	}
	log.Println("Scheduler démarré, envoi chaque jour à 00:01 (Paris)")

	for {
		now := time.Now().In(scrape.Paris)
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, 0, 0, scrape.Paris)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		log.Printf("Prochain envoi : %s", next.Format("2006-01-02 15:04 MST"))
		time.Sleep(time.Until(next))

		// En cas d'échec (API indisponible…), on réessaie toutes les 15 min
		// pendant 2 h plutôt que de sauter la journée.
		const retries = 8
		for attempt := 1; attempt <= retries; attempt++ {
			err := sendToDiscord(true)
			if err == nil {
				break
			}
			log.Printf("Échec de l'envoi (tentative %d/%d) : %v", attempt, retries, err)
			if attempt < retries {
				time.Sleep(15 * time.Minute)
			}
		}
	}
}
