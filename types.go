package pubsub

type Event struct {
	Data   []byte
	Action Action
	Pub    *Actor
}

type Action int

type Handler func(Event) ([]byte, error)
