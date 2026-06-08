package http

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tallquist10/chat-server/api"
	"github.com/tallquist10/chat-server/db"
)

type ChatRoomsApi struct {
	dbConnection       *db.Queries
	closeChan          chan bool
	createChatRoomChan chan *api.DBHandler[api.CreateChatRoomRequest]
}

func NewChatRoomsApi(conn *sql.DB) *ChatRoomsApi {
	return &ChatRoomsApi{
		dbConnection:       db.New(conn),
		createChatRoomChan: make(chan *api.DBHandler[api.CreateChatRoomRequest], 1000),
		closeChan:          make(chan bool),
	}
}

func (cr *ChatRoomsApi) Start() {
	defer close(cr.createChatRoomChan)
	defer close(cr.closeChan)
	for {
		select {
		case req := <-cr.createChatRoomChan:
			fmt.Printf("Creating chat room: %s\n", req.Request.Name)
			cr.createChatRoom(req)
		case _ = <-cr.closeChan:
			return
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (cr *ChatRoomsApi) CreateChatRoom(c *gin.Context) {
	chatroom, err := api.FormatJsonInput[struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}](c)

	if err != nil {
		fmt.Println("Failed to decode CreateChatRoomRequest", err.Error())
		api.WriteResponse(c, http.StatusBadRequest, &err)
		return
	}

	req := &api.CreateChatRoomRequest{
		Name:        chatroom.Name,
		Description: chatroom.Description,
		ResultChan:  make(chan *api.ChatRoom),
		ErrorChan:   make(chan error),
	}

	cr.createChatRoomChan <- &api.DBHandler[api.CreateChatRoomRequest]{
		Request: req,
		Context: c,
	}

	for {
		select {
		case chatRoom := <-req.ResultChan:
			api.WriteResponse(c, http.StatusCreated, chatRoom)
			return
		case err = <-req.ErrorChan:
			api.WriteResponse(c, http.StatusUnprocessableEntity, &err)
			return
		case <-c.Done():
			api.WriteResponse[string](c, http.StatusRequestTimeout, nil)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

}

func (cr *ChatRoomsApi) createChatRoom(req *api.DBHandler[api.CreateChatRoomRequest]) {
	chatRoom, err := cr.dbConnection.CreateChatRoom(
		req.Context,
		db.CreateChatRoomParams{
			Name:        req.Request.Name,
			Description: sql.NullString{String: req.Request.Description, Valid: req.Request.Description != ""},
		})

	if err != nil {
		req.Request.ErrorChan <- err
	}

	req.Request.ResultChan <- &api.ChatRoom{
		Id:       chatRoom.ID,
		Name:     chatRoom.Name,
		Messages: make([]*api.Message, 0),
	}
}
