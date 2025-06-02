package main

import (
	"log"
	"server/internal/db"
	"server/internal/utils"
	"server/internal/websocket"
)

func main() {
	if utils.CheckAndPrintNetwork() {
		defer func() {
			if err := db.DB.Close(); err != nil {
				log.Printf("Error closing MySQL connection: %v", err)
			}
			db.CloseMongo()
		}()
		db.InitMySQL()
		db.InitMongo()

		websocket.InitWebSocketServer()
	}
}
