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
	ticker := time.NewTicker(time.Duration(config.PingIntervalMs) * time.Millisecond)
	return &WebSocketChatServer{
		registerClientChan: make(chan *api.RegisterClientRequest, config.BufferSize),
		closeClientChan:    make(chan *api.CloseClientRequest, config.BufferSize),
		clientConnections:  make(map[int64]*WebSocketClientMessageQueue),
		incomingChan:       make(chan *api.DBHandler[api.ChatRoomMessage], 5*config.BufferSize),
		broadcastChan:      make(chan *api.BroadcastMessage, config.BufferSize),
		pingsChan:          make(chan int64, config.BufferSize),
		closeChan:          make(chan bool),
		dbConnection:       db.New(conn),
		pingInterval:       ticker,
		upgrader: &ws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (cs *WebSocketChatServer) Start() error {
	defer close(cs.registerClientChan)
	defer close(cs.closeClientChan)
	defer close(cs.incomingChan)
	defer close(cs.broadcastChan)
	defer close(cs.closeChan)
	defer close(cs.pingsChan)

	var clientId int64 = 1
	// go cs.handleBroadcastMessage(cs.broadcastChan)
	// go cs.handleRegisterClient(cs.registerClientChan)
	// go cs.handleCloseClient(cs.closeClientChan)
	// go cs.handleReceiveMessage(cs.incomingChan)

	// synchronize map iterations/writes
	// broadcast
	// register
	// close
	// ping

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
		case _ = <-cs.pingInterval.C:
			cs.sendPings()
		case clientId := <-cs.pingsChan:
			go cs.handlePing(clientId)
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
			slog.Info("Connection closed", "clientId", clientId)
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
	// for req := range incomingChan {
	// persist the nessage to the messages table
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
	// }
}

// func (cs *WebSocketChatServer) handleBroadcastMessage(broadcastChan chan *api.BroadcastMessage) {
func (cs *WebSocketChatServer) handleBroadcastMessage(msg *api.BroadcastMessage) {
	// for msg := range broadcastChan {
	slog.Debug("Broadcast message", msg.Message.Sender.Username, msg.Message.Content)
	go func() {
		for clientId, msgQueue := range cs.clientConnections {
			if msg.Message.Sender.Id == clientId || msgQueue.clientClosed {
				return
			}
			msgQueue.msgChan <- &WebSocketClientMessage{
				messageType:    websocket.TextMessage,
				messageContent: []byte(msg.Message.Content),
			}
		}
	}()
	// }
}

func (cs *WebSocketChatServer) sendMessages(clientId int64, msgQueue *WebSocketClientMessageQueue) {
	slog.Debug("Forwarding messages for client", "clientId", clientId)
	for msg := range msgQueue.msgChan {
		if msgQueue.clientClosed {
			slog.Debug("Received message for closed client, skipping")
			continue
		}
		var err error
		if msg.messageType == websocket.TextMessage {
			err = msgQueue.conn.WriteMessage(msg.messageType, msg.messageContent)
		} else {
			err = msgQueue.conn.WriteControl(msg.messageType, msg.messageContent, time.Now().Add(10*time.Second))
		}
		if err != nil {
			slog.Debug("Error writing message to client", "clientId", clientId, "error", err)
		}
	}
}

func (cs *WebSocketChatServer) BroadcastMessage(msg *api.BroadcastMessage) {
	cs.broadcastChan <- msg
}

func (cs *WebSocketChatServer) RegisterClient(msg *api.RegisterClientRequest) {
	cs.registerClientChan <- msg
}

// func (cs *WebSocketChatServer) handleRegisterClient(registerClientChan chan *api.RegisterClientRequest) {
func (cs *WebSocketChatServer) handleRegisterClient(msg *api.RegisterClientRequest, clientId int64) {
	// for msg := range registerClientChan {
	slog.Debug("Register client message", "clientId", clientId, "conn", msg.Connection)
	msgQueue := &WebSocketClientMessageQueue{
		conn:         msg.Connection,
		msgChan:      make(chan *WebSocketClientMessage),
		clientClosed: false,
	}
	cs.clientConnections[clientId] = msgQueue
	go cs.sendMessages(clientId, msgQueue)
	msg.UserIdChan <- clientId
	close(msg.UserIdChan)
	// }
}

// func (cs *WebSocketChatServer) handleCloseClient(closeClientChan chan *api.CloseClientRequest) {
func (cs *WebSocketChatServer) handleCloseClient(msg *api.CloseClientRequest) {
	// for msg := range closeClientChan {
	clientMessageQueue, ok := cs.clientConnections[msg.UserId]
	if !ok {
		slog.Warn("Received close connection request for unknown client id", "clientId", msg.UserId)
		// continue
		return
	}
	close(clientMessageQueue.msgChan)
	clientMessageQueue.clientClosed = true
	delete(cs.clientConnections, msg.UserId)
	// }
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
	case websocket.PingMessage:
		cs.pingsChan <- clientId
		return false, nil
	case websocket.PongMessage:
		slog.Info("Received pong message from client", "clientId", clientId)
		return false, nil
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

func (cs *WebSocketChatServer) handlePing(clientId int64) error {
	mq, ok := cs.clientConnections[clientId]
	if !ok {
		return fmt.Errorf("Received ping from unregistered client %d\n", clientId)
	}

	mq.msgChan <- &WebSocketClientMessage{
		messageType:    websocket.PongMessage,
		messageContent: []byte("don't worry, I'm still here!"),
	}
	return nil
}

func (cs *WebSocketChatServer) sendPings() {
	for clientId, mq := range cs.clientConnections {
		go func() {
			slog.Debug("Sending ping", "clientId", clientId)
			mq.msgChan <- &WebSocketClientMessage{
				messageType:    websocket.PingMessage,
				messageContent: []byte("are you still there?"),
			}
		}()
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
