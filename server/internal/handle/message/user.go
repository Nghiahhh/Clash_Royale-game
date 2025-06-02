package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"server/internal/auth"
	"server/internal/db"
	"server/internal/session"
	"server/internal/types"
	"server/internal/utils"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type LoginRequest struct {
	Gmail    string `json:"gmail"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

type ReLoginRequest struct {
	Token string `json:"token"`
}

type RegisterRequest struct {
	Gmail    string `json:"gmail"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Chỉ dùng cho King Tower (chỉ có name)
type SimpleTowerInfo struct {
	Name string `bson:"name" json:"name"`
}

// Dùng cho Guard Tower (có level và count)
type TowerInfo struct {
	Name  string `bson:"name" json:"name"`
	Level int    `bson:"level" json:"level"`
	Count int    `bson:"count" json:"count"`
}

// Dùng cho user_cards (không có slot/index)
type CardInfo struct {
	Name  string `bson:"name" json:"name"`
	Level int    `bson:"level" json:"level"`
	Count int    `bson:"count" json:"count"`
}

// Dùng cho user_decks (có slot/index)
type DeckCard struct {
	Index int    `bson:"index" json:"index"`
	Name  string `bson:"name" json:"name"`
	Level int    `bson:"level" json:"level"`
}

type UserCards struct {
	UserID     int               `bson:"user_id" json:"user_id"`
	KingTower  []SimpleTowerInfo `bson:"king_tower" json:"king_tower"`   // Mảng
	GuardTower []TowerInfo       `bson:"guard_tower" json:"guard_tower"` // Mảng
	Cards      []CardInfo        `bson:"cards" json:"cards"`
}

type UserDeck struct {
	UserID     int             `bson:"user_id" json:"user_id"`
	KingTower  SimpleTowerInfo `bson:"king_tower" json:"king_tower"`   // Object
	GuardTower TowerInfo       `bson:"guard_tower" json:"guard_tower"` // Object
	Cards      []DeckCard      `bson:"cards" json:"cards"`
}

// Dữ liệu đầu vào chung cho lobby
type LobbyRequest struct {
	RoomType string `json:"room_type"`
	LobbyID  string `json:"lobby_id,omitempty"` // Chỉ dùng khi join
}

func HandleLogin(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID != 0 {
		utils.SendError(c.Send, incoming.ID, "already_logged_in", "User already logged in")
		return
	}

	var req LoginRequest
	if err := json.Unmarshal(incoming.Data, &req); err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_payload", "Invalid login format")
		return
	}

	if req.Gmail == "" || req.Password == "" {
		utils.SendError(c.Send, incoming.ID, "missing_fields", "Username or password missing")
		return
	}

	var storedPassword, username string
	var id int

	query := `SELECT id, username, password FROM users WHERE gmail = ?`
	err := db.DB.QueryRow(query, req.Gmail).Scan(&id, &username, &storedPassword)
	if err == sql.ErrNoRows {
		utils.SendError(c.Send, incoming.ID, "not_found", "User not found")
		return
	} else if err != nil {
		log.Printf("DB error: %v", err)
		utils.SendError(c.Send, incoming.ID, "server_error", "Database error")
		return
	}

	if storedPassword != req.Password {
		utils.SendError(c.Send, incoming.ID, "invalid_credentials", "Invalid username or password")
		return
	}

	if session.IsUserLoggedIn(id) {
		utils.SendError(c.Send, incoming.ID, "already_logged_in", "This account is already logged in on another device.")
		return
	} else {
		c.User.ID = id
		c.User.Frame = 2
	}

	token, err := auth.GenerateToken(id, req.Gmail, username)

	if err != nil {
		utils.SendError(c.Send, incoming.ID, "token_error", "Failed to generate token")
		return
	}

	utils.SendMessage(c.Send, incoming.ID, "login_success", LoginResponse{
		Token:    token,
		Username: username,
	})

}

func HandleReLogin(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID != 0 {
		utils.SendError(c.Send, incoming.ID, "already_logged_in", "User already logged in")
		return
	}

	var req ReLoginRequest
	if err := json.Unmarshal(incoming.Data, &req); err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_payload", "Invalid re_login format")
		return
	}

	if req.Token == "" {
		utils.SendError(c.Send, incoming.ID, "missing_fields", "Token missing")
		return
	}

	// Sử dụng ValidateTokenWithMongo để kiểm tra token và lấy username
	claims, err := auth.ValidateTokenWithMongo(req.Token)
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_token", err.Error())
		return
	}

	if session.IsUserLoggedIn(claims.ID) {
		utils.SendError(c.Send, incoming.ID, "already_logged_in", "This account is already logged in on another device.")
		return
	}

	// Lưu thông tin user vào client
	c.User.ID = claims.ID
	c.User.Frame = 2

	// Gửi lại token cũ (vì là vĩnh viễn)
	utils.SendMessage(c.Send, incoming.ID, "login_success", LoginResponse{
		Token:    req.Token,
		Username: claims.Username,
	})
}

