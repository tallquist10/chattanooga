-- name: CreateUser :one
INSERT INTO users (
    username, display_name
) VALUES ($1, $2)
RETURNING *;
-- name: GetUserById :one
SELECT * FROM users
WHERE id = $1;
-- name: CreateChatRoom :one
INSERT INTO chat_rooms (
    name, description
) VALUES ($1, $2)
RETURNING *;
-- name: GetChatRoom :one
SELECT * FROM chat_rooms
WHERE id = $1;
-- name: GetChatRooms :many
SELECT * FROM chat_rooms
LIMIT $1
OFFSET $2;
-- name: AddUserToChatRoom :exec
INSERT INTO user_chat_room (
    user_id, chat_room_id
) VALUES ($1, $2);
-- name: CreateMessage :one
INSERT INTO chat_message (
    user_id, chat_room_id, content, is_edited, is_deleted
) VALUES ($1, $2, $3, FALSE, FALSE)
RETURNING *;
-- name: EditMessage :exec
UPDATE chat_message
SET content = $2
WHERE id = $1;
-- name: DeleteMessage :exec
DELETE FROM chat_message
WHERE id = $1;
-- name: GetMessage :one
SELECT * FROM chat_message
WHERE id = $1;
-- name: GetMessagesForChatRoom :many
SELECT * FROM chat_message
WHERE chat_room_id = $1
LIMIT $2
OFFSET $3;