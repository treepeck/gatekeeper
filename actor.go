package pubsub

type Actor struct {
	// actor id.
	Id string
	// Topic to which the actor is subscribed.
	Topic *Topic
	// Recieve channel will recieve the events from the topic to which
	// the actor is subscribed.
	Recieve chan []byte
}

func NewActor(id string, t *Topic) *Actor {
	return &Actor{
		Id:      id,
		Topic:   t,
		Recieve: make(chan []byte, 192),
	}
}