func HandleRegister(c *types.Client, incoming utils.IncomingMessage) {
	var req RegisterRequest
	if err := json.Unmarshal(incoming.Data, &req); err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_payload", "Invalid register format")
		return
	}

	if req.Gmail == "" || req.Username == "" || req.Password == "" {
		utils.SendError(c.Send, incoming.ID, "missing_fields", "Missing fields")
		return
	}

	// --- Bắt đầu transaction MySQL ---
	tx, err := db.DB.Begin()
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "server_error", "Failed to start SQL transaction")
		return
	}

	// Đánh dấu để kiểm soát rollback thủ công
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Kiểm tra Gmail đã tồn tại chưa
	var exists int
	err = tx.QueryRow(`SELECT COUNT(*) FROM users WHERE gmail = ?`, req.Gmail).Scan(&exists)
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "server_error", "Database error")
		return
	}
	if exists > 0 {
		utils.SendError(c.Send, incoming.ID, "duplicate", "Gmail already registered")
		return
	}

	// Thêm user mới
	result, err := tx.Exec(`INSERT INTO users (gmail, username, password) VALUES (?, ?, ?)`, req.Gmail, req.Username, req.Password)
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "server_error", "Failed to create user")
		return
	}

	uid64, err := result.LastInsertId()
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "server_error", "Failed to get user ID")
		return
	}
	userID := int(uid64)

	// Thêm user_stats
	_, err = tx.Exec(`INSERT INTO user_stats (user_id, level, experience, gold, gems) VALUES (?, 1, 0, 500, 10)`, userID)
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "server_error", "Failed to insert user stats")
		return
	}

	// --- Thêm MongoDB trước khi commit MySQL ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userCards := bson.M{
		"user_id":     userID,
		"king_tower":  []bson.M{{"name": "King_Tower"}},
		"guard_tower": []bson.M{{"name": "Guard_Tower", "level": 1, "count": 0}},
		"cards": []bson.M{
			{"name": "Pawn", "level": 1, "count": 0},
			{"name": "Bishop", "level": 1, "count": 0},
			{"name": "Rook", "level": 1, "count": 0},
			{"name": "Knight", "level": 1, "count": 0},
			{"name": "Prince", "level": 1, "count": 0},
			{"name": "Queen", "level": 1, "count": 0},
			{"name": "Fireball", "level": 1, "count": 0},
			{"name": "Healing Light", "level": 1, "count": 0},
		},
	}

	userDeck := bson.M{
		"user_id":     userID,
		"king_tower":  bson.M{"name": "King_Tower"},
		"guard_tower": bson.M{"name": "Guard_Tower", "level": 1},
		"cards": []bson.M{
			{"index": 1, "name": "Pawn", "level": 1},
			{"index": 2, "name": "Bishop", "level": 1},
			{"index": 3, "name": "Rook", "level": 1},
			{"index": 4, "name": "Knight", "level": 1},
			{"index": 5, "name": "Prince", "level": 1},
			{"index": 6, "name": "Queen", "level": 1},
			{"index": 7, "name": "Fireball", "level": 1},
			{"index": 8, "name": "Healing Light", "level": 1},
		},
	}

	cardCol := db.MongoDatabase.Collection("user_cards")
	deckCol := db.MongoDatabase.Collection("user_decks")

	if _, err = cardCol.InsertOne(ctx, userCards); err != nil {
		utils.SendError(c.Send, incoming.ID, "mongo_error", "Failed to insert user cards")
		return
	}

	if _, err = deckCol.InsertOne(ctx, userDeck); err != nil {
		// Nếu insert deck fail, xóa userCards để giữ đồng bộ
		_, _ = cardCol.DeleteOne(ctx, bson.M{"user_id": userID})
		utils.SendError(c.Send, incoming.ID, "mongo_error", "Failed to insert user deck")
		return
	}

	// Nếu MongoDB thành công, commit MySQL transaction
	if err := tx.Commit(); err != nil {
		utils.SendError(c.Send, incoming.ID, "server_error", "Failed to commit SQL transaction")
		return
	}
	committed = true

	// Tạo token cho user mới
	token, err := auth.GenerateToken(userID, req.Gmail, req.Username)
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "token_error", "Failed to generate token")
		return
	}

	// Gán userID cho client
	c.User.ID = userID
	c.User.Frame = 2

	utils.SendMessage(c.Send, incoming.ID, "register_success", LoginResponse{
		Token:    token,
		Username: req.Username,
	})

}

