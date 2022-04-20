package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	secret string
	// isServer bool
}

// ClientMessage takes incoming json``
type ClientMessage struct {
	event int32
	data  string
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
	s.client.conn.SetReadLimit(maxMessageSize)
	s.client.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.client.conn.SetPongHandler(func(string) error { s.client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := s.client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// TODO validate some data
		cm := ClientMessage{}
		json.Unmarshal(message, &cm)
		if !json.Valid([]byte(cm.data)) {
			cm.data = "{}"
			cm.event = 0
		}

		bytes, _ := json.Marshal(&cm)
		m := ServerMessage{s.client.secret, false, bytes, s.room}
		s.client.hub.broadcast <- m
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
		s.client.conn.Close()
	}()
	for {
		select {
		case message, ok := <-s.client.send:
			s.client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				s.client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := s.client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// // Add queued chat messages to the current websocket message.
			// n := len(s.client.send)
			// for i := 0; i < n; i++ {
			// 	w.Write([]byte{' '})
			// 	w.Write(<-s.client.send)
			// }

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			s.client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
	c := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), secret: secret}
	s := Subscription{c, room}
	c.hub.register <- s

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go s.writePump()
	go s.readPump()
}
