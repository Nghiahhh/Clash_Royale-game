// internal/types/client.go
package types

import (
	"sync"

	"github.com/gorilla/websocket"
)

type User struct {
	ID        int
	Frame     int
	MatchRoom chan []byte
}

type Client struct {
	Conn      *websocket.Conn
	Send      chan []byte
	Inbox     chan []byte
	Once      sync.Once
	User      User
	RoomType  string
	SlotIndex int
}
