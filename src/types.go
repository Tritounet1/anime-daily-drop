package scrape

import "time"

type Response struct {
	Data       []Anime    `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	HasNextPage bool `json:"has_next_page"`
}

type Anime struct {
	MalID        int       `json:"mal_id"`
	Title        string    `json:"title"`
	TitleEnglish string    `json:"title_english"`
	Images       Images    `json:"images"`
	Broadcast    Broadcast `json:"broadcast"`
}

type Images struct {
	JPG struct {
		LargeURL string `json:"large_image_url"`
	} `json:"jpg"`
}

type Broadcast struct {
	Day  string `json:"day"`
	Time string `json:"time"`
}

type Entry struct {
	Title  string
	Date   time.Time
	Day    string
	Time   string
	ImgURL string
}
