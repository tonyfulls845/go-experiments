package manager

import (
	"go-experiments/internal/transport/ws-v2/connection"
	"go-experiments/internal/utils/syncx"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/google/uuid"
)

type Manager struct {
	connections *syncx.Map[string, *connection.Connection]
}

func (m Manager) Register(conn net.Conn) string {

	uuid := uuid.NewString()

	m.connections.Store(uuid, connection.New(conn, &sync.Mutex{}))

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

func (m Manager) SendMessage(uuid string, buf []byte, opCode ws.OpCode) {
	conn, ok := m.connections.Load(uuid)
	if !ok {
		return
	}

	conn.SendMessage(buf, opCode)
}

func (m Manager) Broadcast(myUuid string, buf []byte, opCode ws.OpCode) {
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
