package utils

import (
	"context"
	"server/internal/handle/game"
	"server/internal/session"
	"server/internal/types"
)

func CreateLobbyRoom(id string, roomType session.RoomType, match bool) *session.LobbyRoom {
	var maxSize int
	switch roomType {
	case session.RoomType1v1:
		maxSize = 2
	case session.RoomType2v2:
		maxSize = 4
	default:
		maxSize = 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Khởi tạo các slot rỗng
	slots := make([]*session.Slot, maxSize)
	for i := 0; i < maxSize; i++ {
		slots[i] = &session.Slot{ID: i, Client: nil}
	}

	room := &session.LobbyRoom{
		ID:         id,
		MaxSize:    maxSize,
		Type:       roomType,
		Slots:      slots,
		Match:      match,
		CancelFunc: cancel,
	}

	session.LobbyMu.Lock()
	defer session.LobbyMu.Unlock()
	session.Lobbies[id] = room

	go game.ControlLobby(ctx, room)

	return room
}

// Trả về cả room, bool thành công và slotIndex
func JoinLobbyRoom(lobbyID string, c *types.Client) (*session.LobbyRoom, bool, int) {
	session.LobbyMu.Lock()
	defer session.LobbyMu.Unlock()

	room, exists := session.Lobbies[lobbyID]
	if !exists {
		return nil, false, -1
	}

	for i, slot := range room.Slots {
		if slot.Client == nil {
			slot.Client = c
			c.RoomType = string(room.Type)
			c.SlotIndex = i
			return room, true, i
		}
	}
	return nil, false, -1
}

func LeaveLobbyRoom(lobbyID string, client *types.Client) bool {
	session.LobbyMu.Lock()
	defer session.LobbyMu.Unlock()

	room, ok := session.Lobbies[lobbyID]
	if !ok {
		return false
	}

	// Tìm slot chứa client và đặt lại nil
	for _, slot := range room.Slots {
		if slot.Client == client {
			slot.Client = nil
			break
		}
	}

	// Nếu không còn ai trong phòng thì hủy phòng
	allEmpty := true
	for _, slot := range room.Slots {
		if slot.Client != nil {
			allEmpty = false
			break
		}
	}

	if allEmpty {
		if room.CancelFunc != nil {
			room.CancelFunc()
		}
		RemoveLobbyRoom(lobbyID)
	}

	return true
}

func RemoveLobbyRoom(id string) {
	delete(session.Lobbies, id)
}

func FindAvailableLobby(roomType session.RoomType) *session.LobbyRoom {
	session.LobbyMu.Lock()
	defer session.LobbyMu.Unlock()

	for _, room := range session.Lobbies {
		if room.Type == roomType && !room.Match && !IsLobbyFull(room.ID) {
			return room
		}
	}
	return nil
}

func IsLobbyFull(lobbyID string) bool {
	session.LobbyMu.Lock()
	defer session.LobbyMu.Unlock()

	room, exists := session.Lobbies[lobbyID]
	if !exists {
		return false
	}

	for _, slot := range room.Slots {
		if slot.Client == nil {
			return false // còn trống
		}
	}
	return true // không còn slot trống
}
