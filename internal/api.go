package internal

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type ChatServer interface {
	RegisterClient(msg *ClientMessage)
	ReceiveMessage(msg *ChatRoomMessage)
	BroadcastMessage(msg *Message, chatRoom *ChatRoom)
}

type ChatServerConfig struct {
	ConcurrentProcessors int
	RateLimitPerUser     int
	BufferSize           int
}

type WebSocketChatServer struct {
	ChatServer
	closeChan          chan bool
	createUserChan     chan *CreateUserRequest
	registerClientChan chan *RegisterClientRequest
	clientConnections  map[int64]*websocket.Conn
	incomingChan       chan *ChatRoomMessage
	broadcastChan      chan *BroadcastMessage
	config             *ChatServerConfig
}

type CreateUserRequest struct {
	User       *User
	ResultChan chan int64
	ErrorChan  chan error
}

type RegisterClientRequest struct {
	Connection *websocket.Conn
	UserIdChan chan int64
}

func NewChatServer(
	config *ChatServerConfig,
) *WebSocketChatServer {
	return &WebSocketChatServer{
		createUserChan:     make(chan *CreateUserRequest, config.BufferSize),
		registerClientChan: make(chan *RegisterClientRequest, config.BufferSize),
		clientConnections:  make(map[int64]*websocket.Conn),
		incomingChan:       make(chan *ChatRoomMessage, config.BufferSize),
		broadcastChan:      make(chan *BroadcastMessage, config.BufferSize),
		closeChan:          make(chan bool),
	}
}

func (cs *WebSocketChatServer) Start() error {
	defer close(cs.registerClientChan)
	defer close(cs.incomingChan)
	defer close(cs.broadcastChan)
	defer close(cs.createUserChan)
	defer close(cs.closeChan)
	var clientCounter int64 = 1
	for {
		select {
		case incomingMsg := <-cs.incomingChan:
			fmt.Println("Incoming message", incomingMsg)
			go cs.handleReceiveMessage(incomingMsg)
		case broadcastMsg := <-cs.broadcastChan:
			fmt.Println("Broadcast message", broadcastMsg)
			go cs.handleBroadcastMessage(broadcastMsg)
		case msg := <-cs.registerClientChan:
			clientId := clientCounter
			clientCounter++
			go cs.handleRegisterClient(msg, clientId)
		case _ = <-cs.closeChan:
			return nil
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (cs *WebSocketChatServer) ReceiveMessage(msg *ChatRoomMessage) {
	cs.incomingChan <- msg
}

func (cs *WebSocketChatServer) handleReceiveMessage(msg *ChatRoomMessage) {
	outgoingMsg := &BroadcastMessage{
		message: &Message{
			Sender: &User{
				Id: msg.UserId,
			},
			Content: string(msg.Content),
		},
	}
	cs.BroadcastMessage(outgoingMsg)
}

func (cs *WebSocketChatServer) handleBroadcastMessage(msg *BroadcastMessage) {
	for clientId, conn := range cs.clientConnections {
		if msg.message.Sender.Id == clientId {
			continue
		}
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg.message.Content))
		if err != nil {
			fmt.Printf("Error writing message to client:%d %w", clientId, err)
		}
	}
}

func (cs *WebSocketChatServer) BroadcastMessage(msg *BroadcastMessage) {
	cs.broadcastChan <- msg
}

func (cs *WebSocketChatServer) RegisterClient(msg *RegisterClientRequest) {
	cs.registerClientChan <- msg
}

func (cs *WebSocketChatServer) handleRegisterClient(msg *RegisterClientRequest, clientId int64) {
	fmt.Println("Register client message", clientId, msg.Connection)
	cs.clientConnections[clientId] = msg.Connection
	msg.UserIdChan <- clientId
	close(msg.UserIdChan)
}

func (cs *WebSocketChatServer) CreateUser(msg *CreateUserRequest) {
	cs.createUserChan <- msg
}
