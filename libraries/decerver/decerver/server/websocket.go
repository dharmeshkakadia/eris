// This file contains code for handling websocket specific stuff, such as
// message types, connection objects and channels, and settings. It is the
// bridge between the server and the SRPC handling code.
package server

import (
	"github.com/gorilla/websocket"
	"io/ioutil"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next 'down' message from the peer.
	downWait = 60 * time.Second

	// Send 'bro's to peer with this period. Must be less than downWait.
	pingPeriod = (downWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
)

// TODO need to look over this and maxMessageSize
var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
}

// Base message type we pass to writer. Text, Bro and close.
type Message struct {
	Data []byte
	Type int
}

// A connection's writer can only be used by one process at a time.
// To avoid any problems, no external process is allowed access
// to the websocket connection object itself. All they can do is pass
// messages via the WriteMsgChannel that the WsConn 'middle-man' provides.
// Note that bro and close is not public, because no external process
// should ever use it, but those messages still conflict with text
// messages and must therefore be passed to the write-routine in the
// same manner.
//
// All text messages must be json formatted strings.

func GetBroMessage() *Message {
	return &Message{Type: websocket.PingMessage}
}

func GetCloseMessage() *Message {
	return &Message{Type: websocket.CloseMessage}
}

type WsConn struct {
	sessionId         uint32
	conn              *websocket.Conn
	writeMsgChannel   chan *Message
	writeCloseChannel chan *Message
	// writeBroChannel   chan *Message
}

func (wc *WsConn) SessionId() uint32 {
	return wc.sessionId
}

func (wc *WsConn) Connection() *websocket.Conn {
	return wc.conn
}

func (wc *WsConn) WriteJsonMsg(msg []byte) {
	wc.writeMsgChannel <- &Message{Data: msg, Type: websocket.TextMessage}
}

func (wc *WsConn) WriteCloseMsg() {
	wc.writeCloseChannel <- &Message{Data: nil, Type: websocket.CloseMessage}
}

// Handle the reader
func reader(ss *Session) {
	conn := ss.wsConn.conn

	conn.SetReadLimit(maxMessageSize)
	// TODO add for hosted. No need for 'bro -> down' packets when only on localhost...
	//wsc.conn.SetReadDeadline(time.Now().Add(downWait))
	conn.SetReadDeadline(time.Time{})
	//wsc.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(downWait)); return nil })

	// TODO add back the reader timeout?
	for {

		mType, message, err := conn.NextReader()

		if err != nil {
			ss.wsConn.writeCloseChannel <- GetCloseMessage()
			return
		}

		if mType == websocket.TextMessage {
			rpcReq, _ := ioutil.ReadAll(message)
			ss.handleRequest(string(rpcReq))
		} else if mType == websocket.CloseMessage {
			return
		}

	}
}

// Handle the writer
func writer(ss *Session) {
	conn := ss.wsConn.conn
	for {
		message, ok := <-ss.wsConn.writeMsgChannel

		if !ok {
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if message.Data == nil {
			return
		}

		if message.Type == websocket.CloseMessage {
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, message.Data); err != nil {
			return
		}

	}
}
