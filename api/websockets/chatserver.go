package websockets

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/gorilla/websocket"
	"github.com/tallquist10/chat-server/api"
	db "github.com/tallquist10/chat-server/db/sqlc/sqlite"
)

func NewChatServer(
	config *api.ChatServerConfig,
	conn *sql.DB,
) *WebSocketChatServer {
	return &WebSocketChatServer{
		registerClientChan: make(chan *api.RegisterClientRequest, config.BufferSize),
		closeClientChan:    make(chan *api.CloseClientRequest, config.BufferSize),
		clientConnections:  make(map[int64]*WebSocketClientMessageQueue),
		incomingChan:       make(chan *api.DBHandler[api.ChatRoomMessage], 5*config.BufferSize),
		broadcastChan:      make(chan *api.BroadcastMessage, config.BufferSize),
		closeChan:          make(chan bool),
		dbConnection:       db.New(conn),
		upgrader: &ws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		config: config,
	}
}

func (cs *WebSocketChatServer) Start() error {
	defer close(cs.registerClientChan)
	defer close(cs.closeClientChan)
	defer close(cs.incomingChan)
	defer close(cs.broadcastChan)
	defer close(cs.closeChan)

	var clientId int64 = 1

	for {
		select {
		case _ = <-cs.closeChan:
			return nil
		case msg := <-cs.broadcastChan:
			cs.handleBroadcastMessage(msg)
		case msg := <-cs.registerClientChan:
			cs.handleRegisterClient(msg, clientId)
			clientId++
		case msg := <-cs.closeClientChan:
			cs.handleCloseClient(msg)
		case msg := <-cs.incomingChan:
			cs.handleReceiveMessage(msg)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (cs *WebSocketChatServer) HandleConnection(c *gin.Context) {
	conn, err := cs.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("Failed to set websocket upgrade:", "error", err)
		return
	}

	clientIdChan := make(chan int64)
	registerClientRequest := &api.RegisterClientRequest{
		Connection: conn,
		UserIdChan: clientIdChan,
	}
	cs.RegisterClient(registerClientRequest)
	clientId := <-registerClientRequest.UserIdChan
	slog.Info("Registered client", "clientId", clientId)
	conn.WriteMessage(ws.TextMessage, []byte(fmt.Sprintf("Your client id is %d. Please use this client id in all of your requests", clientId)))

	// Ensure the connection is closed when the function returns
	defer conn.Close()

	// Read and write messages in a loop
	for {
		msgType, content, err := conn.ReadMessage()
		if err != nil && msgType != WebSocketClose {
			slog.Error("Error reading message", "error", err)
			break
		}
		slog.Debug("Received message", "content", string(content))

		closed, err := cs.HandleWebSocketMessage(c, clientId, msgType, content)
		if err != nil {
			slog.Error("Error encountered on connection", "clientId", clientId, "error", err)
			break
		}

		if closed {
			break
		}
	}
}

func (cs *WebSocketChatServer) ReceiveMessage(c *gin.Context, msg *api.ChatRoomMessage) {
	cs.incomingChan <- &api.DBHandler[api.ChatRoomMessage]{
		Request: msg,
		Context: c,
	}
}

// func (cs *WebSocketChatServer) handleReceiveMessage(incomingChan chan *api.DBHandler[api.ChatRoomMessage]) {
func (cs *WebSocketChatServer) handleReceiveMessage(req *api.DBHandler[api.ChatRoomMessage]) {
	dbMsg, err := cs.dbConnection.CreateMessage(req.Context, db.CreateMessageParams{
		UserID:     req.Request.UserId,
		ChatRoomID: req.Request.ChannelId,
		Content:    req.Request.Content,
	})

	if err != nil {
		slog.Error("Failed to persist chat message", "error", err.Error())
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
	slog.Debug("Broadcast message", msg.Message.Sender.Username, msg.Message.Content)
	// database call for channel participants, loop through those instead
	go func() {
		for clientId, msgQueue := range cs.clientConnections {
			if msg.Message.Sender.Id == clientId {
				return
			}
			clientMsg := &WebSocketClientMessage{
				messageType:    websocket.TextMessage,
				messageContent: []byte(msg.Message.Content),
			}
			err := msgQueue.ReceiveMessage(clientMsg)
			if err != nil {
				slog.Error(err.Error(), "clientId", clientId)
				continue
			}
			slog.Info("Send message to client", "clientId", clientId, "sender", msg.Message.Sender.Id, "content", msg.Message.Content)
		}
	}()
}

func (cs *WebSocketChatServer) BroadcastMessage(msg *api.BroadcastMessage) {
	cs.broadcastChan <- msg
}

func (cs *WebSocketChatServer) RegisterClient(msg *api.RegisterClientRequest) {
	cs.registerClientChan <- msg
}

func (cs *WebSocketChatServer) handleRegisterClient(msg *api.RegisterClientRequest, clientId int64) {
	slog.Debug("Register client message", "clientId", clientId, "conn", msg.Connection)
	msg.Connection.SetPingHandler(func(appData string) error {
		slog.Debug("Received ping from client", "clientId", clientId)
		return msg.Connection.WriteControl(ws.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
	})
	msg.Connection.SetPongHandler(func(appData string) error {
		slog.Info("Received pong from client", "clientId", clientId)
		return nil
	})
	msgQueue := NewMessageQueue(clientId, msg.Connection, MessageQueueBufferSize(25), PingInterval(cs.config.PingIntervalMs))
	cs.clientConnections[clientId] = msgQueue
	go msgQueue.SendMessages()
	msg.UserIdChan <- clientId
	close(msg.UserIdChan)
}

func (cs *WebSocketChatServer) handleCloseClient(msg *api.CloseClientRequest) {
	clientMessageQueue, ok := cs.clientConnections[msg.UserId]
	if !ok {
		slog.Warn("Received close connection request for unknown client id", "clientId", msg.UserId)
		return
	}

	clientMessageQueue.Close()
	delete(cs.clientConnections, msg.UserId)
}

func (cs *WebSocketChatServer) HandleWebSocketMessage(c *gin.Context, clientId int64, msgType int, content []byte) (bool, error) {
	switch msgType {
	case websocket.TextMessage:
		var msg WebSocketMessage
		err := json.Unmarshal(content, &msg)
		if err != nil {
			return false, err
		}
		return false, cs.handleChatMessage(c, &msg)
	case websocket.BinaryMessage:
		return false, fmt.Errorf("Server is not configured to receive binary messages") // error
	case websocket.CloseMessageTooBig, websocket.CloseMessage, WebSocketClose:
		var messageType string
		switch msgType {
		case WebSocketClose:
			messageType = "ClientTerminated"
		case websocket.CloseMessageTooBig:
			messageType = "MessageTooBig"
		case websocket.CloseMessage:
			messageType = "CloseMessage"
		default:
			messageType = strconv.Itoa(msgType)
		}
		slog.Info("Closing connection", "type", messageType, "content", string(content), "clientId", clientId)

		cs.handleCloseClientMessage(clientId)
		return true, nil
	default:
		return false, fmt.Errorf("Unknown message type %d", msgType)
	}

}

func (cs *WebSocketChatServer) handleChatMessage(c *gin.Context, msg *WebSocketMessage) error {
	switch msg.Type {
	case ChatMessage:
		chatMsg, err := parseMessage[api.ChatRoomMessage](msg)
		if err != nil {
			slog.Error("Error parsing chat message", "error", err)
		}
		cs.ReceiveMessage(c, chatMsg)
		return nil
	default:
		return fmt.Errorf("Unrecognized websocket message type: %s", msg.Type)
	}
}

func (cs *WebSocketChatServer) handleCloseClientMessage(clientId int64) {
	cs.closeClientChan <- &api.CloseClientRequest{
		UserId: clientId,
	}
}

func parseMessage[T any](msg *WebSocketMessage) (*T, error) {
	var result T
	err := json.Unmarshal(msg.Payload, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
