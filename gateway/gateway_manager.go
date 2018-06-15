package main

import (
	"github.com/lunixbochs/struc"
	"github.com/ugorji/go-msgpack"
	"github.com/ugorji/go/codec"
	"log"
	"sync"
	"time"
)

type GatewayManager struct {
	identifier    string
	tag           string
	runtime       string
	master_server MasterServerConnection
}

var manager *GatewayManager
var once sync.Once

func Gateway() *GatewayManager {
	once.Do(func() {
		manager = &GatewayManager{}
	})
	return manager
}

func (g GatewayManager) init(runtime string, identifier string, tag string) {
	g.identifier = identifier
	g.tag = tag
	g.runtime = runtime
	go g.update(time.Second)
	//go g.Check(time.Second*5)
	//go g.Monitor(time.Second*300)
}

func (g GatewayManager) update(second time.Duration) {
	ticker := time.NewTicker(second)
	for _ = range ticker.C {
		log.Printf("GatewayManager Update")
	}
}

func (g GatewayManager) onMasterConnect(conn MasterServerConnection) {
	g.master_server = conn
}

func (g GatewayManager) onPlayerConnect(connection PlayerConnect) {
	log.Printf("onPlayerConnect")

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
		"account":   account,
		"session":   session,
		"channel":   channel,
		"device":    device,
		"open_info": open_info,
		"ip":        ip,
		"create":    create,
		"country":   country,
		"tz":        tz,
		"lang":      lang,
		"server":    g.identifier,
		"tag":       tag,
	}
	body, err := msgpack.Marshal(data)
	var buf bytes.Buffer
	header := struc.Pack(&buf, MessageHeader{"<HH", 0xf00a, 0})
	message := header + body
	g.master_server.writeMessage(message)
}

type MessageHeader struct {
	fmt   string
	cmd   int64
	count int64
}
