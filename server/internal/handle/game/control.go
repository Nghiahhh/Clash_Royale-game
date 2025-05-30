package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"math/rand"
	"time"

	"server/internal/db"
	"server/internal/session"
	"server/internal/types"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type outgoingMessage struct {
	ID   string      `json:"id"`
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ControlLobby manages a single lobby room, checking every second if conditions are met
func ControlLobby(ctx context.Context, room *session.LobbyRoom) {
	log.Printf("Lobby control started for room %s", room.ID)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := 600 * time.Second
	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			clientCount := ActiveClientsCount(room)

			if clientCount == room.MaxSize {
				log.Printf("Lobby %s is full. Moving to match.", room.ID)
				if PromoteLobbyToMatch(room) {
					return
				}
			} else if clientCount > room.MaxSize {
				log.Printf("Lobby %s has more clients than allowed. Closing.", room.ID)
				removeLobbyRoom(room.ID, "room_closed", "Lobby has too many clients")
				return
			}

			if time.Since(startTime) > timeout {
				log.Printf("Lobby %s timeout reached. Closing.", room.ID)
				removeLobbyRoom(room.ID, "timeout", "Lobby timeout reached")
				return
			}

		case <-ctx.Done():
			log.Printf("Lobby %s canceled via context. Stopping control.", room.ID)
			removeLobbyRoom(room.ID, "canceled", "Lobby was canceled")
			return
		}
	}
}

// ActiveClientsCount returns the number of connected and alive clients in the room
func ActiveClientsCount(room *session.LobbyRoom) int {
	count := 0
	for _, slot := range room.Slots {
		if slot.Client != nil && isClientAlive(slot.Client) {
			count++
		}
	}
	return count
}

// isClientAlive sends a ping to the client and checks if the connection is still alive
func isClientAlive(c *types.Client) bool {
	if c.Conn == nil {
		return false
	}
	err := c.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(1*time.Second))
	return err == nil
}

// removeLobbyRoom cleans up a lobby and notifies all clients with an error message
func removeLobbyRoom(id string, errorType string, errorMsg string) {
	session.LobbyMu.Lock()
	defer session.LobbyMu.Unlock()

	room, ok := session.Lobbies[id]
	if ok {
		log.Printf("Removing lobby %s: %s", id, errorMsg)
		for _, slot := range room.Slots {
			if slot.Client != nil {
				sendError(slot.Client.Send, "", errorType, errorMsg)
			}
		}
		delete(session.Lobbies, id)
	} else {
		log.Printf("Attempted to remove non-existent lobby %s", id)
	}
}

// sendJSON marshals data to JSON and sends to the send channel
func sendJSON(send chan<- []byte, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	// Safe send to avoid panic if channel closed
	select {
	case send <- jsonData:
	default:
		log.Printf("Send channel is full or closed, dropping message.")
	}
}

// sendError sends a standard error message to the client
func sendError(send chan<- []byte, id, errorType, message string) {
	if id == "" {
		id = "unknown"
	}
	sendJSON(send, outgoingMessage{
		ID:   id,
		Type: "error",
		Data: map[string]string{
			"error":   errorType,
			"message": message,
		},
	})
}

// sendMessage sends a typed message to the client
func sendMessage(send chan<- []byte, id string, typ string, data interface{}) {
	msg := outgoingMessage{
		ID:   id,
		Type: typ,
		Data: data,
	}
	sendJSON(send, msg)
}

func PromoteLobbyToMatch(room *session.LobbyRoom) bool {
	// Lấy danh sách clients alive từ Slots và chuyển thành User
	var users []*session.User
	for _, slot := range room.Slots {
		if slot.Client != nil {
			users = append(users, &session.User{
				ID:     slot.Client.User.ID,
				Client: slot.Client,
				// dataGame để mặc định hoặc khởi tạo nếu cần
			})
		}
	}

	if len(users) < room.MaxSize {
		log.Printf("Lobby %s does not have enough clients to start a match.", room.ID)
		return false
	}

	// Xóa phòng lobby khỏi map
	session.LobbyMu.Lock()
	delete(session.Lobbies, room.ID)
	session.LobbyMu.Unlock()

	// Tạo MatchRoom mới với danh sách User
	match := &session.MatchRoom{
		ID:      room.ID,
		MaxSize: room.MaxSize,
		Type:    room.Type,
		User:    users,
	}

	// Thêm vào danh sách MatchRooms
	session.MatchesMu.Lock()
	session.Matches[room.ID] = match
	session.MatchesMu.Unlock()

	go gameStart(match)

	return true
}

type UserDeck struct {
	UserID    int `bson:"user_id"`
	KingTower struct {
		Name  string `bson:"name"`
		Level int    `bson:"level,omitempty"`
	} `bson:"king_tower"`
	GuardTower struct {
		Name  string `bson:"name"`
		Level int    `bson:"level,omitempty"`
	} `bson:"guard_tower"`
	Cards [8]struct {
		Index int    `bson:"index"`
		Name  string `bson:"name"`
		Level int    `bson:"level"`
	} `bson:"cards"`
}

