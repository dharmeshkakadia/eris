package p2p

import (
	"testing"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/logger"
)

type testLogger struct{ t *testing.T }

func testlog(t *testing.T) testLogger {
	logger.Reset()
	l := testLogger{t}
	logger.AddLogSystem(l)
	return l
}

func (testLogger) GetLogLevel() logger.LogLevel { return logger.DebugLevel }
func (testLogger) SetLogLevel(logger.LogLevel)  {}

func (l testLogger) LogPrint(level logger.LogLevel, msg string) {
	l.t.Logf("%s", msg)
}

func (testLogger) detach() {
	logger.Flush()
	logger.Reset()
}
