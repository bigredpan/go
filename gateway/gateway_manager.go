package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type GatewayManager struct {
	identifier    string
	tag           string
	runtime       string
	master_server *ServerConnection
	players       map[int]*PlayerConnect
	room_servers  map[string]*ServerConnection
}

var manager *GatewayManager
var once sync.Once

func Gateway() *GatewayManager {
	once.Do(func() {
		manager = &GatewayManager{}
		manager.players = make(map[int]*PlayerConnect)
		manager.room_servers = make(map[string]*ServerConnection)
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

func (g *GatewayManager) onMasterConnect(server *ServerConnection) {
	manager.master_server = server
}

func (g *GatewayManager) onServerMessage(server *ServerConnection, message []byte, cid []byte) {
	cmd, id_list, body := unpack(message)

	if cmd == PLAYER_PING {

	} else if cmd == NOTICE_PING {

	} else if cmd == CENTER_GATEWAY_SERVERS {
		var data = make(map[string]interface{})
		json.Unmarshal(body, &data)
		g.gateway_servers(data)
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
						connection.room_server = server.identifier
						var data = make(map[string]interface{})
						json.Unmarshal(body, &data)
						room_data := data["roomData"].(map[string]interface{})
						connection.room = room_data["name"].(string)
					} else if cmd == NOTICE_ROOM_RESULTS || cmd == NOTICE_ROOM_LEAVE {
						connection.room = ""
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
	var server *ServerConnection = nil
	if cmd > CHAT_COMMAND {
		server = nil
	} else if cmd > MASTER_COMMAND {
		server = g.master_server
	} else if cmd > ROOM_COMMAND {
		connection, _ := g.players[player_id]
		if connection.room_server != "" {
			server = g.room_servers[connection.room_server]
		}
	}
	if server != nil {
		server.writeMessage(pack(cmd, []int{player_id}, body), cid)
	}
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

func (g *GatewayManager) onPlayerDisconnect(connection *PlayerConnect) {
	player_id := connection.player_id
	if player_id != 0 {
		data, _ := json.Marshal(nil)
		g.master_server.writeMessage(pack(PLAYER_DISCONNECT, []int{player_id}, data), nil)
		if connection.room_server != "" {
			room_server, ok := g.room_servers[connection.room_server]
			if ok {
				room_server.writeMessage(pack(PLAYER_DISCONNECT, []int{player_id}, data), nil)
			}
		}
		delete(g.players, player_id)
		connection.player_id = 0
		log.Printf("Player disconnection:", player_id)
	}
}

func (g *GatewayManager) gateway_servers(servers map[string]interface{}) {
	room_servers := servers["rooms"].(map[string]interface{})
	for server_id, server_tag := range room_servers {
		if server_tag != g.tag {
			continue
		}
		identifier := fmt.Sprintf("%s-%s", g.identifier, server_id)
		_, ok := g.room_servers[identifier]
		if !ok {
			conn := NewServerConnection(identifier, g.tag, server_id)
			g.add_room_server(conn)
			conn.connect()
		}
	}
}

func (g *GatewayManager) add_room_server(server *ServerConnection) {
	log.Printf("add_room_server", server.identifier)
	old_conn, ok := g.room_servers[server.identifier]
	if ok {
		old_conn.do_close()
	}
	g.room_servers[server.identifier] = server
}
