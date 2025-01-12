package core

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	owner   = User{Username: "owner", Password: "password", Name: "Owner"}
	member1 = User{Username: "member1", Password: "password", Name: "Member 1"}
	member2 = User{Username: "member2", Password: "password", Name: "Member 2"}
)

type ChatFixture struct {
	userStore UserStore
	chatStore ChatStore
	db        *sql.DB
	ctx       context.Context
	tearDown  func()
	t         *testing.T
}

func NewChatFixture(t *testing.T) *ChatFixture {
	ctx, cancel := context.WithCancel(context.Background())

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}

	migrationfs := os.DirFS("../migrations")
	goose.SetBaseFS(migrationfs)

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	if err := goose.Up(db, "."); err != nil {
		t.Fatal(err)
	}

	userStore := NewSqlieUserStore(db)

	f := &ChatFixture{
		userStore: userStore,
		chatStore: NewSQLiteChatStore(db, userStore),
		ctx:       ctx,
		db:        db,
		tearDown: func() {
			cancel()
			db.Close()
		},
		t: t,
	}

	return f
}

func TestCreateRoom(t *testing.T) {

	t.Run("create room successfully", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner)
		roomName := "Group chat"

		id, err := f.chatStore.CreateRoom(f.ctx, roomName, owner.Username)
		require.Nil(t, err)
		require.NotEmpty(t, id)
		room, err := f.chatStore.GetRoomByID(f.ctx, id)
		require.Nil(t, err)
		assert.Equal(t, id, room.ID)
		assert.Equal(t, roomName, room.Name)
		assert.Zero(t, room.LastMessageSentAt)
		assert.Equal(t, 0, room.LastMessageSent)
		assert.Len(t, room.Members, 1)
		assert.Equal(t, owner.Username, room.Members[0].Username)
		assert.Equal(t, Owner, room.Members[0].Role)
	})

	t.Run("create room with invalid owner", func(t *testing.T) {
		fixture := NewChatFixture(t)
		defer fixture.tearDown()

		id, err := fixture.chatStore.CreateRoom(fixture.ctx, "Group chat", "random")
		require.NotNil(t, err)
		require.Zero(t, id)
		assert.Equal(t, ErrInvalidUser, err)
	})

}

func TestGetRoomByID(t *testing.T) {
	t.Run("room exist", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner)
		r := seedRooms(f, owner)[0]

		room, err := f.chatStore.GetRoomByID(f.ctx, r.ID)

		require.Nil(t, err)
		require.NotNil(t, room)
		assert.Equal(t, r.ID, room.ID)
		assert.Equal(t, r.Name, room.Name)
		assert.Equal(t, r.Members, room.Members)
		assert.Equal(t, r.LastMessageSent, room.LastMessageSent)
		assert.Equal(t, r.LastMessageSentAt, room.LastMessageSentAt)
	})

	t.Run("room does not exist", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()

		room, err := f.chatStore.GetRoomByID(f.ctx, "random")

		require.Nil(t, err)
		assert.Nil(t, room)
	})
}

func TestGetRoomMembers(t *testing.T) {
	t.Run("room exist", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		room := seedRooms(f, owner)[0]
		err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)
		require.Nil(t, err, "AddRoomMember")

		members, err := f.chatStore.GetRoomMembers(f.ctx, room.ID)

		require.Nil(t, err)
		require.Len(t, members, 2)
		assert.Contains(t, members, RoomMember{
			Role:     Owner,
			Username: owner.Username,
			RoomID:   room.ID,
		})
		assert.Contains(t, members, RoomMember{
			Role:     Member,
			Username: member1.Username,
			RoomID:   room.ID,
		})
	})

	t.Run("room does not exist", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()

		members, err := f.chatStore.GetRoomMembers(f.ctx, "random")

		require.Nil(t, err)
		assert.Len(t, members, 0)
		assert.Nil(t, members)
	})
}

