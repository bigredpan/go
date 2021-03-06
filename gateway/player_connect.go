package main

import (
	"bytes"
	"container/list"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

func playerHandle(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	var p = NewPlayerConnect(c, w, r)
	p.onConnect()
	defer p.conn.Close()
	go p.check()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("playerHandle read err:", err)
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
	last_pong   int64
	ping        int32
	ball        chan int64
	ball_list   *list.List
}

func NewPlayerConnect(c *websocket.Conn, w http.ResponseWriter, r *http.Request) *PlayerConnect {
	p := new(PlayerConnect)
	p.conn = c
	p.writer = w
	p.request = r
	p.last_pong = time.Now().Unix()
	p.ping = 0xff
	p.ball = make(chan int64, 256)
	p.ball_list = list.New()
	return p
}

func (p *PlayerConnect) check() {
	for {
		select {
		case t := <-p.ball:
			log.Printf("t", t)
			// p.ball_list.PushBack(t)
			// if p.ball_list.Len > 5{
			// }
		}
	}
}

func (p *PlayerConnect) do_close(code int, msg string) {
	log.Printf("PlayerConnect do_close ", code, msg)
	cm := websocket.FormatCloseMessage(code, msg)
	if err := p.conn.WriteMessage(websocket.CloseMessage, cm); err != nil {
		log.Printf("PlayerConnect do_close error:", err)
	}
	p.conn.Close()
}

func (p *PlayerConnect) onConnect() {
	Gateway().onPlayerConnect(p)
}

func (p *PlayerConnect) onDisconnect(err error) {
	Gateway().onPlayerDisconnect(p)
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
		p.on_client_ping(msg)
	} else if cmd == NOTICE_PING {

	} else if cmd == PLAYER_BALL {
		p.ball <- time.Now().Unix()
		// g.player_ball(connection)
		p.writeMessage(msg, nil)
		return
	} else {
		Gateway().onPlayerMessage(p, msg, p_cid)
	}
}

func (p *PlayerConnect) on_client_ping(message []byte) {
	p.last_pong = time.Now().Unix()
	p.writeMessage(message, nil)
	_, _, data := unpack(message)
	var msg interface{}
	err := json.Unmarshal(data, &msg)
	if err == nil {
		p.ping = int32(msg.([]interface{})[1].(float64))
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
