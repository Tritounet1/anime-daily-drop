# Anime Daily Drop

Programme en Go pour récupérer le planning des animes de la saison en cours, avec envoie du planning sur Discord (Webhook) ou accessible dans le terminal.

## Fonctionnement

Le programme utilise l'**[API GraphQL d'AniList](https://docs.anilist.co/)** (gratuite, sans inscription) pour récupérer le planning de diffusion réel de chaque épisode (`airingSchedules`). Les horaires sont des timestamps exacts, convertis en heure de Paris (CET/CEST). La nouvelle saison est prise en compte automatiquement.

Seuls les animes japonais non-adultes sont gardés. Les erreurs transitoires (rate limit, 5xx, réseau) sont réessayées, et en mode `schedule` un envoi raté est retenté toutes les 15 min pendant 2 h.

## Utilisation

```bash
go run .               # les 7 prochains jours
go run . today         # épisodes diffusés aujourd'hui
go run . discord       # envoyer les 7 prochains jours sur Discord
go run . discord today # envoyer aujourd'hui sur Discord
go run . schedule      # envoi quotidien à 00:01 (Paris), utilisé dans Docker
go test ./...          # tests
```

## Notifications Discord

Crée le fichier `.env` à la racine du projet :

```sh
cp .env.example .env
```

Et compléter le avec le webhook du salon de votre serveur discord :

```
DISCORD_WEBHOOK="https://discord.com/api/webhooks/..."
```

Le webhook envoie un embed par épisode avec l'image en haute résolution, le numéro d'épisode, le jour et l'heure de diffusion (Paris), et un lien vers la fiche AniList.

Les animes du jour sont mis en couleur or, les autres en bleu.

## Exemple de notification discord

![Discord notification image](assets/discord-img.png "Discord notification image")
