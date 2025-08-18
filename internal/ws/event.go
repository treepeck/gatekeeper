package ws

import (
	"encoding/json"
	"log"
)

type ServerEvent struct {
	Payload json.RawMessage `json:"p"`
	Act     Action          `json:"a"`
}

type ClientEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"cid"`
	RoomId   string          `json:"rid"`
	Act      Action          `json:"a"`
}

type Action int

const (
	// Server events.
	CLIENTS_COUNTER Action = iota
	ADD_ROOM
	REMOVE_ROOM
	UPDATE_ROOM

	// Client events.
	CREATE_ROOM
)

func encodeOrPanic(v any) []byte {
	p, err := json.Marshal(v)
	if err != nil {
		log.Panicf("cannot encode ServerEvent payload: %s", err)
	}

	return p
}
