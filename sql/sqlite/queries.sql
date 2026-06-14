-- name: CreateUser :one
INSERT INTO users (
    username, display_name
) VALUES (?, ?)
RETURNING *;
-- name: GetUserById :one
SELECT * FROM users
WHERE id = ?;
-- name: CreateChatRoom :one
INSERT INTO chat_rooms (
    name, description
) VALUES (?, ?)
RETURNING *;
-- name: GetChatRoom :one
SELECT * FROM chat_rooms
WHERE id = ?;
-- name: GetChatRooms :many
SELECT * FROM chat_rooms
LIMIT ?
OFFSET ?;
-- name: AddUserToChatRoom :exec
INSERT INTO user_chat_room (
    user_id, chat_room_id
) VALUES (?, ?);
-- name: CreateMessage :one
INSERT INTO chat_message (
    user_id, chat_room_id, content, is_edited, is_deleted
) VALUES (?, ?, ?, FALSE, FALSE)
RETURNING *;
-- name: EditMessage :exec
UPDATE chat_message
SET content = ?
WHERE id = ?;
-- name: DeleteMessage :exec
DELETE FROM chat_message
WHERE id = ?;
-- name: GetMessage :one
SELECT * FROM chat_message
WHERE id = ?;
-- name: GetMessagesForChatRoom :many
SELECT * FROM chat_message
WHERE chat_room_id = ?
LIMIT ?
OFFSET ?;