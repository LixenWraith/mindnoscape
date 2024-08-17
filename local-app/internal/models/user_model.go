package models

type User struct {
	Username     string `json:"username"`
	PasswordHash []byte `json:"-"` // The "-" tag ensures this field is not included in JSON output
}

func NewUser(username string) *User {
	return &User{
		Username: username,
	}
}
