package pubsub

type Event struct {
	Data   []byte
	TopId  TopicId
	Pub    *Actor
	Action Action
}

type Action int

// Predefined actions.
const (
	CREATE Action = iota
	SUBSCRIBE
	PUBLISH
)

type TopicId string

// Predefined topics.
const (
	HUB TopicId = "hub"
)
