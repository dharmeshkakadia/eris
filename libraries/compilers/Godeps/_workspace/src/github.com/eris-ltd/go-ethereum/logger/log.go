package logger

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
)

func openLogFile(datadir string, filename string) *os.File {
	path := ethutil.AbsolutePath(datadir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("error opening log file '%s': %v", filename, err))
	}
	return file
}

func New(datadir string, logFile string, logLevel int) LogSystem {
	var writer io.Writer
	if logFile == "" {
		writer = os.Stdout
	} else {
		writer = openLogFile(datadir, logFile)
	}

	sys := NewStdLogSystem(writer, log.LstdFlags, LogLevel(logLevel))
	AddLogSystem(sys)

	return sys
}