func HandleGetUserCards(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	userCards, err := GetUserCardsByID(c.User.ID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.SendError(c.Send, incoming.ID, "not_found", "No user cards found")
		} else {
			utils.SendError(c.Send, incoming.ID, "mongo_error", "Failed to retrieve user cards")
		}
		return
	}

	utils.SendMessage(c.Send, incoming.ID, "user_cards", userCards)

}

func GetUserCardsByID(userID int) (*UserCards, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result UserCards
	err := db.MongoDatabase.Collection("user_cards").
		FindOne(ctx, bson.M{"user_id": userID}).
		Decode(&result)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func HandleGetUserDeck(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	userDeck, err := GetUserDeckByID(c.User.ID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.SendError(c.Send, incoming.ID, "not_found", "No user deck found")
		} else {
			utils.SendError(c.Send, incoming.ID, "mongo_error", "Failed to retrieve user deck")
		}
		return
	}

	utils.SendMessage(c.Send, incoming.ID, "user_deck", userDeck)

}

func GetUserDeckByID(userID int) (*UserDeck, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result UserDeck
	err := db.MongoDatabase.Collection("user_decks").
		FindOne(ctx, bson.M{"user_id": userID}).
		Decode(&result)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func HandleSwapCard(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	// Đọc yêu cầu từ client
	var req struct {
		CardName  string `json:"card_name"`  // Thẻ mới muốn thay vào
		SlotIndex int    `json:"slot_index"` // Vị trí trong deck (1–8)
	}
	if err := json.Unmarshal(incoming.Data, &req); err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_payload", "Invalid swap card format")
		return
	}

	if req.SlotIndex < 1 || req.SlotIndex > 8 {
		utils.SendError(c.Send, incoming.ID, "invalid_slot", "Invalid slot index (must be from 1 to 8)")
		return
	}

	// Lấy dữ liệu deck hiện tại
	userDeck, err := GetUserDeckByID(c.User.ID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.SendError(c.Send, incoming.ID, "not_found", "No user deck found")
		} else {
			utils.SendError(c.Send, incoming.ID, "mongo_error", "Failed to retrieve user deck")
		}
		return
	}

	// Lấy danh sách thẻ người dùng sở hữu
	userCards, err := GetUserCardsByID(c.User.ID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.SendError(c.Send, incoming.ID, "not_found", "No user cards found")
		} else {
			utils.SendError(c.Send, incoming.ID, "mongo_error", "Failed to retrieve user cards")
		}
		return
	}

	// Kiểm tra người dùng có sở hữu thẻ mới không
	var foundCard *CardInfo
	for _, card := range userCards.Cards {
		if card.Name == req.CardName {
			foundCard = &card
			break
		}
	}
	if foundCard == nil {
		utils.SendError(c.Send, incoming.ID, "card_not_found", "Card not found in user cards")
		return
	}

	// Kiểm tra thẻ đã tồn tại trong deck hay chưa (ngoại trừ vị trí đang muốn thay)
	for i, card := range userDeck.Cards {
		if i != req.SlotIndex-1 && card.Name == req.CardName {
			utils.SendError(c.Send, incoming.ID, "duplicate_card", "Card already exists in deck")
			return
		}
	}

	// (Ghi log nếu thẻ cũ trong slot không nằm trong userCards — để phòng trường hợp lỗi dữ liệu)
	oldCard := userDeck.Cards[req.SlotIndex-1]

	// Kiểm tra thẻ mới có giống thẻ cũ ở vị trí đó không
	if oldCard.Name == req.CardName {
		utils.SendError(c.Send, incoming.ID, "no_change", "New card is the same as the current card in the slot")
		return
	}

	hasOldCard := false
	for _, card := range userCards.Cards {
		if card.Name == oldCard.Name {
			hasOldCard = true
			break
		}
	}
	if !hasOldCard {
		log.Printf("[SECURITY WARNING] User %d tried to replace card '%s' from deck they may not own", c.User.ID, oldCard.Name)
	}

	// Cập nhật vị trí trong deck với thẻ mới
	userDeck.Cards[req.SlotIndex-1] = DeckCard{
		Index: req.SlotIndex,
		Name:  foundCard.Name,
		Level: foundCard.Level,
	}

	// Ghi lại vào MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deckCol := db.MongoDatabase.Collection("user_decks")
	_, err = deckCol.ReplaceOne(ctx, bson.M{"user_id": c.User.ID}, userDeck)
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "mongo_error", "Failed to update user deck")
		return
	}

	// Phản hồi thành công
	utils.SendMessage(c.Send, incoming.ID, "swap_card_success", userDeck)

}

