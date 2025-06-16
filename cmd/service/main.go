package main

import (
	"druid-insight/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var (
	pidPath = filepath.Join(utils.GetProjectRoot(), "pid")
	pidFile = filepath.Join(pidPath, "druid-insight.pid")
	binFile = filepath.Join(utils.GetProjectRoot(), "bin", "druid-insight")
)

func main() {
	_ = utils.EnsureDirExists(pidPath)
	if len(os.Args) < 2 {
		fmt.Println("Usage: service start|stop|reload")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "start":
		start()
	case "stop":
		stop()
	case "reload":
		reload()
	case "restart":
		stop()
		time.Sleep(1 * time.Second)
		start()
	default:
		fmt.Println("Usage: service start|stop|reload|restart")
		os.Exit(1)
	}
}

func start() {
	if _, err := os.Stat(pidFile); err == nil {
		fmt.Println("druid-insight already running!")
		return
	}
	cmd := exec.Command(binFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Println("Failed to start:", err)
		os.Exit(1)
	}
	os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	fmt.Printf("druid-insight started, pid=%d\n", cmd.Process.Pid)
}

func stop() {
	pid, err := readPID()
	if err != nil {
		fmt.Println("Not running")
		return
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		fmt.Println("Failed to stop:", err)
	}
	os.Remove(pidFile)
	fmt.Println("druid-insight stopped.")
}

func reload() {
	pid, err := readPID()
	if err != nil {
		fmt.Println("Not running")
		return
	}
	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		fmt.Println("Failed to reload:", err)
	} else {
		fmt.Println("druid-insight reloaded.")
	}
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}
