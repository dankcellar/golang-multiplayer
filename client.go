package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 256
)

var upgrader = websocket.Upgrader{
	// ReadBufferSize:  1024,
	// WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	Secret string
	// IsServer bool
}

// ClientMessage takes incoming json``
// type ClientMessage struct {
// 	Event int    `json:"event"`
// 	Data  []byte `json:"data"`
// }

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (s Subscription) readPump() {
	defer func() {
		s.Client.Hub.Unregister <- s
		s.Client.Conn.Close()
	}()
	s.Client.Conn.SetReadLimit(maxMessageSize)
	s.Client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	s.Client.Conn.SetPongHandler(func(string) error { s.Client.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := s.Client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		if json.Valid(message) {
			m := ServerMessage{false, s.Client.Secret, string(message), s.Room}
			s.Client.Hub.Broadcast <- m
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (s Subscription) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.Client.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-s.Client.Send:
			s.Client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				s.Client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := s.Client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			// n := len(s.Client.Send)
			// for i := 0; i < n; i++ {
			// 	w.Write([]byte{' '})
			// 	w.Write(<-s.Client.Send)
			// }

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			s.Client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.Client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
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

	secret := uuid.Must(uuid.NewV4()).String()
	c := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256), Secret: secret}
	s := Subscription{c, room}
	c.Hub.Register <- s

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go s.writePump()
	go s.readPump()
}
