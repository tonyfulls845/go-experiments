package manager

import (
	"bufio"
	"go-experiments/internal/transport/ws/connection"
	"go-experiments/internal/utils/syncx"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Manager struct {
	connections *syncx.Map[string, *connection.Connection]
}

func (m Manager) Register(conn net.Conn, bufrw *bufio.ReadWriter) string {

	uuid := uuid.NewString()

	m.connections.Store(uuid, connection.New(conn, bufrw, &sync.Mutex{}))

	return uuid
}

func (m Manager) Unregister(uuid string) {
	connection, ok := m.connections.Load(uuid)

	if !ok {
		return
	}

	connection.Conn.Close()
	m.connections.Delete(uuid)
}

func (m Manager) SendMessage(uuid string, buf []byte, opCode byte) {
	conn, ok := m.connections.Load(uuid)
	if !ok {
		return
	}

	conn.SendMessage(buf, opCode)
}

func (m Manager) Broadcast(myUuid string, buf []byte, opCode byte) {
	m.connections.Range(func(uuid string, conn *connection.Connection) bool {
		if uuid != myUuid {
			m.SendMessage(uuid, buf, opCode)
		}

		return true
	})
}

func New() *Manager {
	connections := new(syncx.Map[string, *connection.Connection])

	return &Manager{connections}
}
