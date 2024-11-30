package chat

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/putto11262002/chatter/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setUp(t *testing.T) (user.UserStore, ChatStore, func()) {

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}

	migrationFS := os.DirFS("../../migrations")
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	if err := goose.Up(db, "."); err != nil {
		t.Fatal(err)
	}

	userStore := user.NewSQLiteUserStore(db)
	chatStore := NewSQLiteChatStore(db, userStore)

	return userStore, chatStore, func() {
		db.Close()
	}
}

var (
	users = []user.User{
		{Username: "user1", Password: "password", Name: "User 1"},
		{Username: "user2", Password: "password", Name: "User 2"},
		{Username: "user3", Password: "password", Name: "User 3"},
		{Username: "user4", Password: "password", Name: "User 4"},
	}
)

func Test_CreatePrivateChat(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	user1 := users[0]
	user2 := users[1]
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, u := range users[:2] {
		if err := userStore.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%+v): %v", u, err)
		}
	}
	t.Run("create_private_chat_with_duplicate_users", func(t *testing.T) {
		id, err := chatStore.CreatePrivateChat(ctx, [2]string{user1.Username, user1.Username})
		require.Equal(t, ErrInvalidUser, err)
		require.Equal(t, "", id)
	})
	t.Run("create_private_chat_with_invalid_user_id", func(t *testing.T) {
		id, err := chatStore.CreatePrivateChat(ctx, [2]string{user1.Username, "invalid"})
		require.Equal(t, ErrInvalidUser, err)
		require.Equal(t, "", id)
	})
	t.Run("create_private_chat_successfully", func(t *testing.T) {
		id, err := chatStore.CreatePrivateChat(ctx, [2]string{user1.Username, user2.Username})
		require.Nil(t, err)
		require.NotEmpty(t, id)
		room, err := chatStore.GetRoomByID(ctx, id)
		require.Nil(t, err)
		require.NotNil(t, room)
		require.Equal(t, id, room.ID)
		require.Equal(t, PrivateChat, room.Type)
		require.Zero(t, room.LastMessageSentAt)
		require.Equal(t, -1, room.LastMessageSent)
		require.Contains(t, room.Members, RoomMember{Username: user1.Username, RoomID: id, RoomName: user2.Username, LastMessageRead: -1})
		require.Contains(t, room.Members, RoomMember{Username: user2.Username, RoomID: id, RoomName: user1.Username, LastMessageRead: -1})
	})
	t.Run("create_private_chat_with_existing_users", func(t *testing.T) {
		id, err := chatStore.CreatePrivateChat(ctx, [2]string{user1.Username, user2.Username})
		require.Equal(t, ErrConflictedRoom, err)
		require.Equal(t, "", id)
	})
}

func Test_CreateGroupChat(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, u := range users {
		if err := userStore.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%+v): %v", u, err)
		}
	}
	roomName := "Group Chat"
	t.Run("create_group_chat_with_insufficient_members", func(t *testing.T) {
		id, err := chatStore.CreateGroupChat(ctx, roomName, users[0].Username)
		require.Equal(t, ErrInvalidUser, err)
		require.Equal(t, "", id)
	})
	t.Run("create_group_chat_with_invalid_user_id", func(t *testing.T) {
		id, err := chatStore.CreateGroupChat(ctx, roomName, users[0].Username, "invalid")
		require.Equal(t, ErrInvalidUser, err)
		require.Equal(t, "", id)
	})
	t.Run("create_group_chat_successfully", func(t *testing.T) {
		id, err := chatStore.CreateGroupChat(ctx, roomName, users[0].Username, users[1].Username, users[2].Username)
		require.Nil(t, err)
		require.NotEmpty(t, id)
		room, err := chatStore.GetRoomByID(ctx, id)
		require.Nil(t, err)
		require.NotNil(t, room)
		require.Equal(t, id, room.ID)
		require.Equal(t, GroupChat, room.Type)
		require.Len(t, room.Members, 3)
		require.Zero(t, room.LastMessageSentAt)
		require.Equal(t, -1, room.LastMessageSent)
		require.Contains(t, room.Members,
			RoomMember{Username: users[0].Username,
				RoomID:          id,
				RoomName:        roomName,
				LastMessageRead: -1},
		)
		require.Contains(t, room.Members,
			RoomMember{Username: users[1].Username,
				RoomID:          id,
				RoomName:        roomName,
				LastMessageRead: -1})
		require.Contains(t, room.Members, RoomMember{Username: users[2].Username, RoomID: id, RoomName: roomName, LastMessageRead: -1})
	})
	t.Run("create_group_chat_with_duplicate_name_and_members", func(t *testing.T) {
		id, err := chatStore.CreateGroupChat(ctx, roomName, users[0].Username, users[1].Username, users[2].Username)
		require.Nil(t, err)
		require.NotEmpty(t, id)
	})
	t.Run("create_group_chat_with_duplicate_members", func(t *testing.T) {
		id, err := chatStore.CreateGroupChat(ctx, roomName, users[0].Username, users[0].Username, users[1].Username)
		require.Nil(t, err)
		require.NotEmpty(t, id)
	})
	t.Run("create_group_chat_with_single_duplicate_member", func(t *testing.T) {
		id, err := chatStore.CreateGroupChat(ctx, roomName, users[1].Username, users[1].Username)
		require.Equal(t, ErrInvalidUser, err)
		require.Equal(t, "", id)
	})
}

