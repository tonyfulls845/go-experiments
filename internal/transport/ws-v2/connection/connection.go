package connection

import (
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
)

type Connection struct {
	Conn net.Conn
	mx   *sync.Mutex
}

func New(Conn net.Conn, mx *sync.Mutex) *Connection {
	return &Connection{Conn, mx}
}

func (c Connection) SendMessage(message []byte, opCode ws.OpCode) {
	ack := ws.NewFrame(opCode, true, message)

	// Compress response unconditionally.
	var err error
	ack, err = wsflate.CompressFrame(ack)
	if err != nil {
		// Handle error.
		return
	}
	c.mx.Lock()
	if err = ws.WriteFrame(c.Conn, ack); err != nil {
		// Handle error.
		return
	}
	c.mx.Unlock()
}
