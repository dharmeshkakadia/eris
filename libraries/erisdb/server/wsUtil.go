// This file contains code for handling websocket specific stuff, such as
// message types, connection objects and channels, and settings. It is the
// bridge between the server and the SRPC handling code.
package server

import (
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"time"
)

const (

	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next 'down' message from the peer.
	downWait = 180 * time.Second

	// Send 'bro's to peer with this period. Must be less than downWait.
	pingPeriod = (downWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
)

// TODO need to look over this and maxMessageSize
var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Base message type we pass to writer. Text, Bro and close.
type Message struct {
	Data []byte
	Type int
}

func GetBroMessage() *Message {
	return &Message{Type: websocket.PingMessage}
}

func GetCloseMessage() *Message {
	return &Message{Type: websocket.CloseMessage}
}

// Handle the reader
func reader(ss *Session) {
	conn := ss.conn

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Time{})

	// TODO re-add maybe
	//conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(downWait)); return nil })

	for {

		mType, message, err := conn.NextReader()

		if err != nil {
			ss.writeCloseChannel <- GetCloseMessage()
			return
		}

		if mType == websocket.TextMessage {
			rpcReq, err := ioutil.ReadAll(message)
			if err != nil {
				logger.Errorf("Error: " + err.Error())
			} else {
				ss.handleRequest(rpcReq)
			}
		} else if mType == websocket.CloseMessage {
			logger.Infoln("Receiving close message")
			return
		}

	}
}

// Handle the writer
func writer(ss *Session) {
	conn := ss.conn
	for {
		message, ok := <-ss.writeMsgChannel

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
