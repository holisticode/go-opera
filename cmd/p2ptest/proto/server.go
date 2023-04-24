package proto

import (
	"fmt"
	"bytes"
	"errors"
	"net"
	"net/http"
	"sync"
)

type sender interface {
	Send(msg []byte) error
}

type receiver interface {
	Receive() (chan []byte, error)
}

const (
	contentType = "application/json"
)

var (
	_ sender   = &defaultSender{}
	_ receiver = &defaultReceiver{}

	ErrSendFailed = errors.New("sending message failed")
)

type defaultSender struct {
	urls []string
}

func NewSender(urls []string) *defaultSender {
	return &defaultSender{
		urls: urls,
	}
}

func (s *defaultSender) Send(msg []byte) error {
	reader := bytes.NewReader(msg)
	http.Post(s.urls[0], contentType, reader)
	return nil
}

type defaultReceiver struct {
	wg sync.WaitGroup
	listener net.Listener
	msg chan []byte
}

func (r *defaultReceiver) Receive() (chan []byte, error) {
	return r.msg, nil
 }

func NewReceiver() (*defaultReceiver, error) {
	var err error
	r := &defaultReceiver{
		msg:  make(chan []byte),
	}
	r.listener, err = net.Listen("tcp", "0.0.0.0:30303")
	if err != nil {
		return nil, err
	}
	// wg.Add(1)
	go r.listenLoop()
	return r, nil
}

func (r *defaultReceiver) listenLoop() {
	for {
        // Listen for an incoming connection
        conn, err := r.listener.Accept()
        if err != nil {
            panic(err)
        }
        // Handle connections in a new goroutine
        go func(conn net.Conn) {
            buf := make([]byte, 1024)
            _, err := conn.Read(buf)
            if err != nil {
                fmt.Printf("Error reading: %#v\n", err)
                return
            }
			r.msg <- buf
            conn.Close()
        }(conn)
    }
}
