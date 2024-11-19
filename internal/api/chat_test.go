package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/putto11262002/chatter/internal/api"
	"github.com/putto11262002/chatter/pkg/chat"
	"github.com/putto11262002/chatter/pkg/user"
)

func sendCreatePrivateChatRequest(t *testing.T, client *http.Client, baseUrl string, payload api.CreatePrivateChatPayload) *http.Response {
	body := encodeJsonBody(t, payload)

	url, err := url.JoinPath(baseUrl, "/chats/private")
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Post(url, "application/json", body)
	if err != nil {
		t.Fatal(err)
	}

	return res

}

func sendGetMyUserRoomsIDRequest(t *testing.T, client *http.Client, baseUrl string) *http.Response {
	url, err := url.JoinPath(baseUrl, "/chats/me/rooms")
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	return res
}

func sendGetRoomByIdRequest(t *testing.T, client *http.Client, baseUrl string, id string) *http.Response {
	url, err := url.JoinPath(baseUrl, "/chats/", id)
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	return res
}

func sendSendMessageToRoomRequest(t *testing.T, uc *UserClient, roomID string, payload api.MessageCreateRequest) *http.Response {
	url, err := url.JoinPath(uc.Server.URL, "/chats/", roomID, "/messages")
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Client().Post(url, "application/json", encodeJsonBody(t, payload))

	if err != nil {
		t.Fatal(err)
	}

	return res
}

func sendGetRoomMessagesRequest(t *testing.T, uc *UserClient, roomID string) *http.Response {
	url, err := url.JoinPath(uc.Server.URL, "/chats/rooms/", roomID, "/messages")
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Client().Get(url)
	if err != nil {
		t.Fatal(err)
	}

	return res
}

func Test_CreatePrivateChatHandler(t *testing.T) {
	server, close := setUpTestApiServer(t)
	defer close()

	user1 := NewAuthenticatedUserClient(t, user.User{
		Username: "foo",
		Password: "fooz",
		Name:     "foo",
	}, server)

	user2 := NewAuthenticatedUserClient(t, user.User{
		Username: "bar",
		Password: "barz",
		Name:     "bar",
	}, server)

	tests := []struct {
		name           string
		uc             *UserClient
		payload        api.CreatePrivateChatPayload
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:           "create private chat successfully",
			uc:             user1,
			payload:        api.CreatePrivateChatPayload{Other: user2.User.Username},
			expectedStatus: http.StatusCreated,
			expectedBody:   api.CreateChatResponse{},
		},
		{
			name:           "private chat already exists between users",
			uc:             user1,
			payload:        api.CreatePrivateChatPayload{Other: user2.User.Username},
			expectedStatus: http.StatusConflict,
			expectedBody: api.ApiError[interface{}]{
				Message: "chat already exists",
				Code:    http.StatusConflict,
			},
		},
		{
			name:           "invalid user",
			uc:             user1,
			payload:        api.CreatePrivateChatPayload{Other: "invalid"},
			expectedStatus: http.StatusBadRequest,
			expectedBody: api.ApiError[interface{}]{
				Message: "invalid user",
				Code:    http.StatusBadRequest,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			res := sendCreatePrivateChatRequest(t, tc.uc.Client(), server.URL, tc.payload)

			assert.Equal(t, tc.expectedStatus, res.StatusCode)

			if res.StatusCode == http.StatusCreated {
				var body api.CreateChatResponse
				decodeJsonBody(t, res, &body)
				assert.NoError(t, uuid.Validate(body.ID))
				assert.NotEmpty(t, body.ID)
			} else {
				var body api.ApiError[interface{}]
				decodeJsonBody(t, res, &body)
				assert.Equal(t, tc.expectedBody, body)
			}
		})
	}
}

