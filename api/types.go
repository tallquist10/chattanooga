package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketMessageType string

const (
	RegisterUser        WebSocketMessageType = "register_user"
	ChatMessage         WebSocketMessageType = "chat_message"
	SubscribeToChatRoom WebSocketMessageType = "subscribe_to_chat_room"
)

type WebSocketMessage struct {
	Type    WebSocketMessageType `json:"type"`
	Payload json.RawMessage      `json:"payload"`
}

type CreateUserRequest struct {
	User       *User
	ResultChan chan int64
	ErrorChan  chan error
}

type CreateChatRoomRequest struct {
	Name        string
	Description string
	ResultChan  chan *ChatRoom
	ErrorChan   chan error
}

type User struct {
	Id          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
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
	UserId    int64  `json:"userId"`
	ChannelId int64  `json:"channelId"`
	Content   string `json:"content"`
	// eventual attachments?
}

type BroadcastMessage struct {
	Message  *Message
	ChatRoom *ChatRoom
}

type RegisterClientRequest struct {
	Connection *websocket.Conn
	UserIdChan chan int64
}

type DBHandler[T any] struct {
	Request *T
	Context *gin.Context
}
