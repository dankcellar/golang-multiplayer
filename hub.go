package main

type Message struct {
	data  []byte
	room  string
	token string
	dest  string
}

type Subscription struct {
	client *Client
	room   string
}

type Token struct {
	Token    string
	Dest     string
	Username string
	IsServer bool
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients.
	rooms map[string]map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan Message

	// Register requests from the clients.
	register chan Subscription

	// Unregister requests from clients.
	unregister chan Subscription
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message),
		register:   make(chan Subscription),
		unregister: make(chan Subscription),
		rooms:      make(map[string]map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case s := <-h.register:
			isServer := false
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*Client]bool)
				h.rooms[s.room] = connections
				isServer = true
			}
			h.rooms[s.room][s.client] = true

			// Init client with reply token and isServer
			message := Token{
				Token:    s.client.token,
				Username: "PLACEHOLDER",
				IsServer: isServer,
			}
			bytes, _ := json.Marshal(&message)
			s.client.send <- bytes

		case s := <-h.unregister:
			connections := h.rooms[s.room]
			if connections != nil {
				if _, ok := connections[s.client]; ok {
					close(s.client.send)
					delete(connections, s.client)
					if len(connections) == 0 {
						delete(h.rooms, s.room)
					}
				}
			}

		case m := <-h.broadcast:
			connections := h.rooms[m.room]
			for c := range connections {
				// select {
				// case c.send <- m.data:
				// default:
				// 	close(c.send)
				// 	delete(connections, c)
				// 	if len(connections) == 0 {
				// 		delete(h.rooms, m.room)
				// 	}
				// }

				if len(c.send) == cap(c.send) {
					close(c.send)
					delete(connections, c)
					if len(connections) == 0 {
						delete(h.rooms, m.room)
					}
				} else {
					if m.token != c.token || m.dest == c.token {
						c.send <- m.data
					}
				}
			}
		}
	}
}
