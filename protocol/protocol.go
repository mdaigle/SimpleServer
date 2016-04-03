package protocol

import(
	"bytes"
	"encoding/binary"
	"fmt"
)

type p0pmessage struct  {
	// Should always be 0xC461
	magic int16
	// Should always be 1
	version int8
	// 0 is HELLO, 1 is DATA, 2 is ALIVE, and 3 is GOODBYE
	command int8
	// specifies place in packet sequence
	sequencenumber int32
	// Unique identifier for each client with an open connection
	sessionid int32
	data [256]byte
}

func encode(message p0pmessage) ([]byte) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, &message)
	if err != nil {
		fmt.Println("Encoding of message failed.")
	}
	return buf
}

func decode(buf []byte) (p0pmessage) {
	message := p0pmessage{}
	err := binary.Read(buf, binary.LittleEndian, &message)
	if err != nil {
		fmt.Println("Decoding of message failed.")
	}
	return message
}