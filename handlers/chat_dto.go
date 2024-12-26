package handlers

type CreatePrivateChatPayload struct {
	Other string `json:"other"`
}

type CreateGroupChatPayload struct {
	Members []string `json:"members"`
	Name    string   `json:"name"`
}

type CreateChatResponse struct {
	ID string `json:"id"`
}

func NewCreateChatResponse(id string) *CreateChatResponse {
	return &CreateChatResponse{
		ID: id,
	}
}
