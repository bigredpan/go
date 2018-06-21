package main

import (
	"bytes"
	"encoding/binary"
	// "log"
	// "math/rand"
)

var SERVER_CID_TAG = []byte{0xfe}
var SERVER_CID_TAG_SIZE = len(SERVER_CID_TAG)
var SERVER_CID_SIZE = 16
var SERVER_CID_HEADER_SIZE = SERVER_CID_TAG_SIZE + SERVER_CID_SIZE
var HEADER_SIZE = 2 + 2

var PLAYER_CID_TAG = []byte{0xff}
var PLAYER_CID_TAG_SIZE = len(PLAYER_CID_TAG)
var PLAYER_CID_SIZE = 2
var PLAYER_CID_HEADER_SIZE = PLAYER_CID_TAG_SIZE + PLAYER_CID_SIZE

func pack(cmd uint16, id_list []int, body []byte) []byte {
	buf := new(bytes.Buffer)
	var a uint16 = cmd
	var b uint16 = 0
	if id_list != nil {
		b = uint16(len(id_list))
	}
	var d uint8 = 1
	binary.Write(buf, binary.LittleEndian, a)
	binary.Write(buf, binary.LittleEndian, b)
	if b > 0 {
		for i := 0; i < int(b); i++ {
			var id int32 = int32(id_list[i])
			binary.Write(buf, binary.LittleEndian, id)
		}
	}
	binary.Write(buf, binary.LittleEndian, d)
	buf.Write(body)
	return buf.Bytes()
}

func unpack(data []byte) (cmd uint16, id_list []int, body []byte) {
	rd := bytes.NewReader(data)
	buf := make([]byte, 2)
	var index int64 = 0
	n, _ := rd.ReadAt(buf, index)
	index += int64(n)
	cmd = binary.LittleEndian.Uint16(buf)
	n, _ = rd.ReadAt(buf, index)
	index += int64(n)
	count := int(binary.LittleEndian.Uint16(buf))
	buf2 := make([]byte, 4)
	id_list = make([]int, count)
	for i := 0; i < count; i++ {
		n, _ = rd.ReadAt(buf2, index)
		iden := int(binary.LittleEndian.Uint32(buf2))
		id_list[i] = iden
		index += int64(n)
	}
	index += 1
	body = data[index:]
	return cmd, id_list, body
}
