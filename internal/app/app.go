package main

import (
	"fmt"
	wsv1 "go-experiments/internal/transport/ws"
	wsv2 "go-experiments/internal/transport/ws-v2"
	managerv2 "go-experiments/internal/transport/ws-v2/manager"
	"go-experiments/internal/transport/ws/frame"
	"go-experiments/internal/transport/ws/manager"
	"net/http"
	"sync"

	"github.com/gobwas/ws"
)

func wsv1Server() {
	http.HandleFunc("/ws", wsv1.GetHTTPHandler(func(m *manager.Manager, uuid string, buf []byte) {
		m.Broadcast(uuid, buf, frame.TextFrame)
	}))

	fmt.Printf("Start listening 8002\n")
	http.ListenAndServe(":8002", nil)
}

func wsv2Server() {
	wsv2.Listen("localhost:8001", func(m *managerv2.Manager, uuid string, payload []byte) {
		m.Broadcast(uuid, payload, ws.OpText)
	})
}

func main() {
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		wsv1Server()
	}()

	go func() {
		defer wg.Done()
		wsv2Server()
	}()

	wg.Wait()
}
