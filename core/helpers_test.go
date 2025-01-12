package core

import (
	"context"
	"testing"
)

func seedUsers(ctx context.Context, t *testing.T, userStore UserStore, users ...User) {
	for _, u := range users {
		err := userStore.CreateUser(ctx, u)
		if err != nil {
			t.Fatal(err)
		}

	}
}

func seedRooms(f *ChatFixture, owner User, names ...string) []Room {

	if len(names) == 0 {
		names = append(names, "Group chat")
	}

	rooms := make([]Room, 0, len(names))
	for _, name := range names {
		roomID, err := f.chatStore.CreateRoom(f.ctx, name, owner.Username)
		if err != nil {
			f.t.Fatal(err)
		}

		newRoom := Room{
			ID:   roomID,
			Name: name,
			Members: []RoomMember{
				{
					Username: owner.Username,
					Role:     Owner,
					RoomID:   roomID,
				},
			},
		}

		rooms = append(rooms, newRoom)
	}
	return rooms
}
