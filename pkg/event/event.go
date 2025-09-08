package event

import (
	"encoding/json"
	"log"
)

/*
EventAction represents a domain of possible event actions.
*/
type EventAction int

const (
	// Events that can be sent by both Gatekeeper and core server.
	Ping EventAction = iota
	GameInfo
	Redirect
	CompletedMove
	ClientsCounter
	// Events that can be sent only by the clients.
	Pong
	MakeMove
	EnterMatchmaking
	// Events that can be sent by both Gatekeeper and clients.
	Chat
	// Events that can be sent only by the core server.
	AddRoom
	RemoveRoom
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
Event payload types.
*/

/*
GameInfoPayload represents information the clients will recieve after connecting
to the room.
*/
type GameInfoPayload struct {
	WhiteId     string `json:"wid"`
	BlackId     string `json:"bid"`
	TimeControl int    `json:"tc"`
	TimeBonus   int    `json:"tb"`
}

/*
CompletedMovePayload represents information the clients will recieve after each
completed move.
*/
type CompletedMovePayload struct {
	// Legal moves for the next turn.
	LegalMoves []int `json:"lm"`
	// Completed move in Standard Algebraic Notation.
	SAN string `json:"san"`
	// Board state in Forsyth-Edwards Notation.
	FEN string `json:"fen"`
	// Remaining seconds on the player's clock after completing the move.
	TimeLeft int `json:"tl"`
}

type EnterMatchmakingPayload struct {
	TimeControl int `json:"tc"`
	TimeBonus   int `json:"tb"`
}

type AddRoomPayload struct {
	Id      string `json:"id"`
	WhiteId string `json:"wid"`
	BlackId string `json:"bid"`
}

/*
EncodeOrPanic is a helper function to encode a JSON payload on the fly skipping
the error check.  Panics if the error occurs.
*/
func EncodeOrPanic(v any) []byte {
	p, err := json.Marshal(v)
	if err != nil {
		log.Panicf("cannot encode payload %v: %s", v, err)
	}
	return p
}
