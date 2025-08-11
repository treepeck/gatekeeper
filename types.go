package pubsub

type Event struct {
	Data   []byte
	Action Action
	Pub    *Actor
}

type Action int

const (
	ADD_SUB Action = iota
	REMOVE_SUB
)

type Handler func(Event) ([]byte, error)
