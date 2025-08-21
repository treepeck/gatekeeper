package event

import (
	"encoding/json"
	"log"
)

/*
ServerEvent represents an event emitted by the core server.  Gatekeeper's room
will broadcast each server event among all connected clients.
*/
type ServerEvent struct {
	Payload json.RawMessage `json:"p"`
	// Clients subscribed to that room will recive the event.
	RoomId string      `json:"rid"`
	Action EventAction `json:"a"`
}

/*
ClientEvent represents an event emitted by a client.  Gatekeeper's room will
forward client events to the core server.
*/
type ClientEvent struct {
	Payload json.RawMessage `json:"p"`
	// Sender id.
	ClientId string `json:"cid"`
	// Room which will handle the event.
	RoomId string      `json:"rid"`
	Action EventAction `json:"a"`
}

/*
EventAction is a domain of possible event types.  Core server can declare custom
event types.  By default Gatekeeper will recognize only actions, defined in the
const block bellow.
*/
type EventAction int

const (
	// Server events.
	CLIENTS_COUNTER EventAction = iota
	REDIRECT
	ADD_ROOM
	REMOVE_ROOM
	UPDATE_ROOM

	// Client events.
	CREATE_ROOM
)

/*
EncodeOrPanic is a helper function to encode a JSON payload on the fly skipping
the error check.  If the error occurs, the panic will be arised.
*/
func EncodeOrPanic(v any) []byte {
	p, err := json.Marshal(v)
	if err != nil {
		log.Panicf("cannot encode payload %v: %s", v, err)
	}
	return p
}

/*
DummyPayload helps to decode a single JSON field without knowing the full object
structure.  Used to decode a created room id.
*/
type DummyPayload struct {
	RoomId string `json:"rid"`
}
