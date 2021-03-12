package data

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	TenantID    string `json:"tenant_id"`
	TenantName  string `json:"tenant_name"`
	accessToken string
}

func (u *User) SetAccessToken(token string) {
	u.accessToken = token
}

func (u *User) GetAccessToken() string {
	return u.accessToken
}
