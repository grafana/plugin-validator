package main

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
	file   *os.File
	prefix string
}

func NewLogger(path, prefix string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	return &Logger{file: f, prefix: prefix}, nil
}

func (l *Logger) Info(msg string) {
	l.write("INFO", msg)
}

func (l *Logger) Error(msg string) {
	l.write("ERROR", msg)
}

func (l *Logger) write(level, msg string) {
	ts := time.Now().Format(time.RFC3339)
	fmt.Fprintf(l.file, "%s [%s] %s: %s\n", ts, level, l.prefix, msg)
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
