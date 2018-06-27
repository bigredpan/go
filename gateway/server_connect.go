package main

import (
	"bytes"
	// "encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/url"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 5 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type ServerConnection struct {
	identifier string
	tag        string
	host       string
	conn       *websocket.Conn
	ping       int64
	ticker     *time.Ticker
	send       chan []byte
}

func NewServerConnection(id string, tag string, host string) *ServerConnection {
	m := new(ServerConnection)
	m.identifier = id
	m.tag = tag
	m.host = host
	m.send = make(chan []byte, 256)
	return m
}

func (c *ServerConnection) connect() {
	u := url.URL{Scheme: "ws", Host: c.host, Path: "/gateway"}
	log.Printf("u", u.String())
	var dialer *websocket.Dialer
	cookie := []string{fmt.Sprintf("id=%s;tag=%s", c.identifier, c.tag)}
	header := map[string][]string{"Cookie": cookie}
	conn, _, err := dialer.Dial(u.String(), header)
	c.conn = conn
	c.ping = 0xffff
	if err != nil {
		log.Printf("connect error: %s", err)
		c.reconnect()
		return
	}
	c.onConnect()
	go c.writePump()
	go c.update()
}

func (c *ServerConnection) do_close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *ServerConnection) reconnect() {
	time.AfterFunc(2*time.Second, c.connect)
}

func (c *ServerConnection) writePump() {
	c.ticker = time.NewTicker(pingPeriod)
	defer func() {
		log.Printf("writePump defer")
		c.ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Printf("writePump return 1")
				return
			}

			err := c.conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				log.Println("ServerConnection writePump err:", err)
				return
			}

			// w, err := c.conn.NextWriter(websocket.TextMessage)
			// if err != nil {
			// 	log.Printf("writePump return 2")
			// 	return
			// }
			// w.Write(message)

			// Add queued chat messages to the current websocket message.
			// n := len(c.send)
			// for i := 0; i < n; i++ {
			// 	w.Write(newline)
			// 	w.Write(<-c.send)
			// }

			// if err := w.Close(); err != nil {
			// 	log.Printf("writePump return 3")
			// 	return
			// }
		case <-c.ticker.C:
			data, _ := json.Marshal([2]int64{time.Now().Unix(), c.ping})
			msg := pack(PLAYER_PING, nil, data)
			if c.conn != nil {
				c.conn.WriteMessage(websocket.BinaryMessage, msg)
			}
		}
	}
}

func (c *ServerConnection) update() {
	defer func() {
		c.conn.Close()
		c.onDisconnect()
		c.reconnect()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("ServerConnection read err:", err)
			log.Printf("ServerConnection identifier:", c.identifier)
			return
		}
		// defer func() {
		// 	if p := recover(); p != nil {
		// 		log.Printf("panic recover! p: %v", p)
		// 	}
		// }()
		c.onReadMessage(message)
	}
}

func (c *ServerConnection) onConnect() {
	log.Printf("onConnect", c.identifier)

	// Gateway().onMasterConnect(c)
}

func (c *ServerConnection) onDisconnect() {
	log.Printf("onDisconnect", c.identifier)
	c.ticker.Stop()
}

type fetchFunc func(data []byte)

var fetchMap = map[string]fetchFunc{}

type MessageInfo struct {
	c   *ServerConnection
	msg []byte
	cid []byte
}

func (c *ServerConnection) onReadMessage(message []byte) {
	tag := message[0]
	var msg []byte
	var s_cid []byte
	if tag == SERVER_CID_TAG[0] {
		cid := message[SERVER_CID_TAG_SIZE:SERVER_CID_HEADER_SIZE]
		msg = message[SERVER_CID_HEADER_SIZE:]
		callback, ok := fetchMap[string(cid)]
		if ok {
			callback(msg)
			return
		}
		s_cid = message[0:SERVER_CID_HEADER_SIZE]
	} else if tag == PLAYER_CID_TAG[0] {
		msg = message[PLAYER_CID_HEADER_SIZE:]
		s_cid = message[0:PLAYER_CID_HEADER_SIZE]
	} else {
		msg = message
		s_cid = nil
	}
	Gateway().receive <- MessageInfo{c, msg, s_cid}
}

func (c *ServerConnection) writeMessage(message []byte, cid []byte) {
	if cid != nil {
		buf := new(bytes.Buffer)
		buf.Write(cid)
		buf.Write(message)
		message = buf.Bytes()
	}
	c.send <- message
}

func (c *ServerConnection) fetch(message []byte, callback fetchFunc) {
	cid := make([]byte, 16)
	rand.Read(cid)
	fetchMap[string(cid)] = callback
	buf := new(bytes.Buffer)
	buf.Write(SERVER_CID_TAG)
	buf.Write(cid)
	buf.Write(message)
	c.send <- buf.Bytes()
}
