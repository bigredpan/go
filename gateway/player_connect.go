package main

import (
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
	p := PlayerConnect{c, w, r}
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
	conn    *websocket.Conn
	writer  http.ResponseWriter
	request *http.Request
}

func (p PlayerConnect) onConnect() {
	Gateway().onPlayerConnect(p)
}

func (p PlayerConnect) onDisconnect(err error) {

}

func (p PlayerConnect) onMessage(message []byte) {

}

func (p PlayerConnect) get_cookie(name string, def ...string) string {
	c, err := p.request.Cookie(name)
	if err != nil {
		if len(def) > 0 {
			return def[0]
		}
		return ""
	}
	return c.Value
}
