package main

type topic struct {
	id   string
	subs map[*client]struct{}
}

func newTopic(id string) *topic {
	return &topic{
		id:   id,
		subs: make(map[*client]struct{}),
	}
}

func (t *topic) subscribe(c *client) {
	t.subs[c] = struct{}{}
}

func (t *topic) unsubscribe(c *client) {
	delete(t.subs, c)
}

func (t *topic) isSub(c *client) bool {
	_, exists := t.subs[c]
	return exists
}
