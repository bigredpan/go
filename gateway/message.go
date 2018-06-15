package main

import (
	"bytes"
	"encoding/binary"
)

func pack(cmd uint16, id_list []int, body []byte) []byte {
	buf := new(bytes.Buffer)
	var a uint16 = cmd
	var b uint16 = 0
	var d uint8 = 1
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
