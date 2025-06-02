package websocket

import (
	"log"
	"net/http"
	"server/internal/session"
	"server/internal/types"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &types.Client{
		Conn:  conn,
		Send:  make(chan []byte, 20),
		Inbox: make(chan []byte, 20),
	}

	session.AddClient(client)

	log.Printf("Client connected: %s", conn.RemoteAddr())

	go readPump(client)
	go writePump(client)
	go processMessages(client)
}

func readPump(c *types.Client) {
	defer func() {
		c.Once.Do(func() {
			session.RemoveClient(c)
			close(c.Send)
			c.Conn.Close()
			log.Printf("Client disconnected: %s", c.Conn.RemoteAddr())
		})
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Read error from %s: %v", c.Conn.RemoteAddr(), err)
			break
		}
		c.Inbox <- msg
	}
}

func writePump(c *types.Client) {
	defer func() {
		c.Once.Do(func() {
			session.RemoveClient(c)
			close(c.Send)
			c.Conn.Close()
			log.Printf("Client disconnected: %s", c.Conn.RemoteAddr())
		})
	}()

	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("Write error to %s: %v", c.Conn.RemoteAddr(), err)
			break
		}
	}
}

func processMessages(c *types.Client) {
	for msg := range c.Inbox {
		handleGameMessage(c, msg)
	}
}
