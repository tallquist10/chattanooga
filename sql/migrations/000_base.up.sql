CREATE schema chats;

CREATE ROLE masterlist SUPERUSER LOGIN PASSWORD 'password';

CREATE TABLE chats.users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(32) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false
);

CREATE TABLE chats.chat_rooms (
    id BIGINT PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false
);

CREATE TABLE chats.user_chat_room (
    user_id BIGINT references users(id),
    chat_room_id BIGINT references chat_rooms(id)
);

CREATE TABLE chats.chat_message (
    id BIGINT PRIMARY KEY,
    user_id BIGINT references users(id) NOT NULL,
    chat_room_id BIGINT references chat_rooms(id) NOT NULL,
    content TEXT NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false
)

