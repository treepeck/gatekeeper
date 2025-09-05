package event

import (
	"encoding/json"
	"log"
)

type EventAction int

const (
	ClientsCounter EventAction = iota
	Ping
	Pong
	Move
	AddRoom
	EnterMatchmaking
	Redirect
	RemoveRoom
	Chat
)

type ExternalEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"-"`
	Action   EventAction     `json:"a"`
}

type InternalEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"cid"`
	RoomId   string          `json:"rid"`
	Action   EventAction     `json:"a"`
}

/*
Event payload types.
*/

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
