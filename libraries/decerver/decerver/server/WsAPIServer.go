package server

import (
	"fmt"
	"github.com/eris-ltd/decerver/interfaces/scripting"
	"github.com/eris-ltd/decerver/util"
	"github.com/gorilla/websocket"
	"net/http"
	"path"
)

// The websocket server handles connections.
type WsAPIServer struct {
	rm                scripting.RuntimeManager
	activeConnections uint32
	maxConnections    uint32
	idPool            *util.IdPool
	sessions          map[uint32]*Session
}

func NewWsAPIServer(rm scripting.RuntimeManager, maxConnections uint32) *WsAPIServer {
	srv := &WsAPIServer{}
	srv.sessions = make(map[uint32]*Session)
	srv.maxConnections = maxConnections
	srv.idPool = util.NewIdPool(maxConnections)
	srv.rm = rm
	return srv
}

func (srv *WsAPIServer) CurrentActiveConnections() uint32 {
	return srv.activeConnections
}

func (srv *WsAPIServer) MaxConnections() uint32 {
	return srv.maxConnections
}

func (srv *WsAPIServer) RemoveSession(ss *Session) {
	srv.activeConnections--
	srv.idPool.ReleaseId(ss.wsConn.SessionId())
	delete(srv.sessions, ss.wsConn.SessionId())
}

func (srv *WsAPIServer) CreateSession(caller string, rt scripting.Runtime, wsConn *WsConn) *Session {
	ss := &Session{}
	ss.wsConn = wsConn
	ss.server = srv
	ss.caller = caller
	ss.runtime = rt
	srv.activeConnections++
	id := srv.idPool.GetId()
	ss.wsConn.sessionId = id
	srv.sessions[id] = ss
	return ss
}

// This is passed to the Martini server.
// Find out what endpoint they called and create a session based on that.
func (srv *WsAPIServer) handleWs(w http.ResponseWriter, r *http.Request) {
	logger.Println("New websocket connection registering.")
	if srv.activeConnections == srv.maxConnections {
		logger.Println("Connection failed: Already at capacity.")
	}
	u := r.URL
	p := u.Path
	caller := path.Base(p)
	rt := srv.rm.GetRuntime(caller)
	// TODO Update this. It's basically how we check if dapp is ready now.
	if rt == nil {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Dapp not in focus")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Printf("Failed to upgrade to websocket (%s)\n", err.Error())
		return
	}
	// TODO buffering...
	wsConn := &WsConn{
		conn:              conn,
		writeMsgChannel:   make(chan *Message, 256),
		writeCloseChannel: make(chan *Message, 256),
	}

	ss := srv.CreateSession(caller, rt, wsConn)
	// We add this session to the callers (dapps) runtime.
	err = rt.BindScriptObject("tempObj", NewSessionJs(ss))

	if err != nil {
		panic(err.Error())
	}

	// TODO fix...
	rt.AddScript("tempObj.sessionId = function(){return this.SessionId()};tempObj.writeJson = function(data){return this.WriteJson(data)};network.newWsSession(tempObj); tempObj = null;")
	//rt.CallFuncOnObj("network", "newWsSession", val)
	go writer(ss)
	reader(ss)
	ss.wsConn.writeMsgChannel <- &Message{Data: nil}
	ss.Close()
}

type Session struct {
	caller    string
	runtime   scripting.Runtime
	server    *WsAPIServer
	wsConn    *WsConn
	sessionJs *SessionJs
}

func (ss *Session) SessionId() uint32 {
	return ss.wsConn.sessionId
}

func (ss *Session) WriteCloseMsg() {
	ss.wsConn.WriteCloseMsg()
}

func (ss *Session) Close() {
	logger.Printf("CLOSING SESSION: %d\n", ss.wsConn.sessionId)
	// Deregister ourselves.
	ss.server.RemoveSession(ss)
	ss.runtime.CallFuncOnObj("network", "deleteWsSession", int(ss.SessionId()))
	if ss.wsConn.conn != nil {
		err := ss.wsConn.conn.Close()
		if err != nil {
			logger.Printf("Failed to close websocket connection, already removed: %d\n", ss.wsConn.sessionId)
		}
	}
}

func (ss *Session) handleRequest(rpcReq string) {
	logger.Println("RPC Message: " + rpcReq)
	ret, err := ss.runtime.CallFuncOnObj("network", "incomingWsMsg", int(ss.wsConn.sessionId), rpcReq)

	if err != nil {
		logger.Printf("Js runtime error, could not pass message. Closing socket. (sesion: %d)\nMessage dump: %s\n", ss.SessionId(), rpcReq)
		ss.wsConn.writeCloseChannel <- GetCloseMessage()
		return
	}
	if ret == nil {
		return
	}
	retStr := ret.(string)
	// If there is a return value, pass to the write channel.
	ss.wsConn.writeMsgChannel <- &Message{Data: []byte(retStr), Type: websocket.TextMessage}
}

type SessionJs struct {
	session *Session
}

func NewSessionJs(ss *Session) *SessionJs {
	return &SessionJs{ss}
}

func (sjs *SessionJs) WriteJson(msg string) {
	sjs.session.wsConn.WriteJsonMsg([]byte(msg))
}

func (sjs *SessionJs) SessionId() int {
	return int(sjs.session.SessionId())
}
