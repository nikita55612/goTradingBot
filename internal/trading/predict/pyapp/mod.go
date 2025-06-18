package pyapp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nikita55612/goTradingBot/internal/pkg/pyexec"
)

var (
	appAddr string = "localhost:8666"
	process *pyexec.PyProcess
	mu      sync.Mutex
)

func SetAddr(addr string) {
	mu.Lock()
	defer mu.Unlock()

	appAddr = addr
}

func SetContext(ctx context.Context) {
	go func() {
		<-ctx.Done()
		Stop()
	}()
}

func Run() error {
	mu.Lock()
	defer mu.Unlock()

	if process != nil {
		return fmt.Errorf("the process has already started")
	}

	parts := strings.Split(appAddr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid appAddr format: %s (expected 'host:port')", appAddr)
	}

	host, port := parts[0], parts[1]
	p, err := pyexec.NewPyProcess(
		"neuralab",
		pyexec.WithVenvDir("venv"),
		pyexec.WithScriptName("main.py"),
		pyexec.WithArgs("-H", host, "-P", port),
	)
	if err != nil {
		return fmt.Errorf("failed to create process: %w", err)
	}

	process = p
	if err := process.Start(); err != nil {
		process = nil
		return fmt.Errorf("failed to start process: %w", err)
	}

	timeout := time.After(time.Minute)
	for pingApp() == "" {
		time.Sleep(100 * time.Millisecond)
		select {
		case <-timeout:
			process.Stop()
			process = nil
			return fmt.Errorf("the process startup timeout was exceeded")
		default:
		}
	}

	return nil
}

func Stop() {
	mu.Lock()
	defer mu.Unlock()

	if process != nil {
		process.Stop()
		process = nil
	}
}

func Restart() error {
	mu.Lock()
	defer mu.Unlock()

	if process != nil {
		process.Stop()
		process = nil
	}
	return Run()
}
