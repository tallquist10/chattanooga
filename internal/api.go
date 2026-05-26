package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

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
}

type WebSocketChatServer struct {
	ChatServer
	msgChan            chan *WebSocketMessage
	closeChan          chan bool
	createUserChan     chan *CreateUserRequest
	registerClientChan chan *RegisterClientRequest
	clientConnections  map[int64]*websocket.Conn
	clientUsers        map[int64]*User
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
		case msg := <-cs.createUserChan:
			cs.handleCreateUser(msg)
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

func (cs *WebSocketChatServer) handleCreateUser(msg *CreateUserRequest) {
	select {
	case user := <-msg.ResultChan:
		cs.clientUsers[user] = msg.User
	case _ = <-msg.ErrorChan:
		fmt.Printf("Error creating user")
	}
}

func (cs *WebSocketChatServer) HandleWebSocketMessage(msg *WebSocketMessage) error {
	switch msg.Type {
	case RegisterUser:
		user, err := parseMessage[User](msg)
		if err != nil {
			fmt.Println("Error parsing user request")
		}
		resultChan := make(chan int64)
		errorChan := make(chan error)
		cs.CreateUser(&CreateUserRequest{
			User:       user,
			ResultChan: resultChan,
			ErrorChan:  errorChan,
		})
		return nil
	case ChatMessage:
		chatMsg, err := parseMessage[ChatRoomMessage](msg)
		if err != nil {
			fmt.Println("Error parsing chat message")
		}
		cs.ReceiveMessage(chatMsg)
		return nil
	default:
		return fmt.Errorf("Unrecognized websocket message type: %s", msg.Type)
	}
}

func parseMessage[T any](msg *WebSocketMessage) (*T, error) {
	var result *T
	err := json.Unmarshal(msg.Payload, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
