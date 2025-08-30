package ws

/*
subman stores two maps of active connections.

By maintaining both mappings, this requirements are statisfied:
 1. Fast addition and removal of clients and rooms;
 2. Efficient lookup of all subscribers for a given room id;
 3. Efficient lookup of the room a client is subscribed to by client id.

Each subman operation is atomic and modifies two maps [except addRoom].
*/
type subman struct {
	byRoom   map[string]map[*client]struct{}
	byClient map[string]*client
}

func newSubman() *subman {
	byRoom := make(map[string]map[*client]struct{}, 1)
	byRoom["hub"] = make(map[*client]struct{})

	s := &subman{
		byRoom:   byRoom,
		byClient: make(map[string]*client),
	}

	return s
}

func (s *subman) addClient(c *client) {
	s.byRoom[c.roomId][c] = struct{}{}
	s.byClient[c.id] = c
}

func (s *subman) removeClient(c *client) {
	delete(s.byRoom[c.roomId], c)
	delete(s.byClient, c.id)
}

func (s *subman) addRoom(roomId string) {
	s.byRoom[roomId] = make(map[*client]struct{})
}

func (s *subman) removeRoom(roomId string) {
	for c := range s.byRoom[roomId] {
		s.removeClient(c)
	}
	delete(s.byRoom, roomId)
}
