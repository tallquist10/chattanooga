-- name: CreateUser :one
INSERT INTO chats.users (
    username, display_name
) VALUES ($1, $2)
RETURNING *;
-- name: GetUserById :one
SELECT * FROM chats.users
WHERE id = $1;
-- name: CreateChatRoom :one
INSERT INTO chats.chat_rooms (
    name, description
) VALUES ($1, $2)
RETURNING *;
-- name: GetChatRoom :one
SELECT * FROM chats.chat_rooms
WHERE id = $1;
-- name: GetChatRooms :many
SELECT * FROM chats.chat_rooms
LIMIT $1
OFFSET $2;
-- name: AddUserToChatRoom :exec
INSERT INTO chats.user_chat_room (
    user_id, chat_room_id
) VALUES ($1, $2);
-- name: CreateMessage :one
INSERT INTO chats.chat_message (
    user_id, chat_room_id, content, is_edited, is_deleted
) VALUES ($1, $2, $3, FALSE, FALSE)
RETURNING *;
-- name: EditMessage :exec
UPDATE chats.chat_message
SET content = $2
WHERE id = $1;
-- name: DeleteMessage :exec
DELETE FROM chats.chat_message
WHERE id = $1;
-- name: GetMessage :one
SELECT * FROM chats.chat_message
WHERE id = $1;
-- name: GetMessagesForChatRoom :many
SELECT * FROM chats.chat_message
WHERE chat_room_id = $1
LIMIT $2
OFFSET $3;