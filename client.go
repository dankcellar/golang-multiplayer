package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// const (
// 	// Time allowed to write a message to the peer.
// 	writeWait = 10 * time.Second

// 	// Time allowed to read the next pong message from the peer.
// 	pongWait = 60 * time.Second

// 	// Send pings to peer with this period. Must be less than pongWait.
// 	pingPeriod = (pongWait * 9) / 10

// 	// Maximum message size allowed from peer.
// 	maxMessageSize = 512
// )

// var (
// 	newline = []byte{'\n'}
// 	space   = []byte{' '}
// )

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	token    string
	username string
	isServer bool
}

// ClientMessage takes incoming json
type ClientMessage struct {
	mType    int32  `json:"Type"`
	dest     string `json:"Dest"`
	token    string `json:"Token"`
	isServer bool   `json:"IsServer"`
	// Player Packets
	username string  `json:"Username,omitempty"`
	posX     float32 `json:"PosX,omitempty"`
	posY     float32 `json:"PosY,omitempty"`
	posZ     float32 `json:"PosZ,omitempty"`
	rotX     float32 `json:"RotX,omitempty"`
	rotY     float32 `json:"RotY,omitempty"`
	rotZ     float32 `json:"RotZ,omitempty"`
	// Voice Packets
	isP2P bool   `json:"IsP2P,omitempty"`
	data  []byte `json:"Data,omitempty"`
	// GameState Packets
	crates      string `json:"Crates,omitempty"`
	action      int32  `json:"Action,omitempty"`
	actionCrate string `json:"ActionCrate,omitempty"`
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (s Subscription) readPump() {
	defer func() {
		s.client.hub.unregister <- s
		s.client.conn.Close()
	}()
	// c.conn.SetReadLimit(maxMessageSize)
	// c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := s.client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		cm := ClientMessage{}
		json.Unmarshal(message, &cm)
		cm.token = s.client.token
		cm.username = s.client.username
		cm.isServer = s.client.isServer

		bytes, _ := json.Marshal(&cm)
		m := Message{bytes, s.room, s.client.token, cm.dest}
		s.client.hub.broadcast <- m
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (s Subscription) writePump() {
	// ticker := time.NewTicker(pingPeriod)
	defer func() {
		// ticker.Stop()
		s.client.conn.Close()
	}()
	for {
		select {
		case message, ok := <-s.client.send:
			// c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				s.client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := s.client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// w, err := s.client.conn.NextWriter(websocket.TextMessage)
			// if err != nil {
			// 	return
			// }
			// w.Write(message)

			// Add queued chat messages to the current websocket message.
			// n := len(s.client.send)
			// for i := 0; i < n; i++ {
			// 	w.Write(space)
			// 	w.Write(<-s.client.send)
			// }

			// if err := w.Close(); err != nil {
			// 	return
			// }
			// case <-ticker.C:
			// 	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			// 	if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			// 		return
			// 	}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request, room string, token string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// token := guuid.New().String()
	c := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), token: token}
	s := Subscription{c, room}
	c.hub.register <- s

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go s.writePump()
	go s.readPump()
}
