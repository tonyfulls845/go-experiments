package connection

import (
	"bufio"
	"encoding/binary"
	"go-experiments/internal/transport/ws/frame"
	"net"
	"sync"
)

type Connection struct {
	Conn  net.Conn
	bufrw *bufio.ReadWriter
	mx    *sync.Mutex
}

func New(Conn net.Conn, bufrw *bufio.ReadWriter, mx *sync.Mutex) *Connection {
	return &Connection{Conn, bufrw, mx}
}

func (c Connection) SendMessage(message []byte, opCode byte) {
	f := frame.New(
		true,
		opCode,
		uint64(binary.Size(message)),
		message,
	)

	c.writeFrame(f)
}

func (c *Connection) writeFrame(f *frame.Frame) {
	buf := make([]byte, 2)
	buf[0] |= byte(f.OpCode)

	if f.IsFin {
		buf[0] |= 0x80
	}

	if f.Length < 126 {
		buf[1] |= byte(f.Length)
	} else if f.Length < 1<<16 {
		buf[1] |= 126
		size := make([]byte, 2)
		binary.BigEndian.PutUint16(size, uint16(f.Length))
		buf = append(buf, size...)
	} else {
		buf[1] |= 127
		size := make([]byte, 8)
		binary.BigEndian.PutUint64(size, f.Length)
		buf = append(buf, size...)
	}
	buf = append(buf, f.Payload...)

	c.mx.Lock()
	c.bufrw.Write(buf)
	c.bufrw.Flush()
	c.mx.Unlock()
}