func Test_GetRoomByIDHandler(t *testing.T) {
	server, close := setUpTestApiServer(t)
	defer close()

	user1 := NewAuthenticatedUserClient(t, user.User{
		Username: "foo",
		Password: "fooz",
		Name:     "foo",
	}, server)

	user2 := NewAuthenticatedUserClient(t, user.User{
		Username: "bar",
		Password: "barz",
		Name:     "bar",
	}, server)

	user3 := NewAuthenticatedUserClient(t, user.User{
		Username: "baz",
		Password: "baz",
		Name:     "baz",
	}, server)

	// Create a private chat between user1 and user2
	res := sendCreatePrivateChatRequest(t, user1.Client(), server.URL, api.CreatePrivateChatPayload{Other: user2.User.Username})
	var room api.CreateChatResponse
	decodeJsonBody(t, res, &room)

	// Define test cases
	tests := []struct {
		name           string
		uc             *UserClient
		roomID         string
		expectedStatus int
		expectedError  *api.ApiError[interface{}]
		expectedBody   *api.RoomResponse
	}{
		{
			name:           "successful get room by id",
			uc:             user1,
			roomID:         room.ID,
			expectedStatus: http.StatusOK,
			expectedBody: &api.RoomResponse{
				ID:   room.ID,
				Type: chat.PrivateChat,
				Users: []api.UserRoomResponse{
					{
						RoomID:   room.ID,
						RoomName: user2.User.Name,
						Username: user1.User.Username,
					},
					{
						RoomID:   room.ID,
						RoomName: user1.User.Name,
						Username: user2.User.Username,
					},
				},
			},
		},
		{
			name:           "room not found",
			uc:             user1,
			roomID:         uuid.New().String(),
			expectedStatus: http.StatusNotFound,
			expectedError: &api.ApiError[interface{}]{
				Message: "room not found",
				Code:    http.StatusNotFound,
			},
		},
		{
			name:           "get other user's room",
			uc:             user3,
			roomID:         room.ID,
			expectedStatus: http.StatusUnauthorized,
			expectedError: &api.ApiError[interface{}]{
				Message: "unauthorized",
				Code:    http.StatusUnauthorized,
			},
		},
	}

	// Execute test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			res := sendGetRoomByIdRequest(t, tc.uc.Client(), server.URL, tc.roomID)
			assert.Equal(t, tc.expectedStatus, res.StatusCode)

			if tc.expectedBody != nil {
				var body api.RoomResponse
				decodeJsonBody(t, res, &body)
				assert.Equal(t, tc.expectedBody.ID, body.ID)
				assert.Equal(t, tc.expectedBody.Type, body.Type)
				assert.ElementsMatch(t, tc.expectedBody.Users, body.Users)
			}

			if tc.expectedError != nil {
				var body api.ApiError[interface{}]
				decodeJsonBody(t, res, &body)
				assert.Equal(t, tc.expectedError.Message, body.Message)
				assert.Equal(t, tc.expectedError.Code, body.Code)
				assert.Nil(t, body.Data)
			}
		})
	}
}

