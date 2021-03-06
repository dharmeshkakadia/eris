package logger

import (
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

type TestLogSystem struct {
	mutex  sync.Mutex
	output string
	level  LogLevel
}

func (ls *TestLogSystem) LogPrint(level LogLevel, msg string) {
	ls.mutex.Lock()
	ls.output += msg
	ls.mutex.Unlock()
}

func (ls *TestLogSystem) SetLogLevel(i LogLevel) {
	ls.mutex.Lock()
	ls.level = i
	ls.mutex.Unlock()
}

func (ls *TestLogSystem) GetLogLevel() LogLevel {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	return ls.level
}

func (ls *TestLogSystem) CheckOutput(t *testing.T, expected string) {
	ls.mutex.Lock()
	output := ls.output
	ls.mutex.Unlock()
	if output != expected {
		t.Errorf("log output mismatch:\n   got: %q\n  want: %q\n", output, expected)
	}
}

type blockedLogSystem struct {
	LogSystem
	unblock chan struct{}
}

func (ls blockedLogSystem) LogPrint(level LogLevel, msg string) {
	<-ls.unblock
	ls.LogSystem.LogPrint(level, msg)
}

func TestLoggerFlush(t *testing.T) {
	Reset()

	logger := NewLogger("TEST")
	ls := blockedLogSystem{&TestLogSystem{level: WarnLevel}, make(chan struct{})}
	AddLogSystem(ls)
	for i := 0; i < 5; i++ {
		// these writes shouldn't hang even though ls is blocked
		logger.Errorf(".")
	}

	beforeFlush := time.Now()
	time.AfterFunc(80*time.Millisecond, func() { close(ls.unblock) })
	Flush() // this should hang for approx. 80ms
	if blockd := time.Now().Sub(beforeFlush); blockd < 80*time.Millisecond {
		t.Errorf("Flush didn't block long enough, blocked for %v, should've been >= 80ms", blockd)
	}

	ls.LogSystem.(*TestLogSystem).CheckOutput(t, "[TEST] .[TEST] .[TEST] .[TEST] .[TEST] .")
}

func TestLoggerPrintln(t *testing.T) {
	Reset()

	logger := NewLogger("TEST")
	testLogSystem := &TestLogSystem{level: WarnLevel}
	AddLogSystem(testLogSystem)
	logger.Errorln("error")
	logger.Warnln("warn")
	logger.Infoln("info")
	logger.Debugln("debug")
	Flush()

	testLogSystem.CheckOutput(t, "[TEST] error\n[TEST] warn\n")
}

func TestLoggerPrintf(t *testing.T) {
	Reset()

	logger := NewLogger("TEST")
	testLogSystem := &TestLogSystem{level: WarnLevel}
	AddLogSystem(testLogSystem)
	logger.Errorf("error to %v\n", []int{1, 2, 3})
	logger.Warnf("warn %%d %d", 5)
	logger.Infof("info")
	logger.Debugf("debug")
	Flush()
	testLogSystem.CheckOutput(t, "[TEST] error to [1 2 3]\n[TEST] warn %d 5")
}

func TestMultipleLogSystems(t *testing.T) {
	Reset()

	logger := NewLogger("TEST")
	testLogSystem0 := &TestLogSystem{level: ErrorLevel}
	testLogSystem1 := &TestLogSystem{level: WarnLevel}
	AddLogSystem(testLogSystem0)
	AddLogSystem(testLogSystem1)
	logger.Errorln("error")
	logger.Warnln("warn")
	Flush()

	testLogSystem0.CheckOutput(t, "[TEST] error\n")
	testLogSystem1.CheckOutput(t, "[TEST] error\n[TEST] warn\n")
}

func TestFileLogSystem(t *testing.T) {
	Reset()

	logger := NewLogger("TEST")
	filename := "test.log"
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	testLogSystem := NewStdLogSystem(file, 0, WarnLevel)
	AddLogSystem(testLogSystem)
	logger.Errorf("error to %s\n", filename)
	logger.Warnln("warn")
	Flush()
	contents, _ := ioutil.ReadFile(filename)
	output := string(contents)
	if output != "[TEST] error to test.log\n[TEST] warn\n" {
		t.Error("Expected contents of file 'test.log': '[TEST] error to test.log\\n[TEST] warn\\n', got ", output)
	} else {
		os.Remove(filename)
	}
}

func TestNoLogSystem(t *testing.T) {
	Reset()

	logger := NewLogger("TEST")
	logger.Warnln("warn")
	Flush()
}

func TestConcurrentAddSystem(t *testing.T) {
	rand.Seed(time.Now().Unix())
	Reset()

	logger := NewLogger("TEST")
	stop := make(chan struct{})
	writer := func() {
		select {
		case <-stop:
			return
		default:
			logger.Infoln("foo")
			Flush()
		}
	}

	go writer()
	go writer()

	stopTime := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(stopTime) {
		time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
		AddLogSystem(NewStdLogSystem(ioutil.Discard, 0, InfoLevel))
	}
	close(stop)
}
