CREATE schema chats;

CREATE ROLE chatserver SUPERUSER LOGIN PASSWORD 'password';

CREATE TABLE chats.users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(32) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false
);
CREATE UNIQUE INDEX users_username_unique on chats.users(username);

CREATE TABLE chats.chat_rooms (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false
);
CREATE UNIQUE INDEX chat_rooms_name_unique on chats.chat_rooms(name);

CREATE TABLE chats.user_chat_room (
    user_id BIGINT references chats.users(id),
    chat_room_id BIGINT references chats.chat_rooms(id)
);
CREATE UNIQUE INDEX user_chat_rooms_unique on chats.user_chat_room(user_id, chat_room_id);

CREATE TABLE chats.chat_message (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT references chats.users(id) NOT NULL,
    chat_room_id BIGINT references chats.chat_rooms(id) NOT NULL,
    content TEXT NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false
)

