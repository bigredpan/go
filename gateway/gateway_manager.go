package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type GatewayManager struct {
	identifier    string
	tag           string
	runtime       string
	master_server *ServerConnection
	players       *sync.Map
	room_servers  *sync.Map
	chat_server   *ServerConnection
	receive       chan MessageInfo
}

var manager *GatewayManager
var once sync.Once

func Gateway() *GatewayManager {
	once.Do(func() {
		manager = &GatewayManager{}
		manager.players = new(sync.Map)
		manager.room_servers = new(sync.Map)
		manager.receive = make(chan MessageInfo, 256)
	})
	return manager
}

func (g *GatewayManager) init(runtime string, identifier string, tag string) {
	g.identifier = identifier
	g.tag = tag
	g.runtime = runtime
	go g.process()
	go g.update(time.Second)
	//go g.Check(time.Second*5)
	//go g.Monitor(time.Second*300)
}

func (g *GatewayManager) process() {
	for {
		select {
		case info := <-g.receive:
			g.onServerMessage(info.c, info.msg, info.cid)
		}
	}
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
	// log.Printf("onServerMessage", cmd, id_list, body)
	if cmd == PLAYER_PING {

	} else if cmd == NOTICE_PING {

	} else if cmd == CENTER_GATEWAY_SERVERS {
		var data = make(map[string]interface{})
		json.Unmarshal(body, &data)
		g.gateway_servers(data)
	} else if cmd == CENTER_GATEWAY_CONFIG {
		// self.gateway_config(Serializer.loads(body))
	} else if cmd == CENTER_GATEWAY_SELF_KICK {
		g.gateway_kick(id_list[0], 4002, "self kick")
	} else if cmd == CENTER_GATEWAY_KICK {
		g.gateway_kick(id_list[0], 4007, "kick player")
	} else {
		msg := pack(cmd, nil, body)

		if cmd == NOTICE_SPEAKER || cmd == NOTICE_CONFIG_CHANGE || cmd == NOTICE_SNATCH_UPDATE {
			g.players.Range(func(key, value interface{}) bool {
				conn := value.(*PlayerConnect)
				conn.writeMessage(msg, cid)
				return true
			})
			// for _, connection := range g.players {
			// 	connection.writeMessage(msg, cid)
			// }
		} else {
			for _, player_id := range id_list {
				value, ok := g.players.Load(player_id)
				if ok {
					connection := value.(*PlayerConnect)
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

func (g *GatewayManager) onPlayerMessage(connection *PlayerConnect, msg []byte, cid []byte) {
	player_id := connection.player_id
	cmd, id_list, body := unpack(msg)
	if cmd == PLAYER_BALL {
		return
	}

	var server *ServerConnection = nil
	if cmd > CHAT_COMMAND {
		server = g.chat_server
	} else if cmd > MASTER_COMMAND {
		server = g.master_server
	} else if cmd > ROOM_COMMAND {
		if connection.room_server != "" {
			value2, ok := g.room_servers.Load(connection.room_server)
			if ok {
				server = value2.(*ServerConnection)
			}
		}
	} else {
		err, _ := json.Marshal(map[string]string{"error": "unknown-cmd"})
		connection.writeMessage(pack(cmd, id_list, err), cid)
	}
	if server != nil {
		server.writeMessage(pack(cmd, []int{player_id}, body), cid)
	}
}

func (g *GatewayManager) onPlayerConnect(connection *PlayerConnect) {
	if g.master_server.conn == nil {
		connection.do_close(4008, "server close")
	}

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
	log.Printf("onPlayerConnect", account, device)
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
		if len(id_list) == 1 {
			player_id := id_list[0]
			old_conn_value, ok := g.players.Load(player_id)
			log.Printf("old connection", ok)
			if ok {
				old_conn := old_conn_value.(*PlayerConnect)
				old_conn.do_close(4002, "self kick")
				old_conn.player_id = 0
			}
			connection.player_id = player_id
			connection.login_data = data
			g.players.Store(player_id, connection)
			if g.chat_server != nil {
				data, _ := json.Marshal(nil)
				g.chat_server.writeMessage(pack(PLAYER_CHAT_CONNECT, []int{player_id}, data), nil)
			}
			connection.writeMessage(pack(NOTICE_INIT, []int{player_id}, body), nil)
		} else {
			var data = make(map[string]interface{})
			json.Unmarshal(body, &data)
			error := data["error"].(string)
			log.Printf("onPlayerConnect error:", error)
			if error == "player-not-found" {
				connection.do_close(4005, "session failed")
			} else if error == "version-error" {
				connection.do_close(4010, "Version Error")
			} else if error == "server-offline" {
				connection.do_close(4008, "server close")
			} else if strings.HasPrefix(error, "forbid-player") {
				time_str := strings.Replace(error, "forbid-player", "", -1)
				connection.do_close(4004, time_str)
			} else {
				connection.do_close(4003, "data invalid")
			}
		}
	})
}

func (g *GatewayManager) onPlayerDisconnect(connection *PlayerConnect) {
	player_id := connection.player_id
	if player_id != 0 {
		data, _ := json.Marshal(nil)
		g.master_server.writeMessage(pack(PLAYER_DISCONNECT, []int{player_id}, data), nil)
		if connection.room_server != "" {
			value, ok := g.room_servers.Load(connection.room_server)
			if ok {
				room_server := value.(*ServerConnection)
				room_server.writeMessage(pack(PLAYER_DISCONNECT, []int{player_id}, data), nil)
			}
		}
		if g.chat_server != nil {
			g.chat_server.writeMessage(pack(PLAYER_CHAT_DISCONNECT, []int{player_id}, data), nil)
		}
		g.players.Delete(player_id)
		connection.player_id = 0
		log.Printf("Player disconnection:", player_id)
	}
}

func (g *GatewayManager) gateway_servers(servers map[string]interface{}) {
	room_servers := servers["rooms"].(map[string]interface{})

	g.room_servers.Range(func(key, value interface{}) bool {
		identifier := key.(string)
		server := value.(*ServerConnection)
		server_id := identifier[strings.Index(identifier, "-")+1:]
		log.Printf("gateway_servers server_id", server_id)
		log.Printf("gateway_servers room_servers", room_servers)
		_, ok := room_servers[server_id]
		if !ok {
			log.Printf("gateway_servers server.id", server.identifier)
			g.remove_room_server(server)
		}
		return true
	})
	// for identifier, server := range g.room_servers {
	// 	server_id := identifier[strings.Index(identifier, "-")+1:]
	// 	log.Printf("gateway_servers server_id", server_id)
	// 	log.Printf("gateway_servers room_servers", room_servers)
	// 	_, ok := room_servers[server_id]
	// 	if !ok {
	// 		log.Printf("gateway_servers server.id", server.identifier)
	// 		g.remove_room_server(server)
	// 	}
	// }

	for server_id, server_tag := range room_servers {
		if server_tag != g.tag {
			continue
		}
		identifier := fmt.Sprintf("%s-%s", g.identifier, server_id)
		_, ok := g.room_servers.Load(identifier)
		if !ok {
			conn := NewServerConnection(identifier, g.tag, server_id)
			g.add_room_server(conn)
			conn.connect()
		}
	}

	chat_server_id := servers["chat"].(string)
	if chat_server_id != "" {
		identifier := fmt.Sprintf("%s-%s", g.identifier, chat_server_id)
		if g.chat_server != nil && g.chat_server.host != chat_server_id {
			g.chat_server.do_close()
			g.chat_server = nil
		}
		if g.chat_server == nil {
			conn := NewServerConnection(identifier, g.tag, chat_server_id)
			g.chat_server = conn
			conn.connect()
		}
	}
}

func (g *GatewayManager) add_room_server(server *ServerConnection) {
	log.Printf("add_room_server", server.identifier)
	value, ok := g.room_servers.Load(server.identifier)
	if ok {
		old_conn := value.(*ServerConnection)
		old_conn.do_close()
	}
	g.room_servers.Store(server.identifier, server)
}

func (g *GatewayManager) remove_room_server(server *ServerConnection) {
	log.Printf("remove_room_server", server.identifier)
	value, ok := g.room_servers.Load(server.identifier)
	if ok {
		old_conn := value.(*ServerConnection)
		old_conn.do_close()
		g.room_servers.Delete(server.identifier)
	}
}

func (g *GatewayManager) gateway_kick(player_id int, code int, reason string) {
	log.Printf("gateway_kick", player_id, reason)
	value, ok := g.players.Load(player_id)
	if ok {
		connection := value.(*PlayerConnect)
		connection.do_close(code, reason)
	}
}

func (g *GatewayManager) player_ball(conn *PlayerConnect) {

}
