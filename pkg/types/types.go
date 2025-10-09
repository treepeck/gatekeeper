package types

import "encoding/json"

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
	// Events that can be send only by the Gatekeeper
	ActionJoinRoom
	ActionLeaveRoom
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
ExternalEvent represents an event without metadata, exchanged between the
clients and frontend.
*/
type ExternalEvent struct {
	Payload json.RawMessage `json:"p"`
	Action  EventAction     `json:"a"`
}

/*
ClientEvent represents an event with enough metadata to route and handle it.
Those events are exchanged only between the core server and Gatekeeper via the
AMQP "gate" queue.
*/
type ClientEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"cid"`
	RoomId   string          `json:"rid"`
	Action   EventAction     `json:"a"`
}

/*
ServerEvent represents an event with additional metadata used to route it.
Those events are exchanged only between the core server and Gatekeeper via the
AMQP "core" queue.
*/
type ServerEvent struct {
	Payload json.RawMessage `json:"p"`
	RoomId  string          `json:"rid"`
	Action  EventAction     `json:"a"`
}

/*
Event payload types.
*/

type EnterMatchmaking struct {
	TimeControl int `json:"tc"`
	TimeBonus   int `json:"tb"`
}

type AddRoom struct {
	WhiteId string `json:"wid"`
	BlackId string `json:"bid"`
}

type CompletedMove struct {
	// Legal moves for the next turn.
	LegalMoves []int `json:"lm"`
	// Completed move in Standard Algebraic Notation.
	San string `json:"san"`
	// Board state in Forsyth-Edwards Notation.
	Fen string `json:"fen"`
	// Remaining seconds on the player's clock after completing the move.
	TimeLeft int `json:"tl"`
}

/*
GameInfo represents information the clients will recieve after connecting to the
room.
*/
type GameInfo struct {
	CompletedMoves []CompletedMove `json:"cm"`
	WhiteId        string          `json:"wid"`
	BlackId        string          `json:"bid"`
}