func TestAddRoomMember(t *testing.T) {

	t.Run("add new valid member", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		room := seedRooms(f, owner)[0]

		err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)

		assert.Nil(t, err)
		members, err := f.chatStore.GetRoomMembers(f.ctx, room.ID)
		require.Nil(t, err, "GetRoomMembers")
		assert.Len(t, members, 2)
		assert.Contains(t, members, RoomMember{
			Role:            Member,
			Username:        member1.Username,
			RoomID:          room.ID,
			LastMessageRead: 0,
		})
		assert.Contains(t, members, RoomMember{
			Role:            Owner,
			Username:        owner.Username,
			RoomID:          room.ID,
			LastMessageRead: 0,
		})
	})

	t.Run("add new invalid user", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		room := seedRooms(f, owner)[0]

		err := f.chatStore.AddRoomMember(f.ctx, room.ID, "random", Member)

		assert.NotNil(t, err)
		assert.Equal(t, ErrInvalidUser, err)
		members, err := f.chatStore.GetRoomMembers(f.ctx, room.ID)
		require.Nil(t, err, "GetRoomMembers")
		assert.Len(t, members, 1)
		assert.Contains(t, members, RoomMember{
			Role:            Owner,
			Username:        owner.Username,
			RoomID:          room.ID,
			LastMessageRead: 0,
		})
	})

	t.Run("add existing member", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		room := seedRooms(f, owner)[0]
		err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)

		err = f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)
		require.Nil(t, err)
		err = f.chatStore.AddRoomMember(f.ctx, room.ID, owner.Username, Member)
		require.Nil(t, err)
		members, err := f.chatStore.GetRoomMembers(f.ctx, room.ID)
		require.Nil(t, err, "GetRoomMembers")
		assert.Len(t, members, 2)
		assert.Contains(t, members, RoomMember{
			Role:            Owner,
			Username:        owner.Username,
			RoomID:          room.ID,
			LastMessageRead: 0,
		})
		assert.Contains(t, members, RoomMember{
			Role:            Member,
			Username:        member1.Username,
			RoomID:          room.ID,
			LastMessageRead: 0,
		})
	})

	t.Run("add member with owner role", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		room := seedRooms(f, owner)[0]

		err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Owner)
		require.NotNil(t, err)
		assert.Equal(t, ErrDisAllowedOperation, err)
	})

	t.Run("add member to non-existent room", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, member1)

		err := f.chatStore.AddRoomMember(f.ctx, "random", member1.Username, Member)
		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidRoom, err)
	})
}

func TestRemoveRoomMember(t *testing.T) {
	t.Run("remove valid member", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		room := seedRooms(f, owner)[0]
		err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)
		require.Nil(t, err, "AddRoomMember")

		err = f.chatStore.RemoveRoomMember(f.ctx, room.ID, member1.Username)

		assert.Nil(t, err)
		members, err := f.chatStore.GetRoomMembers(f.ctx, room.ID)
		require.Nil(t, err)
		assert.Len(t, members, 1)
		assert.NotContains(t, members, RoomMember{
			Role:            Member,
			Username:        member1.Username,
			RoomID:          room.ID,
			LastMessageRead: 0,
		})
	})

	t.Run("remove invalid member", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner)
		room := seedRooms(f, owner)[0]

		err := f.chatStore.RemoveRoomMember(f.ctx, room.ID, "random")

		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidMember, err)
		members, err := f.chatStore.GetRoomMembers(f.ctx, room.ID)
		require.Nil(t, err)
		assert.Len(t, members, 1)
	})

	t.Run("remove owner", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner)
		room := seedRooms(f, owner)[0]

		err := f.chatStore.RemoveRoomMember(f.ctx, room.ID, owner.Username)

		assert.NotNil(t, err)
		assert.Equal(t, ErrDisAllowedOperation, err)
		members, err := f.chatStore.GetRoomMembers(f.ctx, room.ID)
		require.Nil(t, err)
		assert.Len(t, members, 1)
		assert.Contains(t, members, RoomMember{
			Role:            Owner,
			Username:        owner.Username,
			RoomID:          room.ID,
			LastMessageRead: 0,
		})
	})
}

func Test_IsRoomMember(t *testing.T) {
	f := NewChatFixture(t)
	defer f.tearDown()
	seedUsers(f.ctx, f.t, f.userStore, owner, member1, member2)
	room := seedRooms(f, owner)[0]

	err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)
	require.Nil(t, err, "AddRoomMember")

	t.Run("user is a member", func(t *testing.T) {
		ok, role, err := f.chatStore.IsRoomMember(f.ctx, room.ID, member1.Username)
		assert.Nil(t, err)
		assert.True(t, ok)
		assert.Equal(t, Member, role)

		ok, role, err = f.chatStore.IsRoomMember(f.ctx, room.ID, owner.Username)
		assert.Nil(t, err)
		assert.True(t, ok)
		assert.Equal(t, Owner, role)
	})

	t.Run("user is not a room member", func(t *testing.T) {
		ok, role, err := f.chatStore.IsRoomMember(f.ctx, room.ID, member2.Username)
		require.Nil(t, err)
		require.False(t, ok)
		require.Zero(t, role)
	})
}

func sortMembers(rooms []Room) {
	for i := range rooms {
		r := &rooms[i]
		slices.SortFunc(r.Members, func(i, j RoomMember) int {
			return strings.Compare(i.Username, j.Username)
		})
	}
}

