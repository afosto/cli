package data

import "time"

type File struct {
	ID        string            `json:"id"`
	Filename  string            `json:"filename"`
	Label     string            `json:"label"`
	Dir       string            `json:"dir"`
	Type      string            `json:"type"`
	Mime      string            `json:"mime"`
	Url       string            `json:"url"`
	IsPublic  bool              `json:"is_public"`
	IsListed  bool              `json:"is_listed"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type Signature struct {
	ExpiresAt time.Time `json:"expires_at"`
	Signature string    `json:"signature"`
}
