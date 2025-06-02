// internal/utils/json.go
package utils

import (
	"encoding/json"
	"log"
)

type IncomingMessage struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type OutgoingMessage struct {
	ID   string      `json:"id"`
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func UnmarshalMessage(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

func SendJSON(send chan<- []byte, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	send <- jsonData
}

func SendError(send chan<- []byte, id, errorType, message string) {
	if id == "" {
		id = "unknown"
	}
	SendJSON(send, OutgoingMessage{
		ID:   id,
		Type: "error",
		Data: map[string]string{
			"error":   errorType,
			"message": message,
		},
	})
}

func ParseIncomingMessage(data []byte) (*IncomingMessage, error) {
	var msg IncomingMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func SendMessage(send chan<- []byte, id string, typ string, data interface{}) {
	msg := OutgoingMessage{
		ID:   id,
		Type: typ,
		Data: data,
	}
	SendJSON(send, msg)
}
