package session

import (
	"context"
	"server/internal/types"
	"sync"
)

type RoomType string

const (
	RoomType1v1 RoomType = "1v1"
	RoomType2v2 RoomType = "2v2"
)

type Slot struct {
	ID     int
	Client *types.Client
}

type LobbyRoom struct {
	ID         string
	MaxSize    int
	Type       RoomType
	Slots      []*Slot
	Match      bool
	CancelFunc context.CancelFunc
}

type MatchRoom struct {
	ID      string
	MaxSize int
	Type    RoomType
	User    []*User
}

type User struct {
	ID       int
	Client   *types.Client
	DataGame DataGame
}

var (
	Lobbies   = make(map[string]*LobbyRoom)
	Matches   = make(map[string]*MatchRoom)
	MatchesMu sync.RWMutex
	LobbyMu   sync.RWMutex

	Clients   = make(map[*types.Client]bool)
	ClientsMu sync.RWMutex
)

func AddClient(c *types.Client) {
	ClientsMu.Lock()
	Clients[c] = true
	ClientsMu.Unlock()
}

func RemoveClient(c *types.Client) {
	ClientsMu.Lock()
	delete(Clients, c)
	ClientsMu.Unlock()
}

func IsUserLoggedIn(id int) bool {
	ClientsMu.RLock()
	defer ClientsMu.RUnlock()

	for client := range Clients {
		if client.User.ID == id && client.User.ID != 0 {
			return true
		}
	}
	return false
}

type DataGame struct {
	KingTower  KingTower
	GuardTower GuardTower
	Troops     []Card
	Spells     []Card
	Skills     []SkillLevelInfo
}

type KingTower struct {
	Level int
	Name  string
	Info  TowerLevelInfo
}

type GuardTower struct {
	Level int
	Name  string
	Info  TowerLevelInfo
}

type Card struct {
	Index int
	Name  string
	Level int
	Info  CardLevelInfo
}

type TowerLevelInfo struct {
	Skill       SkillLevelInfo `json:"skill,omitempty"`
	Hp          int            `json:"hp,omitempty"`
	Shield      int            `json:"shield,omitempty"`
	Atk         int            `json:"atk,omitempty"`
	Def         int            `json:"def,omitempty"`
	CritRate    float64        `json:"crit_rate,omitempty"`
	AttackSpeed float64        `json:"attack_speed,omitempty"`
	Range       float64        `json:"range,omitempty"`
}

type CardLevelInfo struct {
	Mana        int            `json:"mana,omitempty"`
	Radius      float64        `json:"radius,omitempty"`
	Skill       SkillLevelInfo `json:"skill,omitempty"`
	Hp          int            `json:"hp,omitempty"`
	Shield      int            `json:"shield,omitempty"`
	Atk         int            `json:"atk,omitempty"`
	Def         int            `json:"def,omitempty"`
	CritRate    float64        `json:"crit_rate,omitempty"`
	AttackSpeed float64        `json:"attack_speed,omitempty"`
	Range       float64        `json:"range,omitempty"`
	Speed       float64        `json:"speed,omitempty"`
}

type SkillLevelInfo struct {
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	Time         int    `json:"time,omitempty"`
	Effect_speed int    `json:"effect_speed,omitempty"`
	Value        int    `json:"value,omitempty"`
}
