package ws

import (
	"encoding/json"
	"log"
)

type ServerEvent struct {
	Payload json.RawMessage `json:"p"`
	Act     Action          `json:"a"`
}

func (e ServerEvent) Encode() []byte {
	p, _ := json.Marshal(e)
	return p
}

type ClientEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"cid"`
	RoomId   string          `json:"rid"`
	Act      Action          `json:"a"`
}

type Action int

const (
	// Server messages.
	CLIENTS_COUNTER Action = iota
	ADD_ROOM
	REMOVE_ROOM
	UPDATE_ROOM

	// Client messages.
	CREATE_ROOM
)

func encodeOrPanic(v any) []byte {
	p, err := json.Marshal(v)
	if err != nil {
		log.Panicf("cannot encode ServerEvent payload: %s", err)
	}

	return p
}
