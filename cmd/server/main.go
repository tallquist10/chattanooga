package main

import (
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tallquist10/chat-server/api"
	servers "github.com/tallquist10/chat-server/api/http"
	"github.com/tallquist10/chat-server/api/websockets"
	"github.com/tallquist10/chat-server/db"
	"github.com/tallquist10/chat-server/db/migrations"
)

func main() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	migrations.SqlMigrations()
	conn, err := db.New()
	if err != nil {
		log.Fatal("Failed to inintialize db connection", err)
	}

	bufferSize, err := strconv.Atoi(os.Getenv("BUFFER_SIZE"))
	if err != nil {
		slog.Warn("Failed to extract BUFFER_SIZE, defaulting to 100")
		bufferSize = 100
	}
	chatServer := websockets.NewChatServer(
		&api.ChatServerConfig{
			ConcurrentProcessors: 1,
			BufferSize:           bufferSize,
			RateLimitPerUser:     10,
			PingIntervalMs:       10000,
		},
		conn,
	)
	go chatServer.Start()

	userApi := servers.NewUsersApi(conn)
	go userApi.Start()

	chatRoomsApi := servers.NewChatRoomsApi(conn)
	go chatRoomsApi.Start()

	router := gin.Default()

	router.POST("/users", userApi.CreateUser)
	router.POST("/chatrooms", chatRoomsApi.CreateChatRoom)
	router.GET("/ws", chatServer.HandleConnection)
	address := os.Getenv("ADDR")
	if address == "" {
		log.Fatal("No address for server")
	}
	slog.Info("Chat server started", "address", address)
	log.Fatal(router.Run(address))
}