func Test_IsRoomMember(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, u := range users {
		err := userStore.CreateUser(ctx, u)
		require.Nil(t, err, "CreateUser")

	}

	privateChatRoomId, err := chatStore.CreatePrivateChat(ctx, [2]string{users[0].Username, users[1].Username})
	require.Nil(t, err, "CreatePrivateChat")
	groupChatRoomID, err := chatStore.CreateGroupChat(ctx, "Group chat", users[0].Username, users[1].Username)
	require.Nil(t, err, "CreateGroupChat")

	t.Run("is_room_member_with_valid_room_member", func(t *testing.T) {
		ok, err := chatStore.IsRoomMember(ctx, privateChatRoomId, users[0].Username)
		require.Nil(t, err)
		require.True(t, ok)

		ok, err = chatStore.IsRoomMember(ctx, privateChatRoomId, users[1].Username)
		require.Nil(t, err)
		require.True(t, ok)

		ok, err = chatStore.IsRoomMember(ctx, groupChatRoomID, users[0].Username)
		require.Nil(t, err)
		require.True(t, ok)

		ok, err = chatStore.IsRoomMember(ctx, groupChatRoomID, users[1].Username)
		require.Nil(t, err)
		require.True(t, ok)
	})

	t.Run("is_room_member_with_invalid_room_member", func(t *testing.T) {
		ok, err := chatStore.IsRoomMember(ctx, privateChatRoomId, users[2].Username)
		require.Nil(t, err)
		require.False(t, ok)

		ok, err = chatStore.IsRoomMember(ctx, groupChatRoomID, users[2].Username)
		require.Nil(t, err)
		require.False(t, ok)
	})

}

func Test_SendMessageToRoom(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, u := range users {
		err := userStore.CreateUser(ctx, u)
		require.Nil(t, err, "CreateUser")

	}

	groupChatRoomID, err := chatStore.CreateGroupChat(ctx, "Group chat", users[0].Username, users[1].Username)
	require.Nil(t, err, "CreateGroupChat")

	t.Run("send_message_to_room_with_user_not_in_room", func(t *testing.T) {
		input := MessageCreateInput{
			Type:   TextMessage,
			Data:   "Hi there",
			Sender: users[2].Username,
			RoomID: groupChatRoomID,
		}
		require.Nil(t, input.Validate())
		message, err := chatStore.SendMessageToRoom(ctx, input)
		require.NotNil(t, err)
		require.Equal(t, ErrInvalidRoom, err)
		require.Nil(t, message)

		messages, err := chatStore.GetRoomMessages(ctx, groupChatRoomID, 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)
	})

	t.Run("send_message_to_room_with_invalid_message_type", func(t *testing.T) {
		input := MessageCreateInput{
			Type:   MessageType(100),
			Data:   "Hi there",
			Sender: users[0].Username,
			RoomID: groupChatRoomID,
		}
		require.Nil(t, input.Validate())
		message, err := chatStore.SendMessageToRoom(ctx, input)
		require.NotNil(t, err)
		require.Equal(t, ErrInvalidMessageType, err)
		require.Nil(t, message)

		messages, err := chatStore.GetRoomMessages(ctx, groupChatRoomID, 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)
	})

	t.Run("send_message_to_room_with_invalid_message", func(t *testing.T) {
		input := MessageCreateInput{}
		require.NotNil(t, input.Validate())
		message, err := chatStore.SendMessageToRoom(ctx, input)
		require.NotNil(t, err)
		require.Equal(t, ErrInvalidMessage, err)
		require.Nil(t, message)

		messages, err := chatStore.GetRoomMessages(ctx, groupChatRoomID, 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)

	})

	t.Run("send_message_to_room_successfully", func(t *testing.T) {
		input := MessageCreateInput{
			Type:   TextMessage,
			Data:   "Hi there",
			Sender: users[0].Username,
			RoomID: groupChatRoomID,
		}
		require.Nil(t, input.Validate())
		message, err := chatStore.SendMessageToRoom(ctx, input)
		require.Nil(t, err)
		require.NotNil(t, message)
		require.Equal(t, TextMessage, message.Type)
		require.Equal(t, input.Data, message.Data)
		require.Equal(t, input.Sender, message.Sender)
		require.Equal(t, input.RoomID, message.RoomID)
		require.NotZero(t, message.ID)
		require.NotZero(t, message.SentAt)

		messages, err := chatStore.GetRoomMessages(ctx, groupChatRoomID, 0, 1)
		require.Nil(t, err)
		require.Len(t, messages, 1)

		// the last message read for the sender should be the message sent
		room, err := chatStore.GetRoomByID(ctx, groupChatRoomID)
		require.Nil(t, err)
		require.NotNil(t, room)

		m := getRoomMemberByUsername(room, users[0].Username)
		require.Equal(t, message.ID, m.LastMessageRead)

		// the last message sent should be updated
		require.Equal(t, message.ID, room.LastMessageSent)
		require.Equal(t, message.SentAt, room.LastMessageSentAt)

	})
}

