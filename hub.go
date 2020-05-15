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

type ServerMessage struct {
	Token    string `json:"Token"`
	IsServer bool   `json:"IsServer"`
	Username string `json:"Username"`
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	broadcast  chan Message
	register   chan Subscription
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

			s.client.isServer = isServer
			s.client.username = "PLACEHOLDER" // TODO: Get username from database
			message := ServerMessage{
				Token:    s.client.token,
				Username: s.client.username,
				IsServer: s.client.isServer,
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
					if (m.dest == c.token) || (m.dest == "" && m.token != c.token) || (c.isServer && m.token == c.token) {
						c.send <- m.data
					}
				}
			}
		}
	}
}
