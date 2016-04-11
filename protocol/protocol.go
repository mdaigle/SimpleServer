package protocol

import(
	"bytes"
	"encoding/binary"
	"fmt"
)

const HELLO = 0
const DATA = 1
const ALIVE = 2
const GOODBYE = 3
const MAGIC = 0xC461
const VERSION = 1

type P0Pmessage struct  {
	// Should always be 0xC461
	Magic uint16
	// Should always be 1
	Version uint8
	// 0 is HELLO, 1 is DATA, 2 is ALIVE, and 3 is GOODBYE
	Command uint8
	// specifies place in packet sequence
	Sequencenumber uint32
	// Unique identifier for each client with an open connection
	Sessionid uint32
	Data [256]byte
}

func Encode(message P0Pmessage) (*bytes.Buffer) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, &message)
	if err != nil {
		fmt.Println("Encoding of message failed.")
	}
	return buf
}

func Decode(buf []byte) (P0Pmessage) {
	message := P0Pmessage{}
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &message)
	if err != nil {
		//fmt.Println("Decoding of message failed.")
	}
	return message
}