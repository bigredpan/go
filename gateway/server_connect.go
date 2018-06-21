package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/url"
	"time"
)

type MasterServerConnection struct {
	identifier string
	tag        string
	host       string
	conn       *websocket.Conn
	ping       int64
}

func (c *MasterServerConnection) connect() {
	u := url.URL{Scheme: "ws", Host: c.host, Path: "/gateway"}
	fmt.Println("u %d", u)
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial(u.String(), map[string][]string{"Cookie": {fmt.Sprint("id=%s", c.identifier), fmt.Sprint("tag=%s", c.tag)}})
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

func (c *MasterServerConnection) reconnect() {
	time.AfterFunc(2*time.Second, c.connect)
}

func (c *MasterServerConnection) heart(second time.Duration) {
	ticker := time.NewTicker(second)
	for _ = range ticker.C {
		buf := new(bytes.Buffer)
		var a uint16 = 0xf002
		var b uint16 = 0
		var d uint8 = 1
		binary.Write(buf, binary.LittleEndian, a)
		binary.Write(buf, binary.LittleEndian, b)
		binary.Write(buf, binary.LittleEndian, d)
		data, _ := json.Marshal([2]int64{time.Now().Unix(), c.ping})

		binary.Write(buf, binary.LittleEndian, data)
		msg := buf.Bytes()

		c.conn.WriteMessage(websocket.BinaryMessage, msg)
	}
}

func (c *MasterServerConnection) update() {
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

func (c *MasterServerConnection) onConnect() {
	Gateway().onMasterConnect(c)
}

func (c *MasterServerConnection) onDisconnect() {

}

type fetchFunc func(data []byte)

var fetchMap = map[string]fetchFunc{}

func (c *MasterServerConnection) onReadMessage(message []byte) {
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
	Gateway().onServerMessage(msg, s_cid)
}

func (c *MasterServerConnection) writeMessage(message []byte, cid []byte) {
	if cid != nil {
		buf := new(bytes.Buffer)
		// buf.Write(PLAYER_CID_TAG)
		buf.Write(cid)
		buf.Write(message)
		message = buf.Bytes()
	}
	err := c.conn.WriteMessage(websocket.BinaryMessage, message)
	if err != nil {
		log.Println("write:", err)
	}
}

func (c *MasterServerConnection) fetch(message []byte, callback fetchFunc) {
	cid := make([]byte, 16)
	rand.Read(cid)
	// message, cid := pack_cid(PLAYER_CONNECT, nil, data)
	fetchMap[string(cid)] = callback
	// c.writeMessage(message, cid)

	buf := new(bytes.Buffer)
	buf.Write(SERVER_CID_TAG)
	buf.Write(cid)
	buf.Write(message)

	err := c.conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
	if err != nil {
		log.Println("fetch:", err)
	}
}
