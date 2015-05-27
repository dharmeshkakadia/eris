package server

import (
	"encoding/json"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// JSON RPC error codes.
const PARSE_ERROR = -32700
const INVALID_REQUEST = -32600
const METHOD_NOT_FOUND = -32601
const INVALID_PARAMS = -32602
const INTERNAL_ERROR = -32603

// Handler function template.
type JsonRpcHandler func(*Request, *Response)

// The websocket service handles connections.
// NOTE All sessions use the same handlers, since you can't run
// multiple EPMs on the same machine (or you can, but there's no
// guarantee that they don't try and compete for the same database).
//
// A connection's writer can only be used by one process at a time.
// To avoid any problems, no external process is allowed access
// to the websocket connection object itself. All they can do is pass
// messages via the WriteMsgChannel. Note that bro and close are not
// public, because no external process should ever use it, but those
// messages still conflict with text messages and is therefore passed
// to the write-routine in the same manner.
//
// All text messages must be json formatted strings.
type WsService struct {
	maxConnections uint32
	idPool         *IdPool
	sessions       map[uint32]*Session
	handlers       map[string]JsonRpcHandler
	sessionLock    *sync.Mutex
}

// Create a new websocket service
func NewWsService(maxConnections uint32) *WsService {
	srv := &WsService{}
	srv.sessions = make(map[uint32]*Session)
	srv.maxConnections = maxConnections
	srv.idPool = NewIdPool(maxConnections)
	srv.sessionLock = &sync.Mutex{}
	srv.handlers = make(map[string]JsonRpcHandler)
	// Register handlers here
	srv.handlers["echo"] = srv.echo
	return srv
}

/***************************** Handlers ********************************/

// Simple echo
func (this *WsService) echo(req *Request, resp *Response) {
	sVal := &StringValue{}
	err := json.Unmarshal([]byte(*req.Params), &sVal)
	if err != nil {
		resp.Error = Error(INVALID_PARAMS, "Echo requires a string parameter.")
	}
	logger.Infof("Echo: %s", sVal.Value)
	resp.Result = sVal
}

/***********************************************************************/

// Get the current number of active connections.
func (this *WsService) CurrentActiveConnections() uint32 {
	return uint32(len(this.sessions))
}

// Get the maximum number of active connections.
func (this *WsService) MaxConnections() uint32 {
	return this.maxConnections
}

// This is passed to the Martini server to handle websocket requests.
func (this *WsService) handleWs(w http.ResponseWriter, r *http.Request) {

	// TODO check scheme first.
	logger.Infoln("New websocket connection.")

	if uint32(len(this.sessions)) == this.maxConnections {
		logger.Infoln("Connection failed: Already at capacity.")
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Infof("Failed to upgrade to websocket (%s)\n", err.Error())
		return
	}

	ss := this.newSession(conn)

	go writer(ss)
	reader(ss)
	ss.writeMsgChannel <- &Message{Data: nil}
	ss.closeSession()
}

// Create a new session. Only called by the 'handleWs' function.
func (this *WsService) newSession(conn *websocket.Conn) *Session {
	this.sessionLock.Lock()
	ss := &Session{}
	ss.conn = conn
	ss.server = this
	id := this.idPool.GetId()
	ss.sessionId = id
	ss.writeMsgChannel = make(chan *Message, 256)
	ss.writeCloseChannel = make(chan *Message, 256)

	this.sessions[id] = ss
	this.sessionLock.Unlock()
	return ss
}

// Delete a session. This happens automatically after a close message
//  has been received.
func (this *WsService) deleteSession(sessionId uint32) {
	if this.sessions[sessionId] == nil {
		logger.Infof("Attempted to remove a session that does not exist (id: %d).", sessionId)
		return
	}
	this.sessionLock.Lock()
	delete(this.sessions, sessionId)
	this.idPool.ReleaseId(sessionId)
	logger.Infof("Closing session: %d", sessionId)
	logger.Infof("Connections remaining: %d", len(this.sessions))
	this.sessionLock.Unlock()
}

// Session wraps a websocket connection.
type Session struct {
	conn              *websocket.Conn
	server            *WsService
	writeMsgChannel   chan *Message
	writeCloseChannel chan *Message
	sessionId         uint32
}

// Get session ID
func (ss *Session) SessionId() uint32 {
	return ss.sessionId
}

// Write json object
func (ss *Session) WriteJson(obj interface{}) {
	msg, err := json.Marshal(obj)
	if err != nil {
		// TODO Protocol stuff.
		ss.WriteCloseMsg()
	} else {
		ss.writeMsgChannel <- &Message{Data: msg, Type: websocket.TextMessage}
	}
}

// Write close message
func (ss *Session) WriteCloseMsg() {
	ss.writeCloseChannel <- &Message{Data: nil, Type: websocket.CloseMessage}
}

// Close this websocket session.
func (ss *Session) closeSession() {
	// Deregister ourselves.
	ss.server.deleteSession(ss.sessionId)

	if ss.conn != nil {
		err := ss.conn.Close()
		if err != nil {
			logger.Infof("Failed to close websocket connection, already removed: %d\n", ss.sessionId)
		}
	}
}

// Handle an incomming request.
func (ss *Session) handleRequest(msg []byte) {

	// Start by unmarshalling into a Request object.
	req := &Request{}

	err := json.Unmarshal(msg, req)

	var resp *Response

	if err != nil {
		// Can't really say if it's bad json or not a proper request
		// without looking at the error.
		resp = ErrorResp(INVALID_REQUEST, err.Error())
	} else {
		// Get the handler for the method. If no handler exists then
		// return an error message
		handler, hExists := ss.server.handlers[req.Method]
		if !hExists {
			resp = ErrorResp(METHOD_NOT_FOUND, "Method not found: "+req.Method)
		} else {
			// Set the ID to be the same as in the request, and set json rpc to 2.0.
			resp = &Response{}
			resp.ID = req.ID
			resp.JsonRpc = "2.0"
			// Pass the request and pre-prepared response to the handler.
			// The handler needs to write the result (or error).
			handler(req, resp)
		}
	}
	// Write the response.
	ss.WriteJson(resp)
}

// Get an Error Object
func Error(code int, err string) *ErrorObject {
	return &ErrorObject{code, err}
}

// Get an error response with the fields already filled out.
func ErrorResp(code int, err string) *Response {
	return &Response{
		-1,
		"2.0",
		nil,
		&ErrorObject{code, err},
	}
}

// Basic json rpc structs
type (
	Request struct {
		ID      interface{}      `json:"id"`
		JsonRpc string           `json:"jsonrpc"`
		Method  string           `json:"method"`
		Params  *json.RawMessage `json:"params"`
	}

	Response struct {
		ID      interface{}  `json:"id"`
		JsonRpc string       `json:"jsonrpc"`
		Result  interface{}  `json:"result"`
		Error   *ErrorObject `json:"error"`
	}

	ErrorObject struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

// Other types here
type StringValue struct {
	Value string `json:"value"`
}
