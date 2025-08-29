/*
Package event defines event types.
*/
package event

import (
	"encoding/json"
	"log"
)

/*
ExternalEvent represents an event type exchanged between the Gatekeeper and
WebSocket clients.  ClientId and RoomId are metadata fields that are excluded
from the JSON and added by the client structure.  Payload is an arbitrary
JSON-encoded object whose concrete type depends on the [EventAction].
*/
type ExternalEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"-"`
	RoomId   string          `json:"-"`
	Action   EventAction     `json:"a"`
}

/*
InternalEvent represents an event type which is exchanged between the Gatekeeper
and the core server.  Internal event's JSON contains a metadata which helps to
identify the sender and route the event to the room which will handle it.
*/
type InternalEvent struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"cid"`
	RoomId   string          `json:"rid"`
	Action   EventAction     `json:"a"`
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
	Matchmaking    EventAction = "mm"
	Redirect       EventAction = "r"
	Chat           EventAction = "c"
)

/*
AddRoomPayload is a predefined payload type.  Each message AddRoom event must
have exactly this type of payload.
*/
type AddRoomPayload struct {
	Id   string   `json:"id"`
	Subs []string `json:"s"`
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
