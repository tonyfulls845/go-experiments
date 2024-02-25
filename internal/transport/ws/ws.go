package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"go-experiments/internal/transport/ws/frame"
	"go-experiments/internal/transport/ws/manager"
	"io"
	"net"
	"net/http"
)

type ErrorType string

const (
	WrongHeaders       ErrorType = "wrong headers"
	NotSupportHijacker ErrorType = "not support hijacker"
	ReadBufError       ErrorType = "read buffer error"
)

type GetHandlerCallback func(m *manager.Manager, uuid string, buf []byte)

func upgrade(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, error) {
	// проверяем заголовки
	if r.Header.Get("Upgrade") != "websocket" {
		return nil, nil, errors.New(string(WrongHeaders))
	}
	if r.Header.Get("Connection") != "Upgrade" {
		return nil, nil, errors.New(string(WrongHeaders))
	}
	k := r.Header.Get("Sec-Websocket-Key")
	if k == "" {
		return nil, nil, errors.New(string(WrongHeaders))
	}

	// вычисляем ответ
	sum := k + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hash := sha1.Sum([]byte(sum))
	str := base64.StdEncoding.EncodeToString(hash[:])

	// Берем под контроль соединение https://pkg.go.dev/net/http#Hijacker
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New(string(WrongHeaders))
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return conn, bufrw, err
	}

	// формируем ответ
	bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	bufrw.WriteString("Upgrade: websocket\r\n")
	bufrw.WriteString("Connection: Upgrade\r\n")
	bufrw.WriteString("Sec-Websocket-Accept: " + str + "\r\n\r\n")
	bufrw.Flush()

	return conn, bufrw, nil
}

type OpCode byte

func readFrame(bufrw *bufio.ReadWriter) (frame.Frame, error) {

	frame := frame.Frame{}

	// заголовок состоит из 2 — 14 байт
	buf := make([]byte, 2, 14)
	// читаем первые 2 байта
	_, err := io.ReadFull(bufrw, buf)
	if err != nil {
		return frame, err
	}

	frame.IsFin = buf[0]>>7 == 1 // фрагментированное ли сообщение
	frame.OpCode = buf[0] & 0xf  // опкод

	maskBit := buf[1] >> 7 // замаскированы ли данные

	// оставшийся размер заголовка
	extra := 0
	if maskBit == 1 {
		extra += 4 // +4 байта маскировочный ключ
	}

	size := uint64(buf[1] & 0x7f)
	if size == 126 {
		extra += 2 // +2 байта размер данных
	} else if size == 127 {
		extra += 8 // +8 байт размер данных
	}

	if extra > 0 {
		// читаем остаток заголовка extra <= 12
		buf = buf[:extra]
		_, err := io.ReadFull(bufrw, buf)
		if err != nil {
			return frame, err
		}

		if size == 126 {
			size = uint64(binary.BigEndian.Uint16(buf[:2]))
			buf = buf[2:] // подвинем начало буфера на 2 байта
		} else if size == 127 {
			size = uint64(binary.BigEndian.Uint64(buf[:8]))
			buf = buf[8:] // подвинем начало буфера на 8 байт
		}
	}

	// маскировочный ключ
	var mask []byte
	if maskBit == 1 {
		// остаток заголовка, последние 4 байта
		mask = buf
	}

	// данные фрейма
	frame.Payload = make([]byte, int(size))
	// читаем полностью и ровно size байт
	_, err = io.ReadFull(bufrw, frame.Payload)
	if err != nil {
		return frame, err
	}

	// размаскировываем данные с помощью XOR
	if maskBit == 1 {
		for i := 0; i < len(frame.Payload); i++ {
			frame.Payload[i] ^= mask[i%4]
		}
	}

	return frame, nil
}

func GetHTTPHandler(cb GetHandlerCallback) func(w http.ResponseWriter, r *http.Request) {
	m := manager.New()

	return func(w http.ResponseWriter, r *http.Request) {
		conn, bufrw, err := upgrade(w, r)
		if err != nil {
			return
		}
		uuid := m.Register(conn, bufrw)
		defer m.Unregister(uuid)

		var message []byte
		for {
			f, err := readFrame(bufrw)

			if err != nil {
				return
			}

			message = append(message, f.Payload...)

			if f.OpCode == frame.ConnectionCloseFrame {
				return
			} else if f.IsFin {
				cb(m, uuid, message)
				message = message[:0]
			}
		}
	}
}
