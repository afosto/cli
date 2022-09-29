package data

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
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}

type Signature struct {
	ExpiresAt int64  `json:"expires_at"`
	Signature string `json:"signature"`
}
