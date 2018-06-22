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

type ServerConnection struct {
	identifier string
	tag        string
	host       string
	conn       *websocket.Conn
	ping       int64
	ticker     *time.Ticker
}

func NewServerConnection(id string, tag string, host string) *ServerConnection {
	m := new(ServerConnection)
	m.identifier = id
	m.tag = tag
	m.host = host
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
	go c.heart(time.Second * 5)
	go c.update()
}

func (c *ServerConnection) do_close() {
	c.conn.Close()
}

func (c *ServerConnection) reconnect() {
	time.AfterFunc(2*time.Second, c.connect)
}

func (c *ServerConnection) heart(second time.Duration) {
	c.ticker = time.NewTicker(second)
	for _ = range c.ticker.C {
		data, _ := json.Marshal([2]int64{time.Now().Unix(), c.ping})
		msg := pack(PLAYER_PING, nil, data)
		if c.conn != nil {
			c.conn.WriteMessage(websocket.BinaryMessage, msg)
		}
	}
}

func (c *ServerConnection) update() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("read:", err)
			c.onDisconnect()
			c.reconnect()
			return
		}
		c.onReadMessage(message)

	}
}

func (c *ServerConnection) onConnect() {
	log.Printf("onConnect", c.identifier)

	// Gateway().onMasterConnect(c)
}

func (c *ServerConnection) onDisconnect() {
	log.Printf("onDisconnect", c.identifier)
	c.conn = nil
	c.ticker.Stop()
	c.ticker = nil
}

type fetchFunc func(data []byte)

var fetchMap = map[string]fetchFunc{}

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
		// Gateway().onServerMessage(msg, s_cid)
	} else if tag == PLAYER_CID_TAG[0] {
		msg = message[PLAYER_CID_HEADER_SIZE:]
		// cmd, id_list, body := unpack(msg)
		s_cid = message[0:PLAYER_CID_HEADER_SIZE]
		// Gateway().onServerMessage(msg, p_cid)
	} else {
		msg = message
		s_cid = nil
	}
	Gateway().onServerMessage(c, msg, s_cid)
}

func (c *ServerConnection) writeMessage(message []byte, cid []byte) {
	if cid != nil {
		buf := new(bytes.Buffer)
		buf.Write(cid)
		buf.Write(message)
		message = buf.Bytes()
	}
	err := c.conn.WriteMessage(websocket.BinaryMessage, message)
	if err != nil {
		log.Println("write:", err)
	}
}

func (c *ServerConnection) fetch(message []byte, callback fetchFunc) {
	cid := make([]byte, 16)
	rand.Read(cid)
	fetchMap[string(cid)] = callback

	buf := new(bytes.Buffer)
	buf.Write(SERVER_CID_TAG)
	buf.Write(cid)
	buf.Write(message)

	err := c.conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
	if err != nil {
		log.Println("fetch:", err)
	}
}
