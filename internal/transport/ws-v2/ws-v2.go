package wsv2

import (
	"fmt"
	"go-experiments/internal/transport/ws-v2/manager"
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
)

type Callback func(m *manager.Manager, uuid string, payload []byte)

func Listen(address string, cb Callback) {
	fmt.Printf("Start %s\n", address)
	ln, err := net.Listen("tcp", address)

	m := manager.New()

	if err != nil {
		return
	}
	e := wsflate.Extension{
		// We are using default parameters here since we use
		// wsflate.{Compress,Decompress}Frame helpers below in the code.
		// This assumes that we use standard compress/flate package as flate
		// implementation.
		Parameters: wsflate.DefaultParameters,
	}
	u := ws.Upgrader{
		Negotiate: e.Negotiate,
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		// Reset extension after previous upgrades.
		e.Reset()

		_, err = u.Upgrade(conn)
		if err != nil {
			log.Printf("upgrade error: %s", err)
			continue
		}
		if _, ok := e.Accepted(); !ok {
			log.Printf("didn't negotiate compression for %s", conn.RemoteAddr())
			conn.Close()
			continue
		}

		uuid := m.Register(conn)

		go func() {
			defer conn.Close()

			var message []byte
			for {
				f, err := ws.ReadFrame(conn)
				if err != nil {
					// Handle error.
					return
				}

				f = ws.UnmaskFrameInPlace(f)

				isCompressed, err := wsflate.IsCompressed(f.Header)
				if err != nil {
					// Handle error.
					return
				}

				if isCompressed {
					// Note that even after successful negotiation of
					// compression extension, both sides are able to send
					// non-compressed messages.
					f, err = wsflate.DecompressFrame(f)

					if err != nil {
						// Handle error.
						return
					}
				}

				message = append(message, f.Payload...)

				if f.Header.OpCode == ws.OpClose {
					return
				} else if f.Header.Fin {
					cb(m, uuid, message)
					message = message[:0]
				}
			}
		}()
	}
}
