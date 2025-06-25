package utils

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func GetProjectRoot() string {
	if env := os.Getenv("DRUID_INSIGHT_ROOT"); env != "" {
		return env
	}
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable: %v", err)
	}
	dir := filepath.Dir(executable)
	return filepath.Clean(filepath.Join(dir, ".."))
}

func EnsureDirExists(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func LogToFile(filename string) *os.File {
	EnsureDirExists(filepath.Join(GetProjectRoot(), "log"))
	log_file_name := filepath.Join(GetProjectRoot(), "log", filename)
	_, err := os.Stat(log_file_name)
	// if log file exist, move it to archive and rename
	if err == nil {
		EnsureDirExists(filepath.Join(GetProjectRoot(), "log", "archives"))
		os.Rename(log_file_name, filepath.Join(GetProjectRoot(), "log", "archives", filename+"."+time.Now().Format("2006-01-02-15-04-05")))
	}

	log_file, err := os.OpenFile(log_file_name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(log_file)
	log.SetOutput(mw)
	return log_file
}
