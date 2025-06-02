package websocket

import (
	"log"
	"net/http"
	"server/internal/config"
	"strconv"
)

func InitWebSocketServer() {
	mux := http.NewServeMux()
	RegisterWebSocketRoutes(mux)

	addr := config.Config.WSHost + ":" + strconv.Itoa(config.Config.WSPort)
	log.Printf("Server running on %s", addr)

	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func RegisterWebSocketRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ws", ServeWS)
}
