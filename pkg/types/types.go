package types

import (
	"encoding/json"
)

/*
EventAction represents a domain of possible event actions.
*/
type EventAction int

const (
	// Events that can be sent by both Gatekeeper and core server.
	ActionPing EventAction = iota
	ActionGameInfo
	ActionRedirect
	ActionCompletedMove
	ActionClientsCounter
	// Events that can be sent only by the clients.
	ActionPong
	ActionMakeMove
	ActionEnterMatchmaking
	// Events that can be sent by both Gatekeeper and clients.
	ActionChat
	// Events that can be sent only by the core server.
	ActionAddRoom
	ActionRemoveRoom
)

/*
ExternalEvent represents an event exchanged between the server and WebSocket
clients.
*/
type ExternalEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"-"`
	Action   EventAction     `json:"a"`
}

/*
InternalEvent represents an event exchanged between the Gatekeeper and the core
server.  It contains additional metadata to route and handle the event.
*/
type InternalEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"cid"`
	RoomId   string          `json:"rid"`
	Action   EventAction     `json:"a"`
}

/*
GameInfo represents information the clients will recieve after connecting
to the room.
*/
type GameInfo struct {
	WhiteId     string `json:"wid"`
	BlackId     string `json:"bid"`
	TimeControl int    `json:"tc"`
	TimeBonus   int    `json:"tb"`
}

/*
CompletedMove represents information the clients will recieve after each
completed move.
*/
type CompletedMove struct {
	// Legal moves for the next turn.
	LegalMoves []int `json:"lm"`
	// Completed move in Standard Algebraic Notation.
	SAN string `json:"san"`
	// Board state in Forsyth-Edwards Notation.
	FEN string `json:"fen"`
	// Remaining seconds on the player's clock after completing the move.
	TimeLeft int `json:"tl"`
}

/*
AddRoom represents the payload of the ActionAddRoom event.
*/
type AddRoom struct {
	Id      string `json:"id"`
	WhiteId string `json:"wid"`
	BlackId string `json:"bid"`
}

type RoomParams struct {
	TimeControl int `json:"tc"`
	TimeBonus   int `json:"tb"`
}