func Test_GetMyUserRoomsHandler(t *testing.T) {

	server, close := setUpTestApiServer(t)
	defer close()

	user1 := NewAuthenticatedUserClient(t, user.User{
		Username: "foo",
		Password: "fooz",
		Name:     "foo",
	}, server)

	user2 := NewAuthenticatedUserClient(t, user.User{
		Username: "bar",
		Password: "barz",
		Name:     "bar",
	}, server)

	user3 := NewAuthenticatedUserClient(t, user.User{
		Username: "baz",
		Password: "baz",
		Name:     "baz",
	}, server)

	var res *http.Response

	var user1user2Room api.RoomResponse
	res = sendCreatePrivateChatRequest(t, user1.Client(), server.URL, api.CreatePrivateChatPayload{Other: user2.User.Username})
	decodeJsonBody(t, res, &user1user2Room)

	var user1user3Room api.RoomResponse
	res = sendCreatePrivateChatRequest(t, user1.Client(), server.URL, api.CreatePrivateChatPayload{Other: user3.User.Username})
	decodeJsonBody(t, res, &user1user3Room)

	tests := []struct {
		name              string
		uc                *UserClient
		expectedStatus    int
		expectedUserRooms []api.UserRoomsResponse
		expectedError     *api.ApiError[interface{}]
	}{
		{
			name:           "get user1 rooms",
			uc:             user1,
			expectedStatus: http.StatusOK,
			expectedUserRooms: []api.UserRoomsResponse{
				{
					RoomID:   user1user2Room.ID,
					RoomName: user2.User.Name,
					Username: user1.User.Username,
				},
				{
					RoomID:   user1user3Room.ID,
					RoomName: user3.User.Name,
					Username: user1.User.Username,
				},
			},
		},
		{
			name:           "get user2 rooms",
			uc:             user2,
			expectedStatus: http.StatusOK,
			expectedUserRooms: []api.UserRoomsResponse{
				{
					RoomID:   user1user2Room.ID,
					Username: user2.User.Username,
					RoomName: user1.User.Name,
				},
			},
		},
		{
			name:           "get user3 rooms",
			uc:             user3,
			expectedStatus: http.StatusOK,
			expectedUserRooms: []api.UserRoomsResponse{
				{
					RoomID:   user1user3Room.ID,
					RoomName: user1.User.Name,
					Username: user3.User.Username,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := sendGetMyUserRoomsIDRequest(t, tc.uc.Client(), server.URL)

			assert.Equal(t, tc.expectedStatus, res.StatusCode)

			var body []api.UserRoomsResponse
			decodeJsonBody(t, res, &body)

			assert.Len(t, body, len(tc.expectedUserRooms))

			assert.ElementsMatch(t, tc.expectedUserRooms, body)
		})
	}

}

func Test_SendMessageToRoomHandler(t *testing.T) {
	server, close := setUpTestApiServer(t)
	defer close()

	user1 := NewAuthenticatedUserClient(t, user.User{
		Username: "foo",
		Password: "fooz",
		Name:     "foo",
	}, server)

	user2 := NewAuthenticatedUserClient(t, user.User{
		Username: "bar",
		Password: "barz",
		Name:     "bar",
	}, server)

	user3 := NewAuthenticatedUserClient(t, user.User{
		Username: "baz",
		Password: "baz",
		Name:     "baz",
	}, server)

	// Create a private chat
	res := sendCreatePrivateChatRequest(t, user1.Client(), server.URL, api.CreatePrivateChatPayload{Other: user2.User.Username})
	assert.Equal(t, http.StatusCreated, res.StatusCode, "failed to create private chat")

	var room api.CreateChatResponse
	decodeJsonBody(t, res, &room)

	// Define test cases
	tests := []struct {
		name         string
		user         *UserClient
		roomID       string
		message      api.MessageCreateRequest
		expectedCode int
		expectedBody interface{}
	}{
		{
			name:   "user1 successful send message",
			user:   user1,
			roomID: room.ID,
			message: api.MessageCreateRequest{
				Type: chat.TextMessage,
				Data: fmt.Sprintf("hello from %s", user1.User.Name),
			},
			expectedCode: http.StatusCreated,
			expectedBody: api.CreateMessageResponse{},
		},
		{
			name:   "user2 successful send message",
			user:   user2,
			roomID: room.ID,
			message: api.MessageCreateRequest{
				Data: fmt.Sprintf("hello from %s", user2.User.Name),
				Type: chat.TextMessage,
			},
			expectedCode: http.StatusCreated,
			expectedBody: api.CreateMessageResponse{},
		},
		{
			name:   "invalid roomID",
			user:   user1,
			roomID: "invalid",
			message: api.MessageCreateRequest{
				Data: "hello",
				Type: chat.TextMessage,
			},
			expectedCode: http.StatusNotFound,
			expectedBody: api.ApiError[interface{}]{Code: http.StatusNotFound, Message: "chat not found"},
		},
		{
			name:         "send to room user is not part of",
			user:         user3,
			roomID:       room.ID,
			message:      api.MessageCreateRequest{Data: "hello", Type: chat.TextMessage},
			expectedCode: http.StatusNotFound,
			expectedBody: api.ApiError[interface{}]{Code: http.StatusNotFound, Message: "chat not found"},
		},
		{
			name:         "invalid message type",
			user:         user1,
			roomID:       room.ID,
			message:      api.MessageCreateRequest{Data: "hello", Type: 99},
			expectedCode: http.StatusBadRequest,
			expectedBody: api.ApiError[interface{}]{Code: http.StatusBadRequest, Message: "invalid message"},
		},
	}

	// Run the tests from the table
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendSendMessageToRoomRequest(t, tt.user, tt.roomID, tt.message)

			assert.Equal(t, tt.expectedCode, res.StatusCode)

			var body interface{}
			decodeJsonBody(t, res, &body)

			// Assert expected response body structure
			switch v := body.(type) {
			case api.CreateMessageResponse:
				assert.NotEmpty(t, v.ID)
				assert.NoError(t, uuid.Validate(v.ID))
			case api.ApiError[interface{}]:
				assert.Equal(t, tt.expectedBody.(api.ApiError[interface{}]).Code, v.Code)
				assert.Equal(t, tt.expectedBody.(api.ApiError[interface{}]).Message, v.Message)
				assert.Nil(t, v.Data)
			}
		})
	}
}

