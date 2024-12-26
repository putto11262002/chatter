package models

type User struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserWithoutSecrets struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}