func Test_GetRoomMessages(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, u := range users {
		err := userStore.CreateUser(ctx, u)
		require.Nil(t, err, "CreateUser")

	}

	groupChatRoomID, err := chatStore.CreateGroupChat(ctx, "Group chat", users[0].Username, users[1].Username)

	// reading messages of room with no messages
	messages, err := chatStore.GetRoomMessages(ctx, groupChatRoomID, 0, 1)
	require.Nil(t, err)
	require.Nil(t, messages)

	require.Nil(t, err, "CreateGroupChat")
	inputs := []MessageCreateInput{
		{
			Type:   TextMessage,
			Data:   "Yooo",
			Sender: users[0].Username,
			RoomID: groupChatRoomID,
		},
		{
			Type:   TextMessage,
			Data:   "Hoo",
			Sender: users[1].Username,
			RoomID: groupChatRoomID,
		},
		{
			Type:   TextMessage,
			Data:   "Goooo",
			Sender: users[0].Username,
			RoomID: groupChatRoomID,
		},
		{
			Type:   TextMessage,
			Data:   "Fooo",
			Sender: users[1].Username,
			RoomID: groupChatRoomID,
		},
	}

	expMessages := make([]Message, len(inputs))

	for i, input := range inputs {
		message, err := chatStore.SendMessageToRoom(ctx, input)
		require.Nilf(t, err, "SendMessageToRoom(#message %d): %v", i, err)
		require.NotNil(t, message)
		// messages should be stored in the reverse order they're sent
		expMessages[len(expMessages)-1-i] = *message
	}

	// reading the first two messages
	messages, err = chatStore.GetRoomMessages(ctx, groupChatRoomID, 0, 2)
	require.Nil(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, expMessages[0], messages[0])
	assert.Equal(t, expMessages[1], messages[1])

	// reading the last two messages
	messages, err = chatStore.GetRoomMessages(ctx, groupChatRoomID, 2, 2)
	require.Nil(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, expMessages[2], messages[0])
	assert.Equal(t, expMessages[3], messages[1])
}

func Test_GetRoomSummaries(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, u := range users {
		err := userStore.CreateUser(ctx, u)
		require.Nil(t, err, "CreateUser")

	}

	// no rooms yet
	rooms, err := chatStore.GetRoomSummaries(ctx, users[0].Username, 0, 1)
	require.Nil(t, err)
	require.Nil(t, rooms)

	group1RoomID, err := chatStore.CreateGroupChat(ctx, "group1", users[0].Username, users[1].Username)
	require.Nil(t, err, "CreateGroupChat(1)")
	group2RoomID, err := chatStore.CreateGroupChat(ctx, "group1", users[0].Username, users[2].Username)
	require.Nil(t, err, "CreateGroupChat(2)")

	privateChat1RoomID, err := chatStore.CreatePrivateChat(ctx, [2]string{users[0].Username, users[1].Username})
	require.Nil(t, err, "CreatePrivateChat(1)")
	privateChat2RoomID, err := chatStore.CreatePrivateChat(ctx, [2]string{users[0].Username, users[2].Username})
	require.Nil(t, err, "CreatePrivateChat(2)")

	// since no message sent the rooms should be ordered by name asc
	// get first two
	rooms, err = chatStore.GetRoomSummaries(ctx, users[0].Username, 0, 2)
	require.Nil(t, err)
	require.Len(t, rooms, 2)
	require.Equal(t, group1RoomID, rooms[0].ID)
	require.Equal(t, group2RoomID, rooms[1].ID)

	// get the last two
	rooms, err = chatStore.GetRoomSummaries(ctx, users[0].Username, 2, 2)
	require.Nil(t, err)
	require.Len(t, rooms, 2)
	require.Equal(t, privateChat1RoomID, rooms[0].ID)
	require.Equal(t, privateChat2RoomID, rooms[1].ID)

}

