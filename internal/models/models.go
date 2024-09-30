package models

type Song struct {
	GroupName   string `json:"group" db:"group_name"`
	SongName    string `json:"song" db:"song_name"`
	ReleaseDate string `json:"releaseDate" db:"release_date"`
	SongText    string `json:"songText" db:"song_text"`
	Link        string `json:"link" db:"link"`
}
