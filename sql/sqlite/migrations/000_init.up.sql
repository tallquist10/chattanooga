CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(32) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false
);
CREATE UNIQUE INDEX users_username_unique on users(username);

CREATE TABLE chat_rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT false
);
CREATE UNIQUE INDEX chat_rooms_name_unique on chat_rooms(name);

CREATE TABLE user_chat_room (
    user_id BIGINT references users(id),
    chat_room_id BIGINT references chat_rooms(id)
);
CREATE UNIQUE INDEX user_chat_rooms_unique on user_chat_room(user_id, chat_room_id);

CREATE TABLE chat_message (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id BIGINT references users(id) NOT NULL,
    chat_room_id BIGINT references chat_rooms(id) NOT NULL,
    content TEXT NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false
)

