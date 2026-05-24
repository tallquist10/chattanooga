package internal

import "github.com/gorilla/websocket"

type User struct {
	Id          int64
	Username    string
	DisplayName string
}

type ChatRoom struct {
	Id       int64
	Name     string
	Messages []*Message
}

type Message struct {
	Sender  *User
	Content string
}

type ClientMessage struct {
	UserId     int64
	Connection *websocket.Conn
}

type ChatRoomMessage struct {
	UserId    int64
	ChannelId int64
	Content   string
	// eventual attachments?
}

type BroadcastMessage struct {
	message  *Message
	ChatRoom *ChatRoom
}

type WebSocketMessage[T any] struct {
	MessageType string
	Payload     T
}
