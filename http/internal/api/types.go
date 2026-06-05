package api

import "github.com/tallquist10/chat-server/shared/api"

type CreateUserRequest struct {
	User       *api.User
	ResultChan chan int64
	ErrorChan  chan error
}
