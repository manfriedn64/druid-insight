package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger structure simple (thread-safe pour notre usage)
type Logger struct {
	file *os.File
}

// NewLogger crée (et ouvre en append) un logger fichier
func NewLogger(dir, fname string) (*Logger, error) {
	if dir == "" {
		dir = "./logs"
	}
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, fname), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

// NewLoggerOrDie (pour main.go, pour moins de boilerplate)
func NewLoggerOrDie(dir, fname string) *Logger {
	l, err := NewLogger(dir, fname)
	if err != nil {
		panic(err)
	}
	return l
}

// Write ajoute une ligne datée au log
func (l *Logger) Write(msg string) {
	t := time.Now().Format("2006-01-02 15:04:05")
	l.file.WriteString(fmt.Sprintf("%s %s\n", t, msg))
}

// Close ferme le fichier log proprement
func (l *Logger) Close() {
	l.file.Close()
}
