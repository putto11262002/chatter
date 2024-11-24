package chat

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/putto11262002/chatter/pkg/user"
	"github.com/stretchr/testify/assert"
)

func setUp(t *testing.T) (context.Context, *sql.DB, *user.SQLiteUserStore, *SQLiteChatStore, func()) {

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

	ctx, cancel := context.WithCancel(context.Background())

	userStore := user.NewSQLiteUserStore(db)
	chatStore := NewSQLiteChatStore(db, userStore)

	return ctx, db, userStore, chatStore, func() {
		cancel()
		db.Close()
	}
}

func Test_PrivateChatFlow(t *testing.T) {
	ctx, _, us, cs, tearDown := setUp(t)
	defer tearDown()

	u1 := user.User{
		Username: "user1",
		Password: "password",
		Name:     "User 1",
	}

	u2 := user.User{
		Username: "user2",
		Password: "password",
		Name:     "User 2",
	}

	if err := us.CreateUser(ctx, u1); err != nil {
		t.Fatalf("CreateUser(%+v): %v", u1, err)
	}

	if err := us.CreateUser(ctx, u2); err != nil {
		t.Fatalf("CreateUser(%+v): %v", u2, err)
	}

	// craete private chat with same users
	_, err := cs.CreatePrivateChat(ctx, [2]string{u1.Username, u1.Username})
	assert.Equal(t, ErrInvalidUser, err)

	// create private chat with user that does not exist
	_, err = cs.CreatePrivateChat(ctx, [2]string{u1.Username, "user3"})
	assert.Equal(t, ErrInvalidUser, err)

	roomID, err := cs.CreatePrivateChat(ctx, [2]string{u1.Username, u2.Username})
	if err != nil {
		t.Fatalf("CreatePrivateChat: %v", err)
	}

	assert.NotNil(t, roomID)

	room, err := cs.GetRoomRoomByID(ctx, roomID)
	if err != nil {
		t.Fatalf("GetRoomRoomByID: %v", err)
	}

	assert.NotNil(t, room)
	assert.Equal(t, PrivateChat, room.Type)
	assert.Contains(t, room.Users, RoomUser{
		Username: u1.Username,
		RoomName: u2.Username,
		RoomID:   roomID,
	})

	assert.Contains(t, room.Users, RoomUser{
		Username: u2.Username,
		RoomName: u1.Username,
		RoomID:   roomID,
	})

	// message := MessageCreateInput{
	// 	Data:   "Hello",
	// 	RoomID: roomID,
	// 	Type:   TextMessage,
	// 	Sender: u1.Username,
	// }

}

func Test_GroupChatFlow(t *testing.T) {
	ctx, _, us, s, tearDown := setUp(t)
	defer tearDown()

	u1 := user.User{
		Username: "user1",
		Password: "password",
		Name:     "User 1",
	}

	u2 := user.User{
		Username: "user2",
		Password: "password",
		Name:     "User 2",
	}

	if err := us.CreateUser(ctx, u1); err != nil {
		t.Fatalf("CreateUser(%+v): %v", u1, err)
	}

	if err := us.CreateUser(ctx, u2); err != nil {
		t.Fatalf("CreateUser(%+v): %v", u2, err)
	}

	roomName := "room1"

	// Create group chat with invalid users

	_, err := s.CreateGroupChat(ctx, roomName, u1.Username, u1.Username)
	assert.Equal(t, ErrInvalidUser, err)

	// Create group chat with one user
	_, err = s.CreateGroupChat(ctx, roomName, u1.Username)
	assert.Equal(t, ErrInvalidUser, err)

	// Create group chat successfully
	roomID, err := s.CreateGroupChat(ctx, roomName, u1.Username, u2.Username)
	if err != nil {
		t.Fatalf("CreateGroupChat: %v", err)
	}

	// roomID should not be null
	assert.NotNil(t, roomID)

	room, err := s.GetRoomRoomByID(ctx, roomID)
	if err != nil {
		t.Fatalf("GetRoomRoomByID: %v", err)
	}

	assert.NotNil(t, room)
	assert.Equal(t, GroupChat, room.Type)
	assert.Contains(t, room.Users,
		RoomUser{Username: u1.Username,
			RoomName: roomName,
			RoomID:   roomID})

	assert.Contains(t, room.Users,
		RoomUser{Username: u2.Username,
			RoomName: roomName,
			RoomID:   roomID})

	message, err := s.SendMessageToRoom(ctx, MessageCreateInput{
		Type:   TextMessage,
		Data:   "Hello",
		RoomID: roomID,
		Sender: u1.Username,
	})

	if err != nil {
		t.Fatalf("SendMessageToRoom: %v", err)
	}

	assert.NotNil(t, message)

	assert.Equal(t, TextMessage, message.Type)
	assert.Equal(t, "Hello", message.Data)
	assert.Equal(t, roomID, message.RoomID)
	assert.Equal(t, u1.Username, message.Sender)
	assert.NotNil(t, message.ID)
	assert.True(t, time.Now().After(message.SentAt),
		"SentAt should be in the past")
	assert.True(t, time.Now().Add(-time.Minute).Before(message.SentAt),
		"SentAt should be in the last minute")
	assert.Len(t, message.Interactions, 0)

	messages, err := s.GetRoomMessages(ctx, roomID, u1.Username)
	if err != nil {
		t.Fatalf("GetRoomMessages(before read): %v", err)
	}

	assert.Len(t, messages, 1, "should have 1 message")
	assert.Len(t, messages[0].Interactions, 0,
		"new message should have no interactions")

	if _, err := s.ReadRoomMessages(ctx, roomID, u2.Username); err != nil {
		t.Fatalf("ReadRoomMessages: %v", err)
	}

	messages, err = s.GetRoomMessages(ctx, roomID, u1.Username)
	if err != nil {
		t.Fatalf("GetRoomMessages(after read): %v", err)
	}

	assert.Len(t, messages, 1)
	fmt.Printf("messages: %v\n", messages)
	assert.Len(t, messages[0].Interactions, 1)

}
