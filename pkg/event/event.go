/*
Package event defines event types.
*/
package event

import (
	"encoding/json"
	"log"
)

/*
ExternalEvent represents an event type which is exchanged between the Gatekeeper
and clients.
*/
type ExternalEvent struct {
	// Concrette payload type depends on event action.
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"-"`
	RoomId   string          `json:"-"`
	Action   EventAction     `json:"a"`
}

/*
InternalEvent represents an event type which is exchanged between the Gatekeeper
and the core server.  Internal event contains metadata which helps to identify
the sender and route the room which will handle it.
*/
type InternalEvent struct {
	// Concrette payload type depends on event action.
	Payload json.RawMessage `json:"p"`
	// Sender id.
	ClientId string `json:"cid"`
	// Room which will recieve an event.
	RoomId string      `json:"rid"`
	Action EventAction `json:"a"`
}

/*
EventAction is a domain of possible actions.  The core server can declare custom
event action.
*/
type EventAction string

const (
	ClientsCounter EventAction = "cc"
	AddRoom        EventAction = "ar"
	RemoveRoom     EventAction = "rr"
	Chat           EventAction = "c"
)

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
