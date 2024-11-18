-- +goose Up
CREATE TABLE users (
  username TEXT PRIMARY KEY,
  password TEXT NOT NULL,
  name TEXT
);

CREATE TABLE blacklists  (
    token TEXT PRIMARY KEY
);

CREATE TABLE rooms (
    id TEXT PRIMARY KEY,
    type INTEGER NOT NULL
);

CREATE TABLE room_users (
	room_id TEXT NOT NULL,
	username TEXT NOT NULL,
	room_name TEXT NOT NULL,
	PRIMARY KEY (room_id, username),
	FOREIGN KEY (room_id) REFERENCES rooms(id),
	FOREIGN KEY (username) REFERENCES users(username)
);

CREATE TABLE messages (
	id TEXT PRIMARY KEY,
	type INTEGER NOT NULL,
	room_id TEXT NOT NULL,
	sender TEXT NOT NULL,
	data TEXT NOT NULL,
	sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (room_id) REFERENCES rooms(id),
	FOREIGN KEY (sender) REFERENCES users(username)
);

-- +goose Down
DROP TABLE room_users;
DROP TABLE rooms;
DROP TABLE users
DROP TABLE blacklists;
DROP TABLE messages;
