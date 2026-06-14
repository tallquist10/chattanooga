package api

import (
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
)

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
	Connection *ws.Conn
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
	Connection *ws.Conn
	UserIdChan chan int64
}

type CloseClientRequest struct {
	UserId      int64
	successChan bool
}

type DBHandler[T any] struct {
	Request *T
	Context *gin.Context
}

type ChatServer interface {
	RegisterUser(msg *User)
	RegisterClient(msg *ClientMessage)
	ReceiveMessage(msg *ChatRoomMessage)
	BroadcastMessage(msg *Message, chatRoom *ChatRoom)
}

type ChatServerConfig struct {
	ConcurrentProcessors int
	RateLimitPerUser     int
	BufferSize           int
	PingIntervalMs       int
}
