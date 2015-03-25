package monkrpc

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkpipe"
)

var logger = monklog.NewLogger("JSON")

type JsonRpcServer struct {
	quit     chan bool
	listener net.Listener
	pipe     *monkpipe.JSPipe
}

func (s *JsonRpcServer) exitHandler() {
out:
	for {
		select {
		case <-s.quit:
			s.listener.Close()
			break out
		}
	}

	logger.Infoln("Shutdown JSON-RPC server")
}

func (s *JsonRpcServer) Stop() {
	close(s.quit)
}

func (s *JsonRpcServer) Start() {
	logger.Infoln("Starting JSON-RPC server")
	go s.exitHandler()
	rpc.Register(&TheloniousApi{pipe: s.pipe})
	rpc.HandleHTTP()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			logger.Infoln("Error starting JSON-RPC:", err)
			break
		}
		go jsonrpc.ServeConn(conn)
	}
}

func NewJsonRpcServer(pipe *monkpipe.JSPipe, addr string) (*JsonRpcServer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &JsonRpcServer{
		listener: l,
		quit:     make(chan bool),
		pipe:     pipe,
	}, nil
}
