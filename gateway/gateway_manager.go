package main

import (
	"encoding/json"
	// "log"
	"sync"
	"time"
)

type GatewayManager struct {
	identifier    string
	tag           string
	runtime       string
	master_server *MasterServerConnection
	players       map[int]*PlayerConnect
}

var manager *GatewayManager
var once sync.Once

func Gateway() *GatewayManager {
	once.Do(func() {
		manager = &GatewayManager{}
		manager.players = map[int]*PlayerConnect{}
	})
	return manager
}

func (g *GatewayManager) init(runtime string, identifier string, tag string) {
	g.identifier = identifier
	g.tag = tag
	g.runtime = runtime
	go g.update(time.Second)
	//go g.Check(time.Second*5)
	//go g.Monitor(time.Second*300)
}

func (g *GatewayManager) update(second time.Duration) {
	ticker := time.NewTicker(second)
	for _ = range ticker.C {
		// log.Printf("GatewayManager Update")
	}
}

func (g *GatewayManager) onMasterConnect(conn *MasterServerConnection) {
	manager.master_server = conn
}

func (g *GatewayManager) onServerMessage(message []byte, cid []byte) {
	cmd, id_list, body := unpack(message)
	if cmd == PLAYER_PING {

	} else if cmd == NOTICE_PING {

	} else if cmd == CENTER_GATEWAY_SERVERS {
		g.gateway_servers(json.UnMarshal(body))
	} else if cmd == CENTER_GATEWAY_CONFIG {
		// self.gateway_config(Serializer.loads(body))
	} else if cmd == CENTER_GATEWAY_SELF_KICK {
		// self.gateway_kick(id_list[0], CloseReasons.ERROR_SELF_KICK)
	} else if cmd == CENTER_GATEWAY_KICK {
		// self.gateway_kick(id_list[0], CloseReasons.ERROR_PLAYER_KICK)
	} else {
		msg := pack(cmd, nil, body)

		if cmd == NOTICE_SPEAKER || cmd == NOTICE_CONFIG_CHANGE || cmd == NOTICE_SNATCH_UPDATE {
			for _, connection := range g.players {
				connection.writeMessage(msg, cid)
			}

		} else {
			for _, player_id := range id_list {
				connection, ok := g.players[player_id]
				if ok {
					if cmd == NOTICE_ROOM_JOIN {
						// connection.room_server = server.identifier
						// connection.room = Serializer.loads(body)["roomData"]["name"]
					} else if cmd == NOTICE_ROOM_RESULTS || cmd == NOTICE_ROOM_LEAVE {
						// connection.room = ""
						// del connection.ball_list[:]
					}
					connection.writeMessage(msg, cid)
				}
			}
		}
	}
}

func (g *GatewayManager) onPlayerMessage(player_id int, msg []byte, cid []byte) {
	cmd, _, body := unpack(msg)
	if cmd == PLAYER_BALL {
		return
	}
	g.master_server.writeMessage(pack(cmd, []int{player_id}, body), cid)
}

func (g *GatewayManager) onPlayerConnect(connection *PlayerConnect) {
	device := connection.get_cookie("deviceId")
	account := connection.get_cookie("account")
	session := connection.get_cookie("session")
	open_info := connection.get_cookie("info")
	lang := connection.get_cookie("lang", "en")
	config_md5 := connection.get_cookie("configMD5", "")
	channel := connection.get_cookie("channel")
	create := connection.get_cookie("createUser")
	country := connection.get_cookie("country")
	ip := connection.get_cookie("ip", "")
	tag := connection.get_cookie("tag", "")
	tz := "8"
	// tz := connection.get_cookie("tz", "8")
	// tz = max(-11, min(13, tz))
	data := map[string]string{
		"account":    account,
		"session":    session,
		"channel":    channel,
		"device":     device,
		"open_info":  open_info,
		"ip":         ip,
		"create":     create,
		"country":    country,
		"tz":         tz,
		"lang":       lang,
		"server":     g.identifier,
		"tag":        tag,
		"config_md5": config_md5,
	}
	body, _ := json.Marshal(data)
	message := pack(PLAYER_CONNECT, nil, body)
	g.master_server.fetch(message, func(msg []byte) {
		_, id_list, body := unpack(msg)
		player_id := id_list[0]
		connection.player_id = player_id
		connection.login_data = data
		g.players[player_id] = connection
		connection.writeMessage(pack(NOTICE_INIT, []int{player_id}, body), nil)
	})
}
