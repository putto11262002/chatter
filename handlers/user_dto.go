package handlers

type CreateUserPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func NewCreateUserPayload(username, password, name string) *CreateUserPayload {
	return &CreateUserPayload{
		Username: username,
		Password: password,
		Name:     name,
	}
}
