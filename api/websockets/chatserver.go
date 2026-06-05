package websockets

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tallquist10/chat-server/api"
)

type ChatServer interface {
	RegisterUser(msg *api.User)
	RegisterClient(msg *api.ClientMessage)
	ReceiveMessage(msg *api.ChatRoomMessage)
	BroadcastMessage(msg *api.Message, chatRoom *api.ChatRoom)
}

type ChatServerConfig struct {
	ConcurrentProcessors int
	RateLimitPerUser     int
	BufferSize           int
}

type WebSocketChatServer struct {
	ChatServer
	msgChan            chan *api.WebSocketMessage
	closeChan          chan bool
	registerClientChan chan *api.RegisterClientRequest
	clientConnections  map[int64]*websocket.Conn
	clientUsers        map[int64]*api.User
	incomingChan       chan *api.ChatRoomMessage
	broadcastChan      chan *api.BroadcastMessage
	config             *ChatServerConfig
}

func NewChatServer(
	config *ChatServerConfig,
) *WebSocketChatServer {
	return &WebSocketChatServer{
		registerClientChan: make(chan *api.RegisterClientRequest, config.BufferSize),
		clientConnections:  make(map[int64]*websocket.Conn),
		incomingChan:       make(chan *api.ChatRoomMessage, config.BufferSize),
		broadcastChan:      make(chan *api.BroadcastMessage, config.BufferSize),
		closeChan:          make(chan bool),
	}
}

func (cs *WebSocketChatServer) Start() error {
	defer close(cs.registerClientChan)
	defer close(cs.incomingChan)
	defer close(cs.broadcastChan)
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

func (cs *WebSocketChatServer) ReceiveMessage(msg *api.ChatRoomMessage) {
	cs.incomingChan <- msg
}

func (cs *WebSocketChatServer) handleReceiveMessage(msg *api.ChatRoomMessage) {
	outgoingMsg := &api.BroadcastMessage{
		Message: &api.Message{
			Sender: &api.User{
				Id: msg.UserId,
			},
			Content: string(msg.Content),
		},
	}
	cs.BroadcastMessage(outgoingMsg)
}

func (cs *WebSocketChatServer) handleBroadcastMessage(msg *api.BroadcastMessage) {
	for clientId, conn := range cs.clientConnections {
		if msg.Message.Sender.Id == clientId {
			continue
		}
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Message.Content))
		if err != nil {
			fmt.Printf("Error writing message to client:%d %q", clientId, err)
		}
	}
}

func (cs *WebSocketChatServer) BroadcastMessage(msg *api.BroadcastMessage) {
	cs.broadcastChan <- msg
}

func (cs *WebSocketChatServer) RegisterClient(msg *api.RegisterClientRequest) {
	cs.registerClientChan <- msg
}

func (cs *WebSocketChatServer) handleRegisterClient(msg *api.RegisterClientRequest, clientId int64) {
	fmt.Println("Register client message", clientId, msg.Connection)
	cs.clientConnections[clientId] = msg.Connection
	msg.UserIdChan <- clientId
	close(msg.UserIdChan)
}

func (cs *WebSocketChatServer) HandleWebSocketMessage(msg *api.WebSocketMessage) error {
	switch msg.Type {
	case api.ChatMessage:
		chatMsg, err := parseMessage[api.ChatRoomMessage](msg)
		if err != nil {
			fmt.Println("Error parsing chat message")
		}
		cs.ReceiveMessage(chatMsg)
		return nil
	default:
		return fmt.Errorf("Unrecognized websocket message type: %s", msg.Type)
	}
}

func parseMessage[T any](msg *api.WebSocketMessage) (*T, error) {
	var result T
	err := json.Unmarshal(msg.Payload, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
