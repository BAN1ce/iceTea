package service

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Run(ctx context.Context) error {
	logrus.SetLevel(logrus.DebugLevel)
	return nil
}

func (l *Logger) Stop() error {
	return nil
}

func (l *Logger) Name() string {
	return `logger`
}
