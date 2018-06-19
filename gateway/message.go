package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/rand"
)

var SERVER_CID_TAG = []byte{0xfe}
var SERVER_CID_TAG_SIZE = len(SERVER_CID_TAG)
var SERVER_CID_SIZE = 16
var SERVER_CID_HEADER_SIZE = SERVER_CID_TAG_SIZE + SERVER_CID_SIZE
var HEADER_SIZE = 2 + 2

func pack_cid(cmd uint16, id_list []int, body []byte) []byte {
	cid := make([]byte, 16)
	rand.Read(cid)
	log.Printf("pack_cid:", cid)
	return _pack(cmd, id_list, body, cid)
}

func pack(cmd uint16, id_list []int, body []byte) []byte {
	return _pack(cmd, id_list, body, nil)
}

func _pack(cmd uint16, id_list []int, body []byte, cid []byte) []byte {
	buf := new(bytes.Buffer)
	var a uint16 = cmd
	var b uint16 = 0
	var d uint8 = 1
	if cid != nil {
		buf.Write(SERVER_CID_TAG)
		buf.Write(cid)
	}
	binary.Write(buf, binary.LittleEndian, a)
	binary.Write(buf, binary.LittleEndian, b)
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
	return cmd, nil, body
}