func TestGetUserRooms(t *testing.T) {

	t.Run("filter logic", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		rooms := seedRooms(f, owner, "Room1", "Room2")

		err := f.chatStore.AddRoomMember(f.ctx, rooms[0].ID,
			member1.Username, Member)
		require.Nil(t, err)
		rooms[0].Members = append(rooms[0].Members, RoomMember{
			Username: member1.Username,
			RoomID:   rooms[0].ID,
			Role:     Member,
		})
		sortMembers(rooms)

		ownerRooms, err := f.chatStore.GetUserRooms(f.ctx, owner.Username, 0, len(rooms))
		require.Nil(t, err)
		require.Len(t, ownerRooms, 2)
		sortMembers(ownerRooms)
		for _, room := range rooms {
			assert.Contains(t, ownerRooms, room)
		}

		member1Rooms, err := f.chatStore.GetUserRooms(f.ctx, member1.Username,
			0, len(rooms))
		require.Nil(t, err)
		require.Len(t, member1Rooms, 1)
		sortMembers(member1Rooms)
		assert.Contains(t, member1Rooms, rooms[0])
	})

	t.Run("pagination logic", func(t *testing.T) {
		f := NewChatFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, owner, member1)
		rooms := seedRooms(f, owner, "Room1", "Room2")

		page1, err := f.chatStore.GetUserRooms(f.ctx, owner.Username, 0, 1)
		require.Nil(t, err)
		require.Len(t, page1, 1)
		require.Contains(t, page1, rooms[0])

		page2, err := f.chatStore.GetUserRooms(f.ctx, owner.Username, 1, 1)
		require.Nil(t, err)
		require.Len(t, page2, 1)
		require.Contains(t, page2, rooms[1])
	})
}

func TestSendMessageToRoom(t *testing.T) {
	f := NewChatFixture(t)
	defer f.tearDown()
	seedUsers(f.ctx, f.t, f.userStore, owner, member1)
	room := seedRooms(f, owner)[0]

	t.Run("non member send message to room", func(t *testing.T) {
		input := MessageCreateInput{
			Type:   TextMessage,
			Data:   "Hi there",
			Sender: member1.Username,
			RoomID: room.ID,
		}
		require.Nil(t, input.Validate())
		message, err := f.chatStore.SendMessageToRoom(f.ctx, input)
		require.NotNil(t, err)
		require.Equal(t, ErrInvalidRoom, err)
		require.Nil(t, message)

		messages, err := f.chatStore.GetRoomMessages(f.ctx, room.ID, 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)
	})

	t.Run("send message with invalid message type", func(t *testing.T) {
		input := MessageCreateInput{
			Type:   MessageType(100),
			Data:   "Hi there",
			Sender: owner.Username,
			RoomID: room.ID,
		}
		require.Nil(t, input.Validate())
		message, err := f.chatStore.SendMessageToRoom(f.ctx, input)
		require.NotNil(t, err)
		require.Equal(t, ErrInvalidMessageType, err)
		require.Nil(t, message)

		messages, err := f.chatStore.GetRoomMessages(f.ctx, room.ID, 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)
	})

	t.Run("send invalid message", func(t *testing.T) {
		input := MessageCreateInput{}
		require.NotNil(t, input.Validate())
		message, err := f.chatStore.SendMessageToRoom(f.ctx, input)
		require.NotNil(t, err)
		require.Equal(t, ErrInvalidMessage, err)
		require.Nil(t, message)

		messages, err := f.chatStore.GetRoomMessages(f.ctx, room.ID, 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)
	})

	t.Run("send valid message to room", func(t *testing.T) {
		input := MessageCreateInput{
			Type:   TextMessage,
			Data:   "Hi there",
			Sender: owner.Username,
			RoomID: room.ID,
		}
		require.Nil(t, input.Validate())
		message, err := f.chatStore.SendMessageToRoom(f.ctx, input)
		require.Nil(t, err)
		require.NotNil(t, message)
		require.Equal(t, TextMessage, message.Type)
		require.Equal(t, input.Data, message.Data)
		require.Equal(t, input.Sender, message.Sender)
		require.Equal(t, input.RoomID, message.RoomID)
		require.NotZero(t, message.ID)
		require.NotZero(t, message.SentAt)

		messages, err := f.chatStore.GetRoomMessages(f.ctx, room.ID, 0, 1)
		require.Nil(t, err)
		require.Len(t, messages, 1)

		// the last message read for the sender should be the message sent
		room, err := f.chatStore.GetRoomByID(f.ctx, room.ID)
		require.Nil(t, err)
		require.NotNil(t, room)
		require.Equal(t, message.SentAt, room.LastMessageSentAt)
		require.Equal(t, message.ID, room.LastMessageSent)
		m := getRoomMemberByUsername(*room, owner.Username)
		require.Equal(t, message.ID, m.LastMessageRead)
	})
}