func Test_ReadRoomMessages(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, u := range users {
		err := userStore.CreateUser(ctx, u)
		require.Nil(t, err, "CreateUser")

	}

	groupChatRoomID, err := chatStore.CreateGroupChat(ctx, "Group chat", users[0].Username, users[1].Username)
	require.Nil(t, err, "CreateGroupChat")

	inputs := []MessageCreateInput{
		{
			Type:   TextMessage,
			Data:   "Yooo",
			Sender: users[0].Username,
			RoomID: groupChatRoomID,
		},
		{
			Type:   TextMessage,
			Data:   "Hoo",
			Sender: users[1].Username,
			RoomID: groupChatRoomID,
		},
		{
			Type:   TextMessage,
			Data:   "Goooo",
			Sender: users[1].Username,
			RoomID: groupChatRoomID,
		},
		{
			Type:   TextMessage,
			Data:   "Fooo",
			Sender: users[1].Username,
			RoomID: groupChatRoomID,
		},
	}

	expMessages := make([]Message, len(inputs))

	for i, input := range inputs {
		message, err := chatStore.SendMessageToRoom(ctx, input)
		require.Nilf(t, err, "SendMessageToRoom(#message %d): %v", i, err)
		require.NotNil(t, message)
		expMessages[i] = *message
	}

	room, err := chatStore.GetRoomByID(ctx, groupChatRoomID)
	require.Nil(t, err)
	require.NotNil(t, room)
	m1 := getRoomMemberByUsername(room, users[0].Username)
	require.NotNil(t, m1)
	require.Equal(t, expMessages[0].ID, m1.LastMessageRead)

	m2 := getRoomMemberByUsername(room, users[1].Username)
	require.NotNil(t, m2)
	require.Equal(t, expMessages[3].ID, m2.LastMessageRead)

	u1LastReadMessage, u1ReadAt, err := chatStore.ReadRoomMessages(ctx, groupChatRoomID, users[0].Username)
	require.Nil(t, err)
	require.NotZero(t, u1ReadAt)
	require.Equal(t, expMessages[3].ID, u1LastReadMessage)

	u2LastReadMessage, u2ReadAt, err := chatStore.ReadRoomMessages(ctx, groupChatRoomID, users[1].Username)
	require.Nil(t, err)
	require.NotZero(t, u2ReadAt)
	require.Equal(t, expMessages[3].ID, u2LastReadMessage)
}

func Test_GetFriends(t *testing.T) {
	userStore, chatStore, tearDown := setUp(t)
	defer tearDown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, u := range users {
		err := userStore.CreateUser(ctx, u)
		require.Nil(t, err, "CreateUser")

	}

	// user[0] and user[1] have direct private chat
	// user[0], user[1], user[2] are in the same group chat
	// user[1] and user[3] have direct private chat

	_, err := chatStore.CreatePrivateChat(ctx, [2]string{users[0].Username, users[1].Username})
	require.Nil(t, err, "CreatePrivateChat(user[0], user[1]")

	_, err = chatStore.CreatePrivateChat(ctx, [2]string{users[1].Username, users[3].Username})
	require.Nil(t, err, "CreatePrivateChat(user[1], user[3]")

	_, err = chatStore.CreateGroupChat(ctx, "Group chat", users[0].Username, users[1].Username, users[2].Username)
	require.Nil(t, err, "CreateGroupChat(user[0], user[1], user[2])")

	// user[0] friends are user[1] and user[2]
	friends, err := chatStore.GetFriends(ctx, users[0].Username)
	require.Nil(t, err)
	require.Len(t, friends, 2)
	assert.Contains(t, friends, users[1].Username)
	assert.Contains(t, friends, users[2].Username)

	// user[1] friends are user[0], user[2] and user[3]
	friends, err = chatStore.GetFriends(ctx, users[1].Username)
	require.Nil(t, err)
	require.Len(t, friends, 3)
	require.Contains(t, friends, users[0].Username)
	require.Contains(t, friends, users[3].Username)
	require.Contains(t, friends, users[2].Username)

	// user[2] friends are user[0] and user[1]
	friends, err = chatStore.GetFriends(ctx, users[2].Username)
	require.Nil(t, err)
	require.Len(t, friends, 2)
	require.Contains(t, friends, users[0].Username)
	require.Contains(t, friends, users[1].Username)

	// user[3] friends are user[1]
	friends, err = chatStore.GetFriends(ctx, users[3].Username)
	require.Nil(t, err)
	require.Len(t, friends, 1)
	require.Contains(t, friends, users[1].Username)
}

func getRoomMemberByUsername(room *Room, username string) *RoomMember {
	var target RoomMember
	for _, member := range room.Members {
		if member.Username == username {
			target = member
			break
		}
	}
	return &target
}
