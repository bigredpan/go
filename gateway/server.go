// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var addr = flag.String("addr", "localhost:18011", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", playerHandle)
	Gateway().init("dev", "127.0.0.1:18011", "sh")
	m := NewServerConnection("127.0.0.1:18011", "sh", "127.0.0.1:20011")
	m.connect()
	Gateway().master_server = m
	log.Fatal(http.ListenAndServe(*addr, nil))
}