func Test_GetRoomMessagesHandler(t *testing.T) {
	server, close := setUpTestApiServer(t)
	defer close()

	user1 := NewAuthenticatedUserClient(t, user.User{
		Username: "foo",
		Password: "fooz",
		Name:     "foo",
	}, server)

	user2 := NewAuthenticatedUserClient(t, user.User{
		Username: "bar",
		Password: "barz",
		Name:     "bar",
	}, server)

	user3 := NewAuthenticatedUserClient(t, user.User{
		Username: "baz",
		Password: "baz",
		Name:     "baz",
	}, server)

	var res *http.Response

	res = sendCreatePrivateChatRequest(t, user1.Client(), server.URL, api.CreatePrivateChatPayload{Other: user2.User.Username})

	assert.Equal(t, http.StatusCreated, res.StatusCode, "failed to create chat")

	var room api.CreateChatResponse
	decodeJsonBody(t, res, &room)

	msg1 := api.MessageCreateRequest{
		Data: "hello from foo",
		Type: chat.TextMessage,
	}

	res = sendSendMessageToRoomRequest(t, user1, room.ID, msg1)

	msg2 := api.MessageCreateRequest{
		Data: "hello from bar",
		Type: chat.TextMessage,
	}

	res = sendSendMessageToRoomRequest(t, user2, room.ID, msg2)

	tests := []struct {
		name             string
		user             *UserClient
		roomID           string
		expectedStatus   int
		expectedResponse interface{}
	}{
		{
			name:           "get messages successfully",
			user:           user1,
			roomID:         room.ID,
			expectedStatus: http.StatusOK,
			expectedResponse: []api.MessageResponse{
				{
					Data:   msg1.Data,
					Type:   msg1.Type,
					RoomID: room.ID,
					Sender: user1.User.Username,
				},
				{
					Data:   msg2.Data,
					Type:   msg2.Type,
					RoomID: room.ID,
					Sender: user2.User.Username,
				},
			},
		},
		{
			name:             "invalid roomID",
			user:             user1,
			roomID:           "invalid",
			expectedStatus:   http.StatusNotFound,
			expectedResponse: api.ApiError[interface{}]{Code: http.StatusNotFound, Message: "chat not found"},
		},
		{
			name:             "get messages from room user is not part of",
			user:             user3,
			roomID:           room.ID,
			expectedStatus:   http.StatusNotFound,
			expectedResponse: api.ApiError[interface{}]{Code: http.StatusNotFound, Message: "chat not found"},
		},
	}

	for _, tc := range tests {
		res := sendGetRoomMessagesRequest(t, tc.user, tc.roomID)

		assert.Equal(t, tc.expectedStatus, res.StatusCode)

		if tc.expectedStatus < 400 {
			var body []api.MessageResponse
			decodeJsonBody(t, res, &body)
			assert.ElementsMatch(t, tc.expectedResponse.([]api.MessageResponse), body)
		} else {
			var body api.ApiError[interface{}]
			decodeJsonBody(t, res, &body)
			assert.Equal(t, tc.expectedResponse.(api.ApiError[interface{}]).Code, body.Code)
			assert.Equal(t, tc.expectedResponse.(api.ApiError[interface{}]).Message, body.Message)
			assert.Nil(t, body.Data)
		}

	}

}
