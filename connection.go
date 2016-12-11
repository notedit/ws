package ws

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SendTimeout = 5 * time.Second

	CloseRecvTimeout = 5 * time.Second

	HandshakeTimeout = 5 * time.Second
)

// ErrCloseSent is returned by Send if sending to a connection that is closing.
var ErrCloseSent = websocket.ErrCloseSent

type sendReq struct {
	msg    interface{}
	result chan<- error
}

type recvMsg struct {
	msg []byte
	err error
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: HandshakeTimeout,
	CheckOrigin:      func(*http.Request) bool { return true },
}

type Conn struct {
	id             string
	conn           *websocket.Conn
	send           chan sendReq
	recv           chan recvMsg
	sendCloseOnce  sync.Once
	sendCloseError error
	// this conn can hold some data
	data map[string]string
    hub            *Hub
}

func NewConn(id string, hub *Hub, w http.ResponseWriter, req *http.Request) (*Conn, error) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return nil, err
	}
	return newConn(conn, hub), nil
}

func newConn(conn *websocket.Conn, hub *Hub) *Conn {
	c := &Conn{
		conn: conn,
		send: make(chan sendReq, 10),
		recv: make(chan recvMsg, 10),
        data: make(map[string]string),
        hub:  hub,
	}
	go c.goSend()
	go c.goRecv()
	return c
}

func (c *Conn) Close() error {
	close(c.send)

	err := c.sendClose()
	timer := time.NewTimer(CloseRecvTimeout)
	if err != nil {
		timer.Stop()
		c.conn.Close()
	}

	for {
		select {
		case _, ok := <-c.recv:
			if !ok {
				if timer.Stop() {
					err = c.conn.Close()
				}
				return err
			}
		case <-timer.C:
			err = c.conn.Close()
		}
	}
}

func (c *Conn) Send(msg interface{}) error {
	result := make(chan error)
	c.send <- sendReq{msg: msg, result: result}
	return <-result
}

func (c *Conn) goSend() {
	for req := range c.send {
		dl := time.Now().Add(SendTimeout)
		c.conn.SetWriteDeadline(dl)
		err := c.conn.WriteJSON(req.msg)
		req.result <- err
	}
}

func (c *Conn) Recv() ([]byte, error) {
	r, ok := <-c.recv
	if !ok {
		return nil, io.EOF
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.msg, nil
}

func (c *Conn) goRecv() {
	defer close(c.recv)

	for {
		messageType, msg, err := c.conn.ReadMessage()
		if messageType == websocket.TextMessage {
			c.recv <- recvMsg{msg: msg, err: err}
		}
		if err != nil {
			// If this errors, a subsequent call to Close will return the error.
			c.sendClose()
			// ReadMessage cannot receive messages after it returns an error.
			// So give up on waiting for a Close from the peer.
			return
		}
	}
}

func (c *Conn) sendClose() error {
	c.sendCloseOnce.Do(func() {
		dl := time.Now().Add(SendTimeout)
		c.sendCloseError = c.conn.WriteControl(websocket.CloseMessage, nil, dl)
		// If we receive a Close from the peer,
		// gorilla will send the Close response for us.
		// We don't bother tracking this, so just ignore this. error.
		if c.sendCloseError == websocket.ErrCloseSent {
			c.sendCloseError = nil
		}
	})
	return c.sendCloseError
}

func (c *Conn) Set(key string, value string) {
	c.data[key] = value
}

func (c *Conn) Get(key string) string {
	return c.data[key]
}

func (c *Conn) Hub() *Hub {
    return c.hub
}
