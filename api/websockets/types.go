package websockets

import (
	"encoding/json"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/tallquist10/chat-server/api"
	db "github.com/tallquist10/chat-server/db/sqlc/sqlite"
)

const WebSocketClose = -1

type WebSocketChatServer struct {
	api.ChatServer
	upgrader           *ws.Upgrader
	msgChan            chan *WebSocketMessage
	closeChan          chan bool
	registerClientChan chan *api.RegisterClientRequest
	closeClientChan    chan *api.CloseClientRequest
	clientConnections  map[int64]*WebSocketClientMessageQueue
	clientUsers        map[int64]*api.User
	incomingChan       chan *api.DBHandler[api.ChatRoomMessage]
	broadcastChan      chan *api.BroadcastMessage
	pingsChan          chan int64
	dbConnection       *db.Queries
	config             *api.ChatServerConfig
	pingInterval       *time.Ticker
}

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

type WebSocketClientMessageQueue struct {
	conn         *ws.Conn
	msgChan      chan *WebSocketClientMessage
	clientClosed bool
}

type WebSocketClientMessage struct {
	messageType    int
	messageContent []byte
}
