package websockets

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/gorilla/websocket"
	"github.com/tallquist10/chat-server/api"
	"github.com/tallquist10/chat-server/db"
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
	upgrader           *ws.Upgrader
	msgChan            chan *api.WebSocketMessage
	closeChan          chan bool
	registerClientChan chan *api.RegisterClientRequest
	clientConnections  map[int64]*ws.Conn
	clientUsers        map[int64]*api.User
	incomingChan       chan *api.DBHandler[api.ChatRoomMessage]
	broadcastChan      chan *api.BroadcastMessage
	dbConnection       *db.Queries
	config             *ChatServerConfig
}

func NewChatServer(
	config *ChatServerConfig,
	conn *sql.DB,
) *WebSocketChatServer {
	return &WebSocketChatServer{
		registerClientChan: make(chan *api.RegisterClientRequest, config.BufferSize),
		clientConnections:  make(map[int64]*ws.Conn),
		incomingChan:       make(chan *api.DBHandler[api.ChatRoomMessage], config.BufferSize),
		broadcastChan:      make(chan *api.BroadcastMessage, config.BufferSize),
		closeChan:          make(chan bool),
		dbConnection:       db.New(conn),
		upgrader: &ws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
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

func (cs *WebSocketChatServer) HandleConnection(c *gin.Context) {
	conn, err := cs.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade:", err)
		return
	}

	clientIdChan := make(chan int64)
	registerClientRequest := &api.RegisterClientRequest{
		Connection: conn,
		UserIdChan: clientIdChan,
	}
	cs.RegisterClient(registerClientRequest)
	clientId := <-registerClientRequest.UserIdChan
	fmt.Printf("Registered client %d\n", clientId)
	conn.WriteMessage(ws.TextMessage, []byte(fmt.Sprintf("Your client id is %d. Please use this client id in all of your requests", clientId)))

	// Ensure the connection is closed when the function returns
	defer conn.Close()

	// Read and write messages in a loop
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		fmt.Printf("Received message: %s\n", p)

		var msg api.WebSocketMessage
		err = json.Unmarshal(p, &msg)
		if err != nil {
			fmt.Printf("Failed to parse message: %e", err)
		}

		cs.HandleWebSocketMessage(c, &msg)
	}
}

func (cs *WebSocketChatServer) ReceiveMessage(c *gin.Context, msg *api.ChatRoomMessage) {

	cs.incomingChan <- &api.DBHandler[api.ChatRoomMessage]{
		Request: msg,
		Context: c,
	}
}

func (cs *WebSocketChatServer) handleReceiveMessage(req *api.DBHandler[api.ChatRoomMessage]) {

	// persist the nessage to the messages table
	dbMsg, err := cs.dbConnection.CreateMessage(req.Context, db.CreateMessageParams{
		UserID:     req.Request.UserId,
		ChatRoomID: req.Request.ChannelId,
		Content:    req.Request.Content,
	})

	if err != nil {
		fmt.Printf("Failed to persist chat message: %s", err.Error())
		api.WriteResponse(req.Context, http.StatusInternalServerError, &err)
	}

	outgoingMsg := &api.BroadcastMessage{
		Message: &api.Message{
			Content: dbMsg.Content,
			Sender: &api.User{
				Id: dbMsg.UserID,
			},
		},
		ChatRoom: &api.ChatRoom{
			Id: dbMsg.ChatRoomID,
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

func (cs *WebSocketChatServer) HandleWebSocketMessage(c *gin.Context, msg *api.WebSocketMessage) error {
	switch msg.Type {
	case api.ChatMessage:
		chatMsg, err := parseMessage[api.ChatRoomMessage](msg)
		if err != nil {
			fmt.Println("Error parsing chat message")
		}
		cs.ReceiveMessage(c, chatMsg)
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
