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
	routes   *rpc.Server
	quit     chan bool
	listener net.Listener
	pipe     *monkpipe.JSPipe

	exited bool
}

func (s *JsonRpcServer) exitHandler() {
out:
	for {
		select {
		case <-s.quit:
			s.exited = true
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
	s.routes = rpc.NewServer()
	s.routes.Register(&TheloniousApi{pipe: s.pipe})

	s.exited = false
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.exited {
				logger.Infoln("Error starting JSON-RPC:", err)
			}
			break
		}
		go s.routes.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func NewJsonRpcServer(pipe *monkpipe.JSPipe, addr string) (*JsonRpcServer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &JsonRpcServer{
		routes:   rpc.NewServer(),
		listener: l,
		quit:     make(chan bool),
		pipe:     pipe,
	}, nil
}
