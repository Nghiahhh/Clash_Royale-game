package websocket

import (
	"log"
	"server/internal/handle/message"
	"server/internal/types"
	"server/internal/utils"
)

func handleGameMessage(c *types.Client, msg []byte) {
	var incoming utils.IncomingMessage
	err := utils.UnmarshalMessage(msg, &incoming)
	if err != nil {
		log.Printf("Invalid JSON from %s: %v", c.Conn.RemoteAddr(), err)
		utils.SendError(c.Send, "", "invalid_json", "Malformed JSON")
		return
	}

	switch incoming.Type {
	// Dùng để test
	// case "echo":
	// 	utils.SendJSON(c.Send, utils.OutgoingMessage{
	// 		ID:   incoming.ID,
	// 		Type: "echo",
	// 		Data: incoming.Data,
	// 	})

	case "login":
		message.HandleLogin(c, incoming)

	case "re_login":
		message.HandleReLogin(c, incoming)

	case "register":
		message.HandleRegister(c, incoming)

	case "get_user_cards":
		message.HandleGetUserCards(c, incoming)

	case "get_user_deck":
		message.HandleGetUserDeck(c, incoming)

	case "swap_card":
		message.HandleSwapCard(c, incoming)

	case "create_lobby":
		message.HandleCreateLobby(c, incoming)

	case "join_lobby":
		message.HandleJoinLobby(c, incoming)

	case "match_lobby":
		message.HandleMatchLobby(c, incoming)

	case "leave_lobby":
		message.HandleLeaveLobby(c, incoming)

	case "Release_card":
		message.HandleReleaseCard(c, incoming)

	default:
		utils.SendError(c.Send, incoming.ID, "unknown_type", "Unknown message type")
	}
}
