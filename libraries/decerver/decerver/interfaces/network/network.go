package network

import (
	"github.com/eris-ltd/decerver/interfaces/dapps"		
)

// Websocket session
type WsSession interface {
	SessionId() uint32
	WriteJsonMsg(msg interface{})
	WriteCloseMsg()
}

// Webserver
type Server interface {
	AddDappManager(dapps.DappManager)
	RegisterDapp(dappId string)
	Start() error
}
