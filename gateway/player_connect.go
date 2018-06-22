package main

import (
	"bytes"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

func playerHandle(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	p := PlayerConnect{c, w, r, 0, nil, "", ""}
	p.onConnect()
	defer c.Close()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			p.onDisconnect(err)
			break
		}
		p.onMessage(message)
	}
}

type PlayerConnect struct {
	conn        *websocket.Conn
	writer      http.ResponseWriter
	request     *http.Request
	player_id   int
	login_data  map[string]string
	room_server string
	room        string
}

func (p *PlayerConnect) onConnect() {
	Gateway().onPlayerConnect(p)
}

func (p *PlayerConnect) onDisconnect(err error) {

}

func (p *PlayerConnect) onMessage(message []byte) {
	tag := message[0]
	var msg []byte
	var p_cid []byte
	if tag == PLAYER_CID_TAG[0] {
		msg = message[PLAYER_CID_HEADER_SIZE:]
		p_cid = message[0:PLAYER_CID_HEADER_SIZE]
	} else {
		msg = message
		p_cid = nil
	}
	cmd, _, _ := unpack(msg)

	if cmd == PLAYER_PING {

	} else if cmd == NOTICE_PING {

	} else {

		Gateway().onPlayerMessage(p.player_id, msg, p_cid)
	}

}

func (p *PlayerConnect) get_cookie(name string, def ...string) string {
	c, err := p.request.Cookie(name)
	if err != nil {
		if len(def) > 0 {
			return def[0]
		}
		return ""
	}
	return c.Value
}

func (p *PlayerConnect) writeMessage(message []byte, cid []byte) {
	if cid != nil {
		var buf bytes.Buffer
		buf.Write(cid)
		buf.Write(message)
		message = buf.Bytes()
	}
	err := p.conn.WriteMessage(websocket.BinaryMessage, message)
	if err != nil {
		log.Println("write:", err)
	}
}
