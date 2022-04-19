package main

type Subscription struct {
	client *Client
	room   string
}

type ServerMessage struct {
	player   string
	isServer bool
	data     []byte
	room     string
}

type Hub struct {
	rooms      map[string]map[*Client]string
	broadcast  chan ServerMessage
	register   chan Subscription
	unregister chan Subscription
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan ServerMessage),
		register:   make(chan Subscription),
		unregister: make(chan Subscription),
		rooms:      make(map[string]map[*Client]string),
	}
}

func (h *Hub) run() {
	for {
		select {
		case s := <-h.register:
			// isServer := false
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*Client]string)
				h.rooms[s.room] = connections
				// isServer = true
			}
			h.rooms[s.room][s.client] = s.client.secret

			// s.client.isServer = isServer
			message := ServerMessage{
				player:   s.client.secret,
				isServer: true,
				room:     s.room,
				data:     nil, // some server data like len(connections)
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
				select {
				case c.send <- m.data:
				default:
					close(c.send)
					delete(connections, c)
					if len(connections) == 0 {
						delete(h.rooms, m.room)
					}
				}
			}
		}
	}
}

// func authToken(conn *websocket.Conn, room, token string) bool {
// 	if _, ok := Member[room][conn]; ok {
// 		if token == Member[room][conn]["token"] {
// 			return true
// 		}
// 		return false
// 	}
// 	return false
// }

// func getMemberList(room string) []map[string]interface{} {
// 	var list []map[string]interface{}
// 	log.Println("cur - before: ", Member[room])
// 	for key, _ := range Member[room] {
// 		cur := Member[room][key]
// 		log.Println("cur: ", cur)
// 		if cur["name"] != nil {
// 			list = append(list, map[string]interface{}{
// 				"token": cur["count"],
// 				"name":  cur["name"],
// 			})
// 		}
// 	}
// 	return list
// }
