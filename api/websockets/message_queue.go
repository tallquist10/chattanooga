package websockets

import (
	"log/slog"
	"time"

	ws "github.com/gorilla/websocket"
)

type PingInterval int
type MessageQueueBufferSize int

type WebSocketClientMessageQueue struct {
	ClientId       int64
	conn           *ws.Conn
	msgChan        chan *WebSocketClientMessage
	pingInterval   time.Ticker
	pingIntervalMs time.Duration
	closed         bool
}

type WebSocketClientMessage struct {
	messageType    int
	messageContent []byte
}

type ClosedConnectionError struct {
	ClientId int64
	Message  string
}

func (e *ClosedConnectionError) Error() string {
	return e.Message
}

func NewMessageQueue(clientId int64, conn *ws.Conn, bufferSize MessageQueueBufferSize, pingIntervalMs PingInterval) *WebSocketClientMessageQueue {
	pingInterval := time.Duration(pingIntervalMs) * time.Millisecond
	return &WebSocketClientMessageQueue{
		ClientId:       clientId,
		conn:           conn,
		msgChan:        make(chan *WebSocketClientMessage, bufferSize),
		pingIntervalMs: pingInterval,
		pingInterval:   *time.NewTicker(pingInterval),
	}
}

func (mq *WebSocketClientMessageQueue) ReceiveMessage(msg *WebSocketClientMessage) error {
	if mq.closed {
		return &ClosedConnectionError{
			ClientId: mq.ClientId,
			Message:  "message not sent, connection was previously closed",
		}
	}
	mq.msgChan <- msg
	return nil
}

func (mq *WebSocketClientMessageQueue) SendMessages() {
	slog.Debug("Forwarding messages for client", "clientId", mq.ClientId)
	for {
		var err error
		select {
		case msg := <-mq.msgChan:
			if msg.messageType == ws.TextMessage {
				err = mq.conn.WriteMessage(msg.messageType, msg.messageContent)
			} else {
				err = mq.conn.WriteControl(msg.messageType, msg.messageContent, time.Now().Add(10*time.Second))
			}
			if err != nil {
				slog.Debug("Error writing message to client", "clientId", mq.ClientId, "error", err)
			}
		case _ = <-mq.pingInterval.C:
			slog.Debug("Sending ping", "clientId", mq.ClientId)
			err = mq.conn.WriteControl(ws.PingMessage, []byte("are you still there?"), time.Now().Add(10*time.Second))
			if err != nil {
				slog.Debug("Failed sending ping to client", "clientId", mq.ClientId, "error", err)
			}
		}
	}
}

func (mq *WebSocketClientMessageQueue) Close() {
	close(mq.msgChan)
	mq.closed = true
	slog.Info("Client connection closed", "clientId", mq.ClientId)
}
