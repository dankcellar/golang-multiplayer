package main

import "encoding/json"

type Subscription struct {
	Client *Client
	Room   string
}

type ServerMessage struct {
	IsServer bool   `json:"isServer"`
	Player   string `json:"player"`
	Data     string `json:"data"`
	Room     string `json:"room"`
}

type Hub struct {
	Rooms      map[string]map[*Client]string
	Broadcast  chan ServerMessage
	Register   chan Subscription
	Unregister chan Subscription
}

func newHub() *Hub {
	return &Hub{
		Broadcast:  make(chan ServerMessage),
		Register:   make(chan Subscription),
		Unregister: make(chan Subscription),
		Rooms:      make(map[string]map[*Client]string),
	}
}

func (h *Hub) run() {
	for {
		select {
		case s := <-h.Register:
			// isServer := false
			connections := h.Rooms[s.Room]
			if connections == nil {
				connections = make(map[*Client]string)
				h.Rooms[s.Room] = connections
				// isServer = true
			}
			h.Rooms[s.Room][s.Client] = s.Client.Secret

			// s.Client.IsServer = isServer
			m := ServerMessage{
				Player:   s.Client.Secret,
				IsServer: true,
				Room:     s.Room,
				Data:     "{}",
			}
			data, _ := json.Marshal(m)
			s.Client.Send <- data

		case s := <-h.Unregister:
			connections := h.Rooms[s.Room]
			if connections != nil {
				if _, ok := connections[s.Client]; ok {
					close(s.Client.Send)
					delete(connections, s.Client)
					if len(connections) == 0 {
						delete(h.Rooms, s.Room)
					}
				}
			}

		case m := <-h.Broadcast:
			data, _ := json.Marshal(m)
			connections := h.Rooms[m.Room]
			for c := range connections {
				select {
				case c.Send <- data:
				default:
					close(c.Send)
					delete(connections, c)
					if len(connections) == 0 {
						delete(h.Rooms, m.Room)
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
