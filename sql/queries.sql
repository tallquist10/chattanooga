-- name: createUser :one
INSERT INTO users (
    username, display_name
) VALUES ($1, $2)
RETURNING *;
-- name: getUserById :one
SELECT * FROM users
WHERE id = $1
RETURNING *;
-- name: createChatRoom :one
INSERT INTO chat_rooms (
    name, description
) VALUES ($1, $2)
RETURNING *;
-- name: getChatRoom :one
SELECT * FROM chat_rooms
WHERE id = $1
RETURNING *;
-- name: getChatRooms :many
-- name: createMessage :one
-- name: getMessage :one
-- name: getMessagesForChatRoom :many
