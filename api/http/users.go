package http

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tallquist10/chat-server/api"
	db "github.com/tallquist10/chat-server/db/sqlc/sqlite"
)

type UsersApi struct {
	dbConnection   *db.Queries
	closeChan      chan bool
	createUserChan chan *api.DBHandler[api.CreateUserRequest]
}

func NewUsersApi(conn *sql.DB) *UsersApi {
	return &UsersApi{
		dbConnection:   db.New(conn),
		closeChan:      make(chan bool),
		createUserChan: make(chan *api.DBHandler[api.CreateUserRequest], 1000),
	}
}

func (u *UsersApi) Start() error {
	defer close(u.createUserChan)
	defer close(u.closeChan)
	for {
		select {
		case createUserReq := <-u.createUserChan:
			slog.Debug("Received create user request")
			go u.createUser(createUserReq)
		case _ = <-u.closeChan:
			return nil
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (u *UsersApi) CreateUser(c *gin.Context) {
	user, err := api.FormatJsonInput[api.User](c)

	if err != nil {
		slog.Error("Failed to decode CreateUserRequest", "error", err.Error())
		api.WriteResponse(c, http.StatusBadRequest, &err)
		return
	}

	req := &api.CreateUserRequest{
		User:       user,
		ResultChan: make(chan int64),
		ErrorChan:  make(chan error),
	}

	u.createUserChan <- &api.DBHandler[api.CreateUserRequest]{
		Request: req,
		Context: c,
	}

	for {
		select {
		case user := <-req.ResultChan:
			var result struct {
				Id int64 `json:"id"`
			}
			result.Id = user
			api.WriteResponse(c, http.StatusCreated, &result)
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

func (u *UsersApi) createUser(req *api.DBHandler[api.CreateUserRequest]) {
	user, err := u.dbConnection.CreateUser(
		req.Context,
		db.CreateUserParams{
			Username:    req.Request.User.Username,
			DisplayName: req.Request.User.DisplayName,
		},
	)
	if err != nil {
		req.Request.ErrorChan <- err
	}
	req.Request.ResultChan <- user.ID
}