type ReleaseActionData struct {
	MsgID  string `json:"msg_id"`
	UserID int    `json:"user_id"`
	CardID int    `json:"card"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

func gameStart(match *session.MatchRoom) {
	for _, user := range match.User {
		err := getUserDeck(db.MongoDatabase, user)
		if err != nil {
			log.Printf("Error getting user deck for user %d: %v", user.ID, err)
			sendError(user.Client.Send, "", "error", "Failed to load user deck")
			return
		}
	}

	log.Printf("Lobby %s promoted to MatchRoom.", match.ID)

	for _, user := range match.User {
		if user.Client != nil && user.Client.Send != nil {
			startData := map[string]interface{}{
				"roomID": match.ID,
				"type":   match.Type,
			}
			sendMessage(user.Client.Send, "start game", "start game", startData)
		}
	}

	matchRoom := make(chan []byte, 40)
	for _, client := range match.User {
		client.Client.User.MatchRoom = matchRoom
	}

	// ⚔️ Initialize game state
	gameState := NewGameState(match)

	SendDeckToAllClients(gameState)

	// ⏳ Thời gian trận đấu tối đa, ví dụ 3 phút
	gameTimer := time.NewTimer(3 * time.Minute)
	ticker := time.NewTicker(time.Duration(gameState.Tick) * 10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go updateGameState(gameState)

		case data := <-matchRoom:
			var action map[string]interface{}
			if err := json.Unmarshal(data, &action); err != nil {
				log.Printf("Error unmarshaling action: %v", err)
				continue
			}

			actionType, ok := action["type"].(string)
			if !ok {
				log.Printf("Invalid action type: %v", action["type"])
				continue
			}

			switch actionType {
			case "release":
				dataMap, ok := action["data"].(map[string]interface{})
				if !ok {
					log.Printf("Invalid data format for release action: %v", action["data"])
					continue
				}

				dataBytes, err := json.Marshal(dataMap)
				if err != nil {
					log.Printf("Error marshaling data for release action: %v", err)
					continue
				}

				var releaseData ReleaseActionData
				if err := json.Unmarshal(dataBytes, &releaseData); err != nil {
					log.Printf("Error unmarshaling release action data: %v", err)
					continue
				}

				var player PlayerState
				var playerIndex int
				var playerSubIndex int
				top := false
				found := false
				for i, group := range gameState.Players {
					for j, p := range group {
						if p.User.ID == releaseData.UserID {
							player = *p
							playerIndex = i
							playerSubIndex = j
							if i == 0 {
								top = true
							}
							found = true
							break
						}
					}
					if found {
						break
					}
				}

				if !found || player.User == nil {
					log.Printf("Player with UserID %d not found in game state", releaseData.UserID)
					continue
				}

				// Kiểm tra bài trong tay
				for _, id := range player.Hand {
					if id == releaseData.CardID {
						released := false

						// Troops
						for _, card := range player.User.DataGame.Troops {
							if card.Index == releaseData.CardID {
								if releaseData.Y < 0 || releaseData.Y >= len(gameState.Map) ||
									releaseData.X < 0 || releaseData.X >= len(gameState.Map[0])/2 {
									sendError(player.User.Client.Send, releaseData.MsgID, "invalid_position", "Out of map bounds")
									break
								}

								validTiles := map[int]bool{1: true}
								tileValue := gameState.Map[releaseData.Y][releaseData.X]
								if !validTiles[tileValue] {
									sendError(player.User.Client.Send, releaseData.MsgID, "invalid_position", "Invalid tile type for card release")
									break
								}

								if player.Elixir < float64(card.Info.Mana) {
									sendError(player.User.Client.Send, releaseData.MsgID, "not_enough_elixir", "Not enough elixir to release card")
									break
								}

								player.Elixir -= float64(card.Info.Mana)

								x, y := releaseData.X, releaseData.Y
								if !top {
									x, y = MirrorPosition(x, y, len(gameState.Map), len(gameState.Map[0]))
								}

								troop := Troop{
									HP:          card.Info.Hp,
									Time_attack: 0,
									Shield:      card.Info.Shield,
									Location:    Position{X: x, Y: y, long: 1, wide: 1},
									Skill_using: false,
									CardInfo:    card,
									Skill_info:  card.Info.Skill,
									TargetID:    "",
								}

								allies := Allies{
									ID:     uuid.New().String(),
									Type:   "troop",
									Alive:  true,
									Troops: troop,
								}

								gameState.Allies[player.Side] = append(gameState.Allies[player.Side], allies)

								UpdateHandAfterPlay(&player, releaseData.CardID)
								SendDeckToClients(&player, releaseData.MsgID)
								released = true
								break
							}
						}

						// Spells
						if !released {
							for _, card := range player.User.DataGame.Spells {
								if card.Index == releaseData.CardID {
									if releaseData.Y < 0 || releaseData.Y >= len(gameState.Map) ||
										releaseData.X < 0 || releaseData.X >= len(gameState.Map[0]) {
										sendError(player.User.Client.Send, releaseData.MsgID, "invalid_position", "Out of map bounds")
										break
									}

									validTiles := map[int]bool{1: true, 3: true, 4: true}
									tileValue := gameState.Map[releaseData.Y][releaseData.X]
									if !validTiles[tileValue] {
										sendError(player.User.Client.Send, releaseData.MsgID, "invalid_position", "Invalid tile type for card release")
										break
									}

									if player.Elixir < float64(card.Info.Mana) {
										sendError(player.User.Client.Send, releaseData.MsgID, "not_enough_elixir", "Not enough elixir to release card")
										break
									}

									player.Elixir -= float64(card.Info.Mana)

									x, y := releaseData.X, releaseData.Y
									if !top {
										x, y = MirrorPosition(x, y, len(gameState.Map), len(gameState.Map[0]))
									}

									troop := Troop{
										HP:          card.Info.Hp,
										Time_attack: 0,
										Shield:      card.Info.Shield,
										Location:    Position{X: x, Y: y, long: 1, wide: 1},
										Skill_using: false,
										CardInfo:    card,
										Skill_info:  card.Info.Skill,
										TargetID:    "",
									}

									allies := Allies{
										ID:     uuid.New().String(),
										Type:   "troop",
										Alive:  true,
										Troops: troop,
									}

									gameState.Allies[player.Side] = append(gameState.Allies[player.Side], allies)

									UpdateHandAfterPlay(&player, releaseData.CardID)
									SendDeckToClients(&player, releaseData.MsgID)
									break
								}
							}
						}

						// Cập nhật player sau khi thay đổi elixir
						gameState.Players[playerIndex][playerSubIndex] = &player
						break
					}
				}

			default:

			}

		case <-gameTimer.C:
			log.Printf("Match %s timed out. Evaluating final result.", match.ID)

			// Đã check king chết trong checkGameEnd(), giờ gọi resolveDrawOutcome
			winner := resolveDrawOutcome(gameState)

			for side := 0; side < 2; side++ {
				for _, player := range gameState.Players[side] {
					result := "lose"
					if winner == -1 {
						result = "draw"
					} else if player.Side == winner {
						result = "win"
					}
					UpdateUserRewards(player.User.ID, result)
					sendMessage(player.User.Client.Send, "end_game", "game_end", map[string]interface{}{
						"result": result,
					})
				}
			}
			return

		}
	}
}

func resolveDrawOutcome(gs *GameState) int {
	guardCount := [2]int{}
	minHP := [2]int{1<<31 - 1, 1<<31 - 1} // max int value

	for side := 0; side < 2; side++ {
		for _, ally := range gs.Allies[side] {
			hp := 0

			switch ally.Type {
			case "guard_tower":
				hp = ally.Guard.HP
				guardCount[side]++
			case "king_tower":
				hp = ally.King.HP
			default:
				continue
			}

			if hp < minHP[side] {
				minHP[side] = hp
			}
		}
	}

	// Bên nào có nhiều trụ bảo vệ hơn thì thắng
	if guardCount[0] > guardCount[1] {
		return 0
	} else if guardCount[1] > guardCount[0] {
		return 1
	}

	// Nếu số trụ bảo vệ bằng nhau → xét máu thấp nhất
	if minHP[0] < minHP[1] {
		return 1 // bên 0 có máu thấp hơn → thua
	} else if minHP[1] < minHP[0] {
		return 0
	}

	return -1 // hoà
}

func updateGameState(gs *GameState) {
	// 1. Cập nhật tài nguyên Elixir cho mỗi người chơi
	updateElixir(gs)

	// 3. Xử lý combat giữa các entity (Troop vs Troop, Guard vs Troop, ...)
	handleCombat(gs)

	// 5. cập nhật alive cho các entity
	UpdateAliveStatus(gs)

	// 4. Cleanup các entity đã chết (HP <= 0 hoặc hết thời gian tồn tại)
	CleanupAllies(gs)

	// 6. Gửi sự kiện cập nhật trạng thái đến Client
	emitGameStateEvents(gs)

	// 5. Kiểm tra điều kiện kết thúc trận đấu (King Tower bị phá)
	winner := checkGameEnd(gs)
	if winner != -1 {
		// Gửi sự kiện kết thúc trận đấu cho tất cả client
		for side := 0; side < 2; side++ {
			for _, player := range gs.Players[side] {
				result := "lose"
				if player.Side == winner {
					result = "win"
				}
				UpdateUserRewards(player.User.ID, result)
				sendMessage(player.User.Client.Send, "end_game", "game_end", map[string]interface{}{
					"result": result,
				})
			}
		}
		return
	}
}

func UpdateUserRewards(userID int, result string) error {
	var expGain, gold, gems int

	switch result {
	case "win":
		expGain, gold, gems = 30, 200, 1
	case "draw":
		expGain, gold, gems = 10, 50, 0
	case "lose":
		expGain, gold, gems = 5, 5, 0
	default:
		return nil
	}

	// 1. Cộng reward ban đầu
	_, err := db.DB.Exec(`
		UPDATE user_stats
		SET experience = experience + ?, gold = gold + ?, gems = gems + ?
		WHERE user_id = ?
	`, expGain, gold, gems, userID)
	if err != nil {
		log.Printf("❌ Error updating rewards: %v", err)
		return err
	}

	// 2. Lấy lại experience và level hiện tại
	var experience, level int
	err = db.DB.QueryRow(`
		SELECT experience, level
		FROM user_stats
		WHERE user_id = ?
	`, userID).Scan(&experience, &level)
	if err != nil {
		log.Printf("❌ Error getting current stats: %v", err)
		return err
	}

	// 3. Truy MongoDB để lấy requirement
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("user_level_config")

	var levelConfig struct {
		Level              int `bson:"level"`
		ExperienceRequired int `bson:"experience_required"`
	}

	err = collection.FindOne(ctx, bson.M{"level": level}).Decode(&levelConfig)
	if err != nil {
		return nil
	}

	// 4. Kiểm tra có đủ để lên cấp không
	if experience >= levelConfig.ExperienceRequired {
		newLevel := level + 1
		newExperience := experience - levelConfig.ExperienceRequired

		_, err = db.DB.Exec(`
			UPDATE user_stats
			SET level = ?, experience = ?
			WHERE user_id = ?
		`, newLevel, newExperience, userID)
		if err != nil {
			log.Printf("❌ Error updating level up: %v", err)
			return err
		}
	}

	return nil
}

func UpdateAliveStatus(gs *GameState) {
	for side := 0; side < 2; side++ {
		for i := range gs.Allies[side] {
			unit := &gs.Allies[side][i]

			// Troop
			if unit.Type == "troop" {
				if unit.Troops.HP <= 0 {
					unit.Alive = false
				}
			}

			// Guard
			if unit.Type == "guard" {
				if unit.Guard.HP <= 0 {
					unit.Alive = false
				}
			}

			// King
			if unit.Type == "king" {
				if unit.King.HP <= 0 {
					unit.Alive = false
				}
			}
		}
	}
}

// Kiểm tra King Tower của mỗi bên, nếu bị phá thì trả về side thắng, chưa kết thúc trả về -1
func checkGameEnd(gs *GameState) int {
	for side := 0; side < 2; side++ {
		for _, ally := range gs.Allies[side] {
			if ally.Type == "king_tower" && ally.King.HP <= 0 {
				return 1 - side // Đối phương thắng
			}
		}
	}
	return -1
}

func CreateUpdateEvent(gs *GameState, side int) map[string]interface{} {
	// Sao chép map
	var displayMap [][]int
	for _, row := range gs.Map {
		newRow := append([]int{}, row...)
		displayMap = append(displayMap, newRow)
	}

	// Nếu là side 1, đảo map qua tâm (xoay 180 độ)
	if side == 1 {
		// Đảo hàng (top-down)
		for i := 0; i < len(displayMap)/2; i++ {
			displayMap[i], displayMap[len(displayMap)-1-i] = displayMap[len(displayMap)-1-i], displayMap[i]
		}
		// Đảo cột (left-right)
		for i := 0; i < len(displayMap); i++ {
			for j := 0; j < len(displayMap[i])/2; j++ {
				displayMap[i][j], displayMap[i][len(displayMap[i])-1-j] = displayMap[i][len(displayMap[i])-1-j], displayMap[i][j]
			}
		}
	}

	// Mirror Allies nếu side = 1
	displayAllies := [2][]Allies{}
	for i := 0; i < 2; i++ {
		for _, ally := range gs.Allies[i] {
			clone := ally // sao chép tránh modify gốc
			if side == 1 {
				switch ally.Type {
				case "troop":
					clone.Troops.Location.X, clone.Troops.Location.Y =
						MirrorPosition(ally.Troops.Location.X, ally.Troops.Location.Y, len(gs.Map), len(gs.Map[0]))
				case "spell":
					clone.Spells.Location.X, clone.Spells.Location.Y =
						MirrorPosition(ally.Spells.Location.X, ally.Spells.Location.Y, len(gs.Map), len(gs.Map[0]))
				case "guard_tower":
					clone.Guard.Location.X, clone.Guard.Location.Y =
						MirrorPosition(ally.Guard.Location.X, ally.Guard.Location.Y, len(gs.Map), len(gs.Map[0]))
				case "king_tower":
					clone.King.Location.X, clone.King.Location.Y =
						MirrorPosition(ally.King.Location.X, ally.King.Location.Y, len(gs.Map), len(gs.Map[0]))
				}
			}
			displayAllies[i] = append(displayAllies[i], clone)
		}
	}

	player := gs.Players[side][0]
	return map[string]interface{}{
		"map":      displayMap,
		"allies":   displayAllies,
		"elixir":   player.Elixir,
		"hand":     player.Hand,
		"nextCard": player.NextCard,
	}
}

// Gửi event cho client
func SendToClient(user *session.User, event map[string]interface{}) {
	sendMessage(user.Client.Send, "update", "update", event)
}

func updateElixir(gs *GameState) {
	for side := 0; side < 2; side++ {
		for _, player := range gs.Players[side] {
			if player.Elixir < 10 {
				player.ElixirTimer += float64(gs.Tick) / 1000 // Tăng theo 30 tick = 1 giây
				if player.ElixirTimer >= 1 {
					player.Elixir += 1
					player.ElixirTimer -= 1
				}
			} else {
				player.ElixirTimer = 0
			}
		}
	}
}

func handleCombat(gs *GameState) {
	for side := 0; side < 2; side++ {
		enemySide := 1 - side

		for i := range gs.Allies[side] {
			attacker := &gs.Allies[side][i]
			if !attacker.Alive {
				continue
			}

			switch attacker.Type {
			case "troop":
				handleTroopCombat(gs, attacker, enemySide)
			case "guard_tower":
				handleGuardCombat(gs, attacker, enemySide)
			case "king_tower":
				handleKingCombat(gs, attacker, enemySide)
			case "spell":
				handleSpellEffect(gs, attacker)
			}
		}
	}
}

func (a *Allies) GetLocation() Position {
	switch a.Type {
	case "troop":
		return a.Troops.Location
	case "guard_tower":
		return a.Guard.Location
	case "king_tower":
		return a.King.Location
	case "spell":
		return a.Spells.Location
	default:
		return Position{-1, -1, 0, 0} // vị trí không hợp lệ
	}
}

type Node struct {
	Pos  Position
	Prev *Node
}

// Tọa độ trên bản đồ
// Hàm tìm đường từ start đến goal (dùng Position thay cho Coord)
func bfsPath(gs *GameState, start, goal Position) []Position {
	h, w := len(gs.Map), len(gs.Map[0])
	visited := make([][]bool, h)
	for i := range visited {
		visited[i] = make([]bool, w)
	}

	// 4 hướng: lên, xuống, trái, phải
	dirs := []struct{ X, Y int }{
		{X: -1, Y: 0},
		{X: 1, Y: 0},
		{X: 0, Y: -1},
		{X: 0, Y: 1},
	}

	type Node struct {
		Pos  Position
		Prev *Node
	}

	queue := []Node{{Pos: start}}
	visited[start.X][start.Y] = true
	var endNode *Node

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.Pos.X == goal.X && current.Pos.Y == goal.Y {
			nodeCopy := current
			endNode = &nodeCopy
			break
		}

		for _, d := range dirs {
			nx := current.Pos.X + d.X
			ny := current.Pos.Y + d.Y
			if nx >= 0 && nx < h && ny >= 0 && ny < w &&
				!visited[nx][ny] && gs.Map[nx][ny] == 1 {
				visited[nx][ny] = true
				next := Node{Pos: Position{X: nx, Y: ny}, Prev: &current}
				queue = append(queue, next)
			}
		}
	}

	if endNode == nil {
		return nil // không tìm được đường
	}

	// Truy vết đường đi từ goal về start
	var path []Position
	for node := endNode; node != nil; node = node.Prev {
		path = append([]Position{node.Pos}, path...)
	}

	return path
}

func moveTowards(gs *GameState, attacker *Allies, targetID string) {
	if targetID == "" {
		return
	}
	target := getAllyByID(gs, targetID)
	if target == nil || !target.Alive {
		return
	}

	from := attacker.GetLocation()
	to := target.GetLocation()

	log.Printf("Moving %s from %v to %v", attacker.Type, from, to)
	path := bfsPath(gs, from, to)
	log.Printf("Path found: %v", path)
	if len(path) >= 2 {
		// path[0] là vị trí hiện tại, path[1] là bước kế tiếp
		nextStep := Position{
			X:    path[1].X,
			Y:    path[1].Y,
			long: attacker.GetLocation().long,
			wide: attacker.GetLocation().wide,
		}
		attacker.SetLocation(nextStep)
	}
}

func (a *Allies) SetLocation(pos Position) {
	switch a.Type {
	case "troop":
		a.Troops.Location = pos
	case "guard_tower":
		a.Guard.Location = pos
	case "king_tower":
		a.King.Location = pos
	}
}

func handleTroopCombat(gs *GameState, troop *Allies, enemySide int) {
	t := &troop.Troops

	// Skip nếu đã chết
	if t.HP <= 0 {
		return
	}

	// Kiểm tra mục tiêu có trong tầm không
	if t.TargetID == "" || !isInRange(t.Location, getLocationByID(gs, t.TargetID), t.CardInfo.Info.Range) {
		t.TargetID = findNearestTargetID(gs, t.Location, enemySide, t.CardInfo.Info.Range)
		t.Time_attack = 0 // reset nếu ngoài tầm hoặc không có target
		moveTowards(gs, troop, t.TargetID)
	} else {
		// Chỉ cộng thời gian nếu trong tầm
		t.Time_attack += 0.5
		if t.Time_attack >= float32(1.0/t.CardInfo.Info.AttackSpeed) {
			target := getAllyByID(gs, t.TargetID)
			if target != nil && target.Alive {
				damage := calculateDamage(troop, target)
				target.ReduceHP(damage)
				log.Println("Attacking target:", target.ID, "with damage:", damage)
				t.Time_attack -= float32(1.0 / t.CardInfo.Info.AttackSpeed)

				if t.Skill_info.Name != "" && !t.Skill_using {
					t.Skill_using = true
					applySkillEffect(gs, troop)
				}
			}
		}
	}

}

func handleGuardCombat(gs *GameState, guard *Allies, enemySide int) {
	g := &guard.Guard

	if g.HP <= 0 {
		return
	}

	if g.TargetID == "" || !isInRange(g.Location, getLocationByID(gs, g.TargetID), g.GuardInfo.Info.Range) {
		g.TargetID = findNearestTargetID(gs, g.Location, enemySide, g.GuardInfo.Info.Range)
		g.Time_attack = 0 // reset thời gian tấn công nếu target mới hoặc ngoài tầm
	} else {
		g.Time_attack += 0.5
		if g.Time_attack >= float32(1.0/g.GuardInfo.Info.AttackSpeed) {
			target := getAllyByID(gs, g.TargetID)
			if target != nil && target.IsAlive() {
				damage := calculateDamage(guard, target)
				log.Println("Attacking target:", target.ID, "with damage:", damage)
				target.ReduceHP(damage)
				g.Time_attack -= float32(1.0 / g.GuardInfo.Info.AttackSpeed)
			}
		}
	}
}

func handleKingCombat(gs *GameState, king *Allies, enemySide int) {
	k := &king.King

	if !k.Active || k.HP <= 0 {
		return
	}

	if k.TargetID == "" || !isInRange(k.Location, getLocationByID(gs, k.TargetID), k.KingInfo.Info.Range) {
		k.TargetID = findNearestTargetID(gs, k.Location, enemySide, k.KingInfo.Info.Range)
		k.Time_attack = 0 // reset thời gian tấn công nếu target mới hoặc ngoài tầm
	} else {
		k.Time_attack += 0.5
		if k.Time_attack >= float32(1.0/k.KingInfo.Info.AttackSpeed) {
			target := getAllyByID(gs, k.TargetID)
			if target != nil && target.IsAlive() {
				damage := calculateDamage(king, target)
				log.Println("Attacking target:", target.ID, "with damage:", damage)
				target.ReduceHP(damage)
				k.Time_attack -= float32(1.0 / k.KingInfo.Info.AttackSpeed)
			}
		}
	}
}

func handleSpellEffect(gs *GameState, spell *Allies) {
	s := &spell.Spells
	s.Time_effect += 0.5
	if s.Time_effect >= 1.0/float32(s.Skill_info.Effect_speed) {
		if s.Time < s.Skill_info.Time {
			log.Println("Applying spell effect at location:", s.Location, "with skill:", s.Skill_info.Name)
			ApplySkillArea(gs, s.Location, &s.Skill_info)
			s.Time_effect -= 1.0 / float32(s.Skill_info.Effect_speed)
			s.Time += 1
		}
		spell.Alive = false
	}
}

func isInRange(from Position, to Position, rangeVal float64) bool {
	dx := float64(from.X - to.X)
	dy := float64(from.Y - to.Y)
	return math.Sqrt(dx*dx+dy*dy) <= rangeVal
}

func getLocationByID(gs *GameState, id string) Position {
	for side := 0; side < 2; side++ {
		for i := range gs.Allies[side] {
			if gs.Allies[side][i].ID == id {
				return gs.Allies[side][i].GetLocation()
			}
		}
	}
	return Position{-1, -1, 0, 0}
}

func findNearestTargetID(gs *GameState, from Position, enemySide int, rangeVal float64) string {
	minDist := math.MaxFloat64
	nearestID := ""
	for i := range gs.Allies[enemySide] {
		target := &gs.Allies[enemySide][i]
		if !target.Alive || target.Type == "spell" {
			continue
		}
		dist := GetMinDistanceBetweenAreas(from, target.GetLocation(), enemySide)
		if dist < minDist {
			minDist = dist
			nearestID = target.ID
		}
	}
	return nearestID
}

func GetMinDistanceBetweenAreas(from, to Position, enemySide int) float64 {
	minDist := math.MaxFloat64

	for dy1 := 0; dy1 < from.wide; dy1++ {
		for dx1 := 0; dx1 < from.long; dx1++ {
			var x1, y1 int
			if enemySide == 0 {
				x1 = from.X + dx1
				y1 = from.Y + dy1
			} else {
				x1 = from.X - dx1
				y1 = from.Y - dy1
			}

			for dy2 := 0; dy2 < to.wide; dy2++ {
				for dx2 := 0; dx2 < to.long; dx2++ {
					var x2, y2 int
					if enemySide == 0 {
						x2 = to.X + dx2
						y2 = to.Y + dy2
					} else {
						x2 = to.X - dx2
						y2 = to.Y - dy2
					}

					dist := GetDistance(
						Position{X: x1, Y: y1},
						Position{X: x2, Y: y2},
					)

					if dist < minDist {
						minDist = dist
					}
				}
			}
		}
	}

	return minDist
}

func GetDistance(from, to Position) float64 {
	dx := float64(from.X - to.X)
	dy := float64(from.Y - to.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func getAllyByID(gs *GameState, id string) *Allies {
	for side := 0; side < 2; side++ {
		for i := range gs.Allies[side] {
			if gs.Allies[side][i].ID == id {
				return &gs.Allies[side][i]
			}
		}
	}
	return nil
}

func calculateDamage(attacker *Allies, defender *Allies) int {
	var atk, def int
	var critRate float64
	var shield *int

	// Lấy thông tin ATK và CRIT của attacker
	switch attacker.Type {
	case "troop":
		atk = attacker.Troops.CardInfo.Info.Atk
		critRate = attacker.Troops.CardInfo.Info.CritRate
	case "guard_tower":
		atk = attacker.Guard.GuardInfo.Info.Atk
		critRate = attacker.Guard.GuardInfo.Info.CritRate
	case "king_tower":
		atk = attacker.King.KingInfo.Info.Atk
		critRate = attacker.King.KingInfo.Info.CritRate
	}

	// Lấy thông tin DEF và SHIELD của defender
	switch defender.Type {
	case "troop":
		def = defender.Troops.CardInfo.Info.Def
		shield = &defender.Troops.Shield
	case "guard_tower":
		def = defender.Guard.GuardInfo.Info.Def
		shield = &defender.Guard.Shield
	case "king_tower":
		def = defender.King.KingInfo.Info.Def
		shield = &defender.King.Shield
	}

	// 1. Tính sát thương cơ bản
	effectiveAtk := float64(atk)
	if rand.Float64() < critRate {
		effectiveAtk *= 1.2
	}
	damage := int(effectiveAtk) - def
	if damage < 0 {
		damage = 0
	}

	// 2. Trừ vào shield trước nếu có
	if *shield > 0 {
		*shield -= damage
		if *shield < 0 {
			*shield = 0
		}
		return 0

	}

	// 3. Nếu không có shield → damage trực tiếp vào máu
	return damage
}

func applySkillEffect(gs *GameState, troop *Allies) {

	switch troop.Troops.Skill_info.Type {
	case "heal":
		if troop.Troops.Skill_info.Effect_speed == 0 {
			troop.Troops.HP += troop.Troops.Skill_info.Value * troop.Troops.Skill_info.Time
		}

	case "shield":
		if troop.Troops.Skill_info.Effect_speed == 0 {
			troop.Troops.Shield += troop.Troops.Skill_info.Value * troop.Troops.Skill_info.Time
		}
	}
}

func ApplySkillArea(gs *GameState, center Position, skillInfo *session.SkillLevelInfo) {
	for side := 0; side < 2; side++ {
		for i := range gs.Allies[side] {
			target := &gs.Allies[side][i]
			if isInRange(center, target.GetLocation(), float64(skillInfo.Effect_speed)) && target.IsAlive() {
				switch skillInfo.Type {
				case "damage":
					target.ReduceHP(skillInfo.Value)
				case "heal":
					target.Heal(skillInfo.Value)
				}
			}
		}
	}
}

func (a *Allies) ReduceHP(amount int) {
	switch a.Type {
	case "troop":
		a.Troops.HP -= amount
	case "guard_tower":
		a.Guard.HP -= amount
	case "king_tower":
		a.King.HP -= amount
		if !a.King.Active {
			a.King.Active = true
		}
	}
}

// IsAlive returns true if the unit is alive (HP > 0 and Active)
func (a *Allies) IsAlive() bool {
	switch a.Type {
	case "troop":
		return a.Troops.HP > 0 && a.Alive
	case "guard_tower":
		return a.Guard.HP > 0 && a.Alive
	case "king_tower":
		return a.King.HP > 0 && a.Alive
	default:
		return a.Alive
	}
}

func (a *Allies) Heal(amount int) {
	switch a.Type {
	case "troop":
		if a.Troops.HP <= 0 {
			return // Không hồi máu nếu đã chết
		}
		if a.Troops.HP+amount > a.Troops.CardInfo.Info.Hp {
			a.Troops.HP = a.Troops.CardInfo.Info.Hp
		} else {
			a.Troops.HP += amount
		}
	case "guard_tower":
		if a.Guard.HP <= 0 {
			return // Không hồi máu nếu đã chết
		}
		if a.Guard.HP+amount > a.Guard.GuardInfo.Info.Hp {
			a.Guard.HP = a.Guard.GuardInfo.Info.Hp
		} else {
			a.Guard.HP += amount
		}
	case "king_tower":
		if a.King.HP <= 0 {
			return // Không hồi máu nếu đã chết
		}
		if a.King.HP+amount > a.King.KingInfo.Info.Hp {
			a.King.HP = a.King.KingInfo.Info.Hp
		} else {
			a.King.HP += amount
		}
	}
}

func emitGameStateEvents(gs *GameState) {
	// Duyệt tất cả người chơi để gửi cập nhật
	for side := 0; side < 2; side++ {
		for _, player := range gs.Players[side] {
			event := CreateUpdateEvent(gs, player.Side)
			SendToClient(player.User, event)
		}
	}
}

func UpdateHandAfterPlay(player *PlayerState, usedCardID int) {
	for i, id := range player.Hand {
		if id == usedCardID {
			// ✅ Thêm lá bài vừa dùng vào cuối Deck TRƯỚC
			player.Deck = append(player.Deck, usedCardID)

			// ✅ Ghi đè vị trí đã dùng trong tay bằng NextCard
			player.Hand[i] = player.NextCard

			// ✅ Cập nhật NextCard từ Deck
			if len(player.Deck) > 0 {
				player.NextCard = player.Deck[0]
				player.Deck = player.Deck[1:]
			} else {
				player.NextCard = -1 // Hoặc giá trị mặc định
			}
			break
		}
	}
}

func SendDeckToClients(player *PlayerState, id string) {
	deckData := map[string]interface{}{
		"hand":        player.Hand,
		"nextCard":    player.NextCard,
		"elixir":      player.Elixir,
		"elixirTimer": player.ElixirTimer,
	}
	sendMessage(player.User.Client.Send, id, "deck", deckData)
}

func SendDeckToAllClients(gs *GameState) {
	for side := 0; side < 2; side++ {
		for _, player := range gs.Players[side] {
			deckData := map[string]interface{}{
				"hand":        player.Hand,
				"nextCard":    player.NextCard,
				"elixir":      player.Elixir,
				"elixirTimer": player.ElixirTimer,
			}
			sendMessage(player.User.Client.Send, "deck", "deck", deckData)
		}
	}
}

func CleanupAllies(gs *GameState) {
	for side := 0; side < 2; side++ {
		var alive []Allies
		for _, ally := range gs.Allies[side] {
			if !ally.Alive {
				// Nếu là guard tower đã chết, làm sạch vùng chiếm dụng
				if ally.Type == "guard_tower" {
					loc := ally.Guard.Location
					long := loc.long
					wide := loc.wide
					startX, startY := loc.X, loc.Y

					if side == 0 {
						// Phe trên: duyệt từ trái trên sang phải dưới
						for y := startY; y < startY+wide; y++ {
							for x := startX; x < startX+long; x++ {
								if y >= 0 && y < len(gs.Map) && x >= 0 && x < len(gs.Map[0]) {
									gs.Map[y][x] = 1
								}
							}
						}
					} else {
						// Phe dưới: duyệt từ phải dưới sang trái trên
						startX = loc.X - long + 1
						startY = loc.Y - wide + 1
						for y := startY + wide - 1; y >= startY; y-- {
							for x := startX + long - 1; x >= startX; x-- {
								if y >= 0 && y < len(gs.Map) && x >= 0 && x < len(gs.Map[0]) {
									gs.Map[y][x] = 1
								}
							}
						}
					}
				}
				continue
			}
			alive = append(alive, ally)
		}
		gs.Allies[side] = alive
	}
}

func MirrorPosition(x, y, rows, cols int) (int, int) {
	mirroredX := rows - 1 - x
	mirroredY := cols - 1 - y
	return mirroredX, mirroredY
}

type Allies struct {
	ID     string
	Type   string `json:"type"`
	Alive  bool
	Troops Troop
	Spells Spell
	King   King
	Guard  Guard
}

type Guard struct {
	HP          int
	Shield      int
	Time_attack float32
	Location    Position
	Skill_using bool
	GuardInfo   session.GuardTower
	Skill_info  session.SkillLevelInfo
	TargetID    string
}

type King struct {
	HP          int
	Shield      int
	Time_attack float32
	Location    Position
	Skill_using bool
	KingInfo    session.KingTower
	Skill_info  session.SkillLevelInfo
	TargetID    string
	Active      bool
}

type Troop struct {
	HP          int
	Time_attack float32
	Shield      int
	Location    Position
	Skill_using bool
	CardInfo    session.Card
	Skill_info  session.SkillLevelInfo
	TargetID    string
}

type Spell struct {
	Time        int // số lần ảnh hưởng
	Time_effect float32
	Location    Position
	Skill_using bool
	CardInfo    session.Card
	Skill_info  session.SkillLevelInfo
}

type GameState struct {
	Map     [][]int
	Players [2][]*PlayerState // 0: bên trái, 1: bên phải
	Allies  [2][]Allies
	Match   *session.MatchRoom
	Tick    int64
}

type PlayerState struct {
	User        *session.User
	Side        int    // 0: bên trên, 1: bên dưới
	Deck        []int  // Hàng đợi các chỉ số lá bài còn lại
	Hand        [4]int // 4 lá đang trên tay
	NextCard    int    // Lá kế tiếp sẽ vào tay
	Elixir      float64
	ElixirTimer float64
}

// long và wide là kích thước của ô trong game, có thể dùng để tính toán vị trí
// x,y là tọa độ của nó tính từ góc trên bên trái của bản đồ
type Position struct {
	X    int
	Y    int
	long int
	wide int
}

func NewGameState(match *session.MatchRoom) *GameState {
	mapData := loadMapFromMongoDB("Basic Map 20x35")

	var topPlayers []*PlayerState
	var botPlayers []*PlayerState

	for _, user := range match.User {
		side := 0
		if user.ID%2 != 0 {
			side = 1
		}

		// Combine Troops + Spells, shuffle, extract
		allCards := append(user.DataGame.Troops, user.DataGame.Spells...)
		shuffled := shuffleCards(allCards)
		indexes := extractCardIndexes(shuffled)

		hand := [4]int{indexes[0], indexes[1], indexes[2], indexes[3]}
		nextCard := indexes[4]
		deck := indexes[5:]

		player := &PlayerState{
			User:        user,
			Side:        side,
			Deck:        deck,
			Hand:        hand,
			NextCard:    nextCard,
			Elixir:      5.0,
			ElixirTimer: 1.0,
		}

		if side == 0 {
			topPlayers = append(topPlayers, player)
		} else {
			botPlayers = append(botPlayers, player)
		}
	}

	var topLeftGuardTower Allies
	var topRightGuardTower Allies
	var topKingTower Allies

	var bottomLeftGuardTower Allies
	var bottomRightGuardTower Allies
	var bottomKingTower Allies

	if len(topPlayers) == 1 && len(botPlayers) == 1 {
		topLeftGuardTower = Allies{
			ID:    uuid.New().String(), // Replace with uuid.New().String() if you want a UUID
			Type:  "guard_tower",
			Alive: true,
			Guard: Guard{
				HP:          topPlayers[0].User.DataGame.GuardTower.Info.Hp,
				Shield:      topPlayers[0].User.DataGame.GuardTower.Info.Shield,
				Time_attack: 0,
				Location:    Position{X: 3, Y: 6, long: 3, wide: 3}, // Set actual values as needed
				Skill_using: false,
				GuardInfo:   topPlayers[0].User.DataGame.GuardTower,
				Skill_info:  topPlayers[0].User.DataGame.GuardTower.Info.Skill,
				TargetID:    "",
			},
		}
		topRightGuardTower = Allies{
			ID:    uuid.New().String(), // Replace with uuid.New().String() if you want a UUID
			Type:  "guard_tower",
			Alive: true,
			Guard: Guard{
				HP:          topPlayers[0].User.DataGame.GuardTower.Info.Hp,
				Shield:      topPlayers[0].User.DataGame.GuardTower.Info.Shield,
				Time_attack: 0,
				Location:    Position{X: 14, Y: 6, long: 3, wide: 3}, // Set actual values as needed
				Skill_using: false,
				GuardInfo:   topPlayers[0].User.DataGame.GuardTower,
				Skill_info:  topPlayers[0].User.DataGame.GuardTower.Info.Skill,
				TargetID:    "",
			},
		}
		topKingTower = Allies{
			ID:    uuid.New().String(), // Replace with uuid.New().String() if you want a UUID
			Type:  "king_tower",
			Alive: true,
			King: King{
				HP:          topPlayers[0].User.DataGame.KingTower.Info.Hp,
				Shield:      topPlayers[0].User.DataGame.KingTower.Info.Shield,
				Time_attack: 0,
				Location:    Position{X: 8, Y: 2, long: 4, wide: 4}, // Set actual values as needed
				Skill_using: false,
				KingInfo:    topPlayers[0].User.DataGame.KingTower,
				Skill_info:  topPlayers[0].User.DataGame.KingTower.Info.Skill,
				TargetID:    "",
				Active:      false,
			},
		}

		x, y := MirrorPosition(14, 6, len(mapData), len(mapData[0]))
		bottomLeftGuardTower = Allies{
			ID:    uuid.New().String(),
			Type:  "guard_tower",
			Alive: true,
			Guard: Guard{
				HP:          botPlayers[0].User.DataGame.GuardTower.Info.Hp,
				Shield:      botPlayers[0].User.DataGame.GuardTower.Info.Shield,
				Time_attack: 0,
				Location:    Position{X: x, Y: y, long: 3, wide: 3}, // ví dụ tọa độ phía dưới
				Skill_using: false,
				GuardInfo:   botPlayers[0].User.DataGame.GuardTower,
				Skill_info:  botPlayers[0].User.DataGame.GuardTower.Info.Skill,
				TargetID:    "",
			},
		}

		x, y = MirrorPosition(3, 6, len(mapData), len(mapData[0]))
		bottomRightGuardTower = Allies{
			ID:    uuid.New().String(),
			Type:  "guard_tower",
			Alive: true,
			Guard: Guard{
				HP:          botPlayers[0].User.DataGame.GuardTower.Info.Hp,
				Shield:      botPlayers[0].User.DataGame.GuardTower.Info.Shield,
				Time_attack: 0,
				Location:    Position{X: x, Y: y, long: 3, wide: 3},
				Skill_using: false,
				GuardInfo:   botPlayers[0].User.DataGame.GuardTower,
				Skill_info:  botPlayers[0].User.DataGame.GuardTower.Info.Skill,
				TargetID:    "",
			},
		}

		x, y = MirrorPosition(8, 2, len(mapData), len(mapData[0]))
		bottomKingTower = Allies{
			ID:    uuid.New().String(),
			Type:  "king_tower",
			Alive: true,
			King: King{
				HP:          botPlayers[0].User.DataGame.KingTower.Info.Hp,
				Shield:      botPlayers[0].User.DataGame.KingTower.Info.Shield,
				Time_attack: 0,
				Location:    Position{X: x, Y: y, long: 4, wide: 4}, // ví dụ vị trí phía dưới
				Skill_using: false,
				KingInfo:    botPlayers[0].User.DataGame.KingTower,
				Skill_info:  botPlayers[0].User.DataGame.KingTower.Info.Skill,
				TargetID:    "",
				Active:      false,
			},
		}

	}

	return &GameState{
		Map: mapData,
		Players: [2][]*PlayerState{
			0: topPlayers,
			1: botPlayers,
		},
		Allies: [2][]Allies{
			0: {topKingTower, topLeftGuardTower, topRightGuardTower},
			1: {bottomKingTower, bottomLeftGuardTower, bottomRightGuardTower},
		},
		Match: match,
		Tick:  500,
	}
}

func extractCardIndexes(cards []session.Card) []int {
	indexes := make([]int, len(cards))
	for i, card := range cards {
		indexes[i] = card.Index
	}
	return indexes
}

func shuffleCards(cards []session.Card) []session.Card {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffled := make([]session.Card, len(cards))
	copy(shuffled, cards)
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func loadMapFromMongoDB(mapName string) [][]int {
	// Get the "map" collection from the "gameDB" database
	collection := db.MongoDatabase.Collection("map")

	// Create a filter to find the document with the field "name" matching mapName
	filter := bson.M{"name": mapName}

	// Temporary struct to hold the result
	var result struct {
		Data [][]int `bson:"tiles"`
	}

	// Find a single document that matches the filter
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		log.Printf("Map not found: %v\n", err)
		return nil
	}

	// Return the map data
	return result.Data
}

func getUserDeck(db *mongo.Database, user *session.User) error {
	collection := db.Collection("user_decks")
	filter := bson.M{"user_id": user.ID}

	var deck UserDeck
	err := collection.FindOne(context.Background(), filter).Decode(&deck)
	if err != nil {
		return err
	}

	// Tạo biến DataGame mới
	var dataGame session.DataGame

	// Lấy KingTowerLevel từ MySQL user_stats
	kingTowerLevel, err := getKingTowerLevelFromMySQL(user.ID)
	if err != nil {
		return err
	}

	// Lấy dữ liệu KingTower từ MongoDB theo level vừa lấy được
	kingTowerData, err := getTowerLevelInfoKing(db, deck.KingTower.Name, kingTowerLevel)
	if err != nil {
		return err
	}

	// Gán vào DataGame.KingTower
	dataGame.KingTower = session.KingTower{
		Level: kingTowerLevel,
		Name:  deck.KingTower.Name,
		Info:  kingTowerData,
	}

	// Lấy dữ liệu GuardTower tương tự
	guardTowerData, err := getTowerLevelInfoGuard(db, deck.GuardTower.Name, deck.GuardTower.Level)
	if err != nil {
		return err
	}
	dataGame.GuardTower = session.GuardTower{
		Level: deck.GuardTower.Level,
		Name:  deck.GuardTower.Name,
		Info:  guardTowerData,
	}

	// Xử lý card
	for idx, cardInfo := range deck.Cards {
		cardLevelInfo, cardType, err := getCardLevelInfo(db, cardInfo.Name, cardInfo.Level)
		if err != nil {
			return err
		}

		card := session.Card{
			Index: idx,
			Name:  cardInfo.Name,
			Level: cardInfo.Level,
			Info:  cardLevelInfo,
		}

		switch cardType {
		case "troop":
			dataGame.Troops = append(dataGame.Troops, card)
		case "spell":
			dataGame.Spells = append(dataGame.Spells, card)
		}
	}
	// Gán trực tiếp vào user.DataGame
	user.DataGame = dataGame

	return nil
}

func getKingTowerLevelFromMySQL(userID int) (int, error) {
	var level int
	query := "SELECT level FROM user_stats WHERE user_id = ?"
	err := db.DB.QueryRow(query, userID).Scan(&level)
	if err != nil {
		return 1, err
	}
	return level, nil
}

// Ví dụ hàm getTowerLevelInfo lấy dữ liệu level Tower từ db (hoặc cache)
func getTowerLevelInfoGuard(db *mongo.Database, name string, level int) (session.TowerLevelInfo, error) {
	collection := db.Collection("guard_tower") // giả sử
	filter := bson.M{"name": name}
	var TowerData struct {
		Name   string `bson:"name"`
		Skill  string `bson:"skill"`
		Levels []struct {
			Level       int     `bson:"level"`
			Hp          int     `bson:"hp"`
			Atk         int     `bson:"atk"`
			Def         int     `bson:"def"`
			CritRate    float64 `bson:"crit_rate"`
			AttackSpeed float64 `bson:"attack_speed"`
			Range       float64 `bson:"range"`
		} `bson:"levels"`
	}

	err := collection.FindOne(context.Background(), filter).Decode(&TowerData)
	if err != nil {
		return session.TowerLevelInfo{}, err
	}
	// Tìm thông tin level cụ thể
	for _, lvl := range TowerData.Levels {
		if lvl.Level == level {
			var skillInfo session.SkillLevelInfo

			// Nếu tower có skill, lấy thông tin skill theo level
			if TowerData.Skill != "" {
				skillInfo, _ = getSkillLevelInfo(db, TowerData.Skill, level)
			}

			// Trả về dữ liệu level
			return session.TowerLevelInfo{
				Skill:       skillInfo,
				Hp:          lvl.Hp,
				Atk:         lvl.Atk,
				Def:         lvl.Def,
				CritRate:    lvl.CritRate,
				AttackSpeed: lvl.AttackSpeed,
				Range:       lvl.Range,
			}, nil
		}
	}
	return session.TowerLevelInfo{}, errors.New("level not found")
}

// Ví dụ hàm getTowerLevelInfo lấy dữ liệu level Tower từ db (hoặc cache)
func getTowerLevelInfoKing(db *mongo.Database, name string, level int) (session.TowerLevelInfo, error) {
	collection := db.Collection("king_tower") // giả sử
	filter := bson.M{"name": name}
	var TowerData struct {
		Name   string `bson:"name"`
		Skill  string `bson:"skill"`
		Levels []struct {
			Level       int     `bson:"level"`
			Hp          int     `bson:"hp"`
			Atk         int     `bson:"atk"`
			Def         int     `bson:"def"`
			CritRate    float64 `bson:"crit_rate"`
			AttackSpeed float64 `bson:"attack_speed"`
			Range       float64 `bson:"range"`
		} `bson:"levels"`
	}

	err := collection.FindOne(context.Background(), filter).Decode(&TowerData)
	if err != nil {
		return session.TowerLevelInfo{}, err
	}

	// Tìm thông tin level cụ thể
	for _, lvl := range TowerData.Levels {
		if lvl.Level == level {
			var skillInfo session.SkillLevelInfo

			// Nếu tower có skill, lấy thông tin skill theo level
			if TowerData.Skill != "" {
				skillInfo, _ = getSkillLevelInfo(db, TowerData.Skill, level)
			}

			// Trả về dữ liệu level
			return session.TowerLevelInfo{
				Skill:       skillInfo,
				Hp:          lvl.Hp,
				Atk:         lvl.Atk,
				Def:         lvl.Def,
				CritRate:    lvl.CritRate,
				AttackSpeed: lvl.AttackSpeed,
				Range:       lvl.Range,
			}, nil
		}
	}
	return session.TowerLevelInfo{}, errors.New("level not found")
}

func getCardLevelInfo(db *mongo.Database, name string, level int) (session.CardLevelInfo, string, error) {
	collection := db.Collection("cards")
	filter := bson.M{"name": name}

	var cardMeta struct {
		Type string
	}

	err := collection.FindOne(context.Background(), filter).Decode(&cardMeta)
	if err != nil {
		return session.CardLevelInfo{}, "", err
	}

	switch cardMeta.Type {
	case "troop":
		var troopData struct {
			Mana   int    `bson:"mana"`
			Skill  string `bson:"skill"`
			Levels []struct {
				Level       int     `bson:"level"`
				Hp          int     `bson:"hp"`
				Atk         int     `bson:"atk"`
				Def         int     `bson:"def"`
				CritRate    float64 `bson:"crit_rate"`
				AttackSpeed float64 `bson:"attack_speed"`
				Range       float64 `bson:"range"`
				Speed       float64 `bson:"speed"`
			}
		}
		err := db.Collection("cards").FindOne(context.Background(), filter).Decode(&troopData)
		if err != nil {
			return session.CardLevelInfo{}, "", err
		}

		var skillInfo session.SkillLevelInfo
		if troopData.Skill != "" {
			skillInfo, _ = getSkillLevelInfo(db, troopData.Skill, level)
		}

		for _, lvl := range troopData.Levels {
			if lvl.Level == level {
				return session.CardLevelInfo{
					Skill:       skillInfo,
					Mana:        troopData.Mana,
					Hp:          lvl.Hp,
					Atk:         lvl.Atk,
					Def:         lvl.Def,
					CritRate:    lvl.CritRate,
					AttackSpeed: lvl.AttackSpeed,
					Range:       lvl.Range,
					Speed:       lvl.Speed,
				}, "troop", nil
			}
		}
		return session.CardLevelInfo{}, "", errors.New("troop level not found")

	case "spell":
		var spellData struct {
			Mana   int    `bson:"mana"`
			Skill  string `bson:"skill"`
			Levels []struct {
				Level  int     `bson:"level"`
				Radius float64 `bson:"radius"`
			}
		}
		err := db.Collection("cards").FindOne(context.Background(), filter).Decode(&spellData)
		if err != nil {
			return session.CardLevelInfo{}, "", err
		}

		var skillInfo session.SkillLevelInfo
		if spellData.Skill != "" {
			skillInfo, _ = getSkillLevelInfo(db, spellData.Skill, level)
		}
		for _, lvl := range spellData.Levels {
			if lvl.Level == level {
				return session.CardLevelInfo{
					Skill:  skillInfo,
					Mana:   spellData.Mana,
					Radius: lvl.Radius,
				}, "spell", nil
			}
		}
		return session.CardLevelInfo{}, "", errors.New("spell level not found")

	default:
		return session.CardLevelInfo{}, "", errors.New("invalid card type")
	}
}

func getSkillLevelInfo(db *mongo.Database, skillName string, level int) (session.SkillLevelInfo, error) {
	var skillDoc struct {
		Name        string `bson:"name"`
		Type        string `bson:"type"`
		Time        int    `bson:"time"`
		EffectSpeed int    `bson:"effect_speed"`
		Levels      []struct {
			Level int `bson:"level"`
			Value int `bson:"value"`
		} `bson:"levels"`
	}

	collection := db.Collection("skills")
	filter := bson.M{"name": skillName}

	err := collection.FindOne(context.Background(), filter).Decode(&skillDoc)
	if err != nil {
		return session.SkillLevelInfo{}, err
	}

	var value int
	for _, lvl := range skillDoc.Levels {
		if lvl.Level == level {
			value = lvl.Value
			break
		}
	}

	return session.SkillLevelInfo{
		Type:         skillDoc.Type,
		Time:         skillDoc.Time,
		Effect_speed: skillDoc.EffectSpeed,
		Value:        value,
	}, nil
}
