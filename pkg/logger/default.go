package logger

import (
	"log"
	"os"
)

type Default struct {
	log *log.Logger
}

func NewDefault() *Default {
	log := log.New(os.Stdout, "", log.LstdFlags)
	return &Default{
		log: log,
	}
}

func (l *Default) Infof(format string, args ...interface{}) {
	l.log.Printf(format, args...)
}

func (l *Default) Errorf(format string, args ...interface{}) {
	l.log.Printf(format, args...)
}

func (l *Default) Debugf(format string, args ...interface{}) {
	l.log.Printf(format, args...)
}
