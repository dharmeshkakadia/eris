package logging

import (
	"log"
	"os"
)

func NewLogger(name string) *log.Logger {
	return log.New(os.Stdout, "["+name+"] ", log.LstdFlags)
}
