package pubsub

type Actor struct {
	// actor id.
	id string
	// topic to which the actor is subscribed.
	topic *Topic
	// Recieve channel will recieve the events from the topic to which
	// the actor is subscribed.
	Recieve chan []byte
}

func NewActor(id string, t *Topic) *Actor {
	return &Actor{
		id:      id,
		topic:   t,
		Recieve: make(chan []byte, 192),
	}
}
