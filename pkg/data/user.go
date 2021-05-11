package data

import (
	"encoding/json"
)

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

func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		AccessToken string `json:"token"`
		*Alias
	}{
		AccessToken: u.GetAccessToken(),
		Alias:       (*Alias)(u),
	})
}

func (u *User) UnmarshalJSON(data []byte) error {
	type Alias User
	aux := &struct {
		AccessToken string `json:"token"`
		*Alias
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	u.accessToken = aux.AccessToken
	return nil
}