func TestGetRoomMessages(t *testing.T) {
	f := NewChatFixture(t)
	defer f.tearDown()
	seedUsers(f.ctx, f.t, f.userStore, owner, member1)
	room := seedRooms(f, owner)[0]
	err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)
	require.Nil(t, err)

	t.Run("get messages from a non-existent room", func(t *testing.T) {
		messages, err := f.chatStore.GetRoomMessages(f.ctx, "random", 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)
		require.Len(t, messages, 0)
	})

	t.Run("get messages from an empty room", func(t *testing.T) {
		messages, err := f.chatStore.GetRoomMessages(f.ctx, room.ID, 0, 1)
		require.Nil(t, err)
		require.Nil(t, messages)
	})

	t.Run("get messages from a room", func(t *testing.T) {
		inputs := []MessageCreateInput{
			{
				Type:   TextMessage,
				Data:   "Yooo",
				Sender: owner.Username,
				RoomID: room.ID,
			},
			{
				Type:   TextMessage,
				Data:   "Hoo",
				Sender: member1.Username,
				RoomID: room.ID,
			},
			{
				Type:   TextMessage,
				Data:   "Goooo",
				Sender: owner.Username,
				RoomID: room.ID,
			},
			{
				Type:   TextMessage,
				Data:   "Fooo",
				Sender: member1.Username,
				RoomID: room.ID,
			},
		}

		expected := make([]Message, len(inputs))

		for i, input := range inputs {
			message, err := f.chatStore.SendMessageToRoom(f.ctx, input)
			require.Nil(t, err)
			require.NotNil(t, message)
			// messages should be stored in the reverse order they're sent
			expected[len(expected)-1-i] = *message
		}

		// reading the first two messages
		page1, err := f.chatStore.GetRoomMessages(f.ctx, room.ID, 0, 2)
		require.Nil(t, err)
		require.Len(t, page1, 2)
		assert.Equal(t, expected[0], page1[0])
		assert.Equal(t, expected[1], page1[1])

		// reading the last two messages
		page2, err := f.chatStore.GetRoomMessages(f.ctx, room.ID, 2, 2)
		require.Nil(t, err)
		require.Len(t, page2, 2)
		assert.Equal(t, expected[2], page2[0])
		assert.Equal(t, expected[3], page2[1])
	})

}

func TestReadRoomMessages(t *testing.T) {
	f := NewChatFixture(t)
	defer f.tearDown()

	seedUsers(f.ctx, f.t, f.userStore, owner, member1)
	rooms := seedRooms(f, owner)
	room := rooms[0]
	err := f.chatStore.AddRoomMember(f.ctx, room.ID, member1.Username, Member)
	require.Nil(t, err)

	inputs := []MessageCreateInput{
		{
			Type:   TextMessage,
			Data:   "Yooo",
			Sender: owner.Username,
			RoomID: room.ID,
		},
		{
			Type:   TextMessage,
			Data:   "Hoo",
			Sender: member1.Username,
			RoomID: room.ID,
		},
		{
			Type:   TextMessage,
			Data:   "Goooo",
			Sender: owner.Username,
			RoomID: room.ID,
		},
		{
			Type:   TextMessage,
			Data:   "Fooo",
			Sender: member1.Username,
			RoomID: room.ID,
		},
	}

	expMessages := make([]Message, len(inputs))

	for i, input := range inputs {
		message, err := f.chatStore.SendMessageToRoom(f.ctx, input)
		require.Nil(t, err)
		require.NotNil(t, message)
		expMessages[i] = *message
	}

	r, err := f.chatStore.GetRoomByID(f.ctx, room.ID)
	require.Nil(t, err)
	require.NotNil(t, r)
	_owner := getRoomMemberByUsername(*r, owner.Username)
	require.NotNil(t, _owner)
	require.Equal(t, expMessages[2].ID, _owner.LastMessageRead)

	_member1 := getRoomMemberByUsername(*r, member1.Username)
	require.NotNil(t, _member1)
	require.Equal(t, expMessages[3].ID, _member1.LastMessageRead)

	ownerLastReadMessage, ownerReadAt, err := f.chatStore.ReadRoomMessages(f.ctx, room.ID, owner.Username)
	require.Nil(t, err)
	require.NotZero(t, ownerReadAt)
	require.Equal(t, expMessages[3].ID, ownerLastReadMessage)

	member1ReadMessage, member1ReadAt, err := f.chatStore.ReadRoomMessages(f.ctx, room.ID, member1.Username)
	require.Nil(t, err)
	require.NotZero(t, member1ReadAt)
	require.Equal(t, expMessages[3].ID, member1ReadMessage)
}

func getRoomMemberByUsername(room Room, username string) RoomMember {
	var target RoomMember
	for _, member := range room.Members {
		if member.Username == username {
			target = member
			break
		}
	}
	return target
}
