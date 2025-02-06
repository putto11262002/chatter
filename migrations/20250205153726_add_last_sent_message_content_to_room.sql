-- +goose Up
ALTER TABLE rooms ADD COLUMN last_message_sent_data TEXT DEFAULT '';



-- +goose Down
ALTER TABLE rooms DROP COLUMN last_message_sent_data;