func HandleCreateLobby(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	var req LobbyRequest
	if err := json.Unmarshal(incoming.Data, &req); err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_data", "Invalid lobby create data")
		return
	}

	lobbyID := uuid.NewString()

	room := utils.CreateLobbyRoom(lobbyID, req.RoomType, false)

	joinedRoom, success, slotIndex := utils.JoinLobbyRoom(lobbyID, c)
	if !success {
		utils.SendError(c.Send, incoming.ID, "join_failed", "Could not join lobby")
		return
	}

	utils.SendMessage(c.Send, incoming.ID, "lobby_created", map[string]any{
		"lobby_id": room.ID,
		"type":     string(joinedRoom.Type),
		"slot":     slotIndex,
	})
}

func HandleJoinLobby(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	var req LobbyRequest
	if err := json.Unmarshal(incoming.Data, &req); err != nil || req.LobbyID == "" {
		utils.SendError(c.Send, incoming.ID, "invalid_data", "Missing or invalid lobby ID")
		return
	}

	joinedRoom, success, slotIndex := utils.JoinLobbyRoom(req.LobbyID, c)
	if !success {
		utils.SendError(c.Send, incoming.ID, "join_failed", "Could not join lobby")
		return
	}

	utils.SendMessage(c.Send, incoming.ID, "lobby_joined", map[string]any{
		"lobby_id": req.LobbyID,
		"type":     string(joinedRoom.Type),
		"slot":     slotIndex,
	})
}

func HandleMatchLobby(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	var req LobbyRequest
	if err := json.Unmarshal(incoming.Data, &req); err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_data", "Invalid match request")
		return
	}

	room := utils.FindAvailableLobby(req.RoomType)
	if room == nil {
		roomID := uuid.NewString()
		// Tạo phòng mới với match = true
		room = utils.CreateLobbyRoom(roomID, req.RoomType, true)
	}

	joinedRoom, success, slotIndex := utils.JoinLobbyRoom(room.ID, c)
	if !success {
		utils.SendError(c.Send, incoming.ID, "join_failed", "Could not join lobby")
		return
	}

	utils.SendMessage(c.Send, incoming.ID, "matched_lobby", map[string]any{
		"lobby_id": room.ID,
		"type":     string(joinedRoom.Type),
		"slot":     slotIndex,
	})
}

func HandleLeaveLobby(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	var req LobbyRequest
	if err := json.Unmarshal(incoming.Data, &req); err != nil || req.LobbyID == "" {
		utils.SendError(c.Send, incoming.ID, "invalid_data", "Missing or invalid lobby ID")
		return
	}

	if !utils.LeaveLobbyRoom(req.LobbyID, c) {
		utils.SendError(c.Send, incoming.ID, "leave_failed", "Could not leave lobby")
		return
	}

	c.RoomType = ""
	c.SlotIndex = -1

	utils.SendMessage(c.Send, incoming.ID, "lobby_left", map[string]string{
		"lobby_id": req.LobbyID,
	})
}

func HandleReleaseCard(c *types.Client, incoming utils.IncomingMessage) {
	if c.User.ID == 0 {
		utils.SendError(c.Send, incoming.ID, "unauthorized", "User not logged in")
		return
	}

	var req struct {
		CardID int `json:"card_id"`
		X      int `json:"x"`
		Y      int `json:"y"`
	}

	if err := json.Unmarshal(incoming.Data, &req); err != nil {
		utils.SendError(c.Send, incoming.ID, "invalid_payload", "Invalid release card format")
		return
	}

	if req.CardID < 0 || req.CardID > 7 {
		utils.SendError(c.Send, incoming.ID, "missing_fields", "Card name missing")
		return
	}

	// Gói dữ liệu gửi đi
	action := map[string]interface{}{
		"type": "release",
		"data": map[string]interface{}{
			"msg_id":  incoming.ID,
			"user_id": c.User.ID,
			"cardID":  req.CardID,
			"x":       req.X,
			"y":       req.Y,
		},
	}

	data, err := json.Marshal(action)
	if err != nil {
		utils.SendError(c.Send, incoming.ID, "internal_error", "Failed to encode action")
		return
	}

	select {
	case c.User.MatchRoom <- data:
		return
	default:
		utils.SendError(c.Send, incoming.ID, "match_room_busy", "Unable to send to match room")
	}
}
