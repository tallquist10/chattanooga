package http

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tallquist10/chat-server/api"
	"github.com/tallquist10/chat-server/db"
)

type UsersApi struct {
	dbConnection   *db.Queries
	createUserChan chan *DBHandler[api.CreateUserRequest]
	closeChan      chan bool
}

type DBHandler[T any] struct {
	request *T
	context *gin.Context
}

func NewUsersApi(conn *sql.DB) *UsersApi {
	return &UsersApi{
		dbConnection: db.New(conn),
	}
}

func (u *UsersApi) Start() error {
	defer close(u.createUserChan)
	defer close(u.closeChan)
	for {
		select {
		case createUserReq := <-u.createUserChan:
			go u.createUser(createUserReq)
		case _ = <-u.closeChan:
			return nil
		}
	}
}

func (u *UsersApi) CreateUser(c *gin.Context) {
	req, err := api.FormatJsonInput[api.CreateUserRequest](c)

	if err != nil {
		fmt.Println("Failed to decode CreateUserRequest", err.Error())
		api.WriteResponse(c, http.StatusBadRequest, &err)
		return
	}

	u.createUserChan <- &DBHandler[api.CreateUserRequest]{
		request: req,
		context: c,
	}

	select {
	case user := <-req.ResultChan:
		api.WriteResponse(c, http.StatusCreated, &struct{ id int64 }{id: user})
	case err = <-req.ErrorChan:
		api.WriteResponse(c, http.StatusUnprocessableEntity, &err)
	}
}
func (u *UsersApi) createUser(req *DBHandler[api.CreateUserRequest]) {
	user, err := u.dbConnection.CreateUser(
		req.context,
		db.CreateUserParams{
			Username:    req.request.User.Username,
			DisplayName: req.request.User.DisplayName,
		},
	)
	if err != nil {
		req.request.ErrorChan <- err
	}
	req.request.ResultChan <- user.ID
}
