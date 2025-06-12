package pyexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// status представляет состояние процесса
type status int

const (
	statusCreated status = iota // процесс создан, но не запущен
	statusRunning               // процесс запущен и работает
	statusStopped               // процесс остановлен
)

// PyProcess управляет выполнением Python-скрипта в виртуальном окружении
type PyProcess struct {
	cmd        *exec.Cmd          // команда для выполнения
	ctx        context.Context    // контекст процесса
	cancel     context.CancelFunc // функция отмены контекста
	workingDir string             // рабочий каталог
	venv       string             // путь к виртуальному окружению
	script     string             // путь к скрипту
	args       []string           // аргументы командной строки
	status     status             // текущее состояние процесса
	mu         sync.Mutex         // мьютекс для безопасного доступа к состоянию
	stdout     io.Writer          // stdout процесса
	stderr     io.Writer          // stderr процесса
	stdin      io.Reader          // stdin процесса
}

// NewPyProcess создает новый экземпляр PyProcess
// workingDir - путь к рабочему каталогу
// opts - опции для настройки процесса
func NewPyProcess(workingDir string, opts ...Option) (*PyProcess, error) {
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("недопустимый рабочий каталог: %w", err)
	}
	if _, err := os.Stat(absWorkingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("рабочий каталог не существует: %w", err)
	}
	p := &PyProcess{
		workingDir: absWorkingDir,
		venv:       filepath.Join(absWorkingDir, "venv"),
		script:     filepath.Join(absWorkingDir, "main.py"),
		status:     statusCreated,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		stdin:      os.Stdin,
	}
	for _, option := range opts {
		option(p)
	}
	if p.ctx == nil {
		p.ctx, p.cancel = context.WithCancel(context.Background())
	}
	return p, nil
}

// Option определяет тип функции для настройки PyProcess
type Option func(*PyProcess)

// WithVenv устанавливает абсолютный путь к виртуальному окружению
func WithVenv(absPath string) Option {
	return func(p *PyProcess) {
		p.venv = absPath
	}
}

// WithVenvDir устанавливает относительный путь к виртуальному окружению (относительно workingDir)
func WithVenvDir(venvDir string) Option {
	return func(p *PyProcess) {
		p.venv = filepath.Join(p.workingDir, venvDir)
	}
}

// WithScript устанавливает абсолютный путь к скрипту
func WithScript(absPath string) Option {
	return func(p *PyProcess) {
		p.script = absPath
	}
}

// WithScriptName устанавливает имя скрипта (ищет в workingDir)
func WithScriptName(scriptName string) Option {
	return func(p *PyProcess) {
		p.script = filepath.Join(p.workingDir, scriptName)
	}
}

// WithContext устанавливает контекст для процесса
func WithContext(ctx context.Context) Option {
	return func(p *PyProcess) {
		p.ctx, p.cancel = context.WithCancel(ctx)
	}
}

// WithArgs устанавливает аргументы командной строки для скрипта
func WithArgs(args ...string) Option {
	return func(p *PyProcess) {
		p.args = args
	}
}

// WithStdout устанавливает вывод для stdout
func WithStdout(w io.Writer) Option {
	return func(p *PyProcess) {
		p.stdout = w
	}
}

// WithStderr устанавливает вывод для stderr
func WithStderr(w io.Writer) Option {
	return func(p *PyProcess) {
		p.stderr = w
	}
}

// WithStdin устанавливает ввод для stdin
func WithStdin(r io.Reader) Option {
	return func(p *PyProcess) {
		p.stdin = r
	}
}

// WithStdWriter устанавливает общий вывод для stdout и stderr
func WithStdWriter(w io.Writer) Option {
	return func(p *PyProcess) {
		p.stdout = w
		p.stderr = w
	}
}

// Start запускает Python процесс
func (p *PyProcess) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != statusCreated {
		return fmt.Errorf("недопустимое состояние процесса: %v", p.status)
	}
	if _, err := os.Stat(p.script); os.IsNotExist(err) {
		return fmt.Errorf("скрипт не найден: %s", p.script)
	}
	var pythonPath string
	if runtime.GOOS == "windows" {
		pythonPath = filepath.Join(p.venv, "Scripts", "python.exe")
	} else {
		pythonPath = filepath.Join(p.venv, "bin", "python")
	}
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		return fmt.Errorf("python в venv не найден: %s", pythonPath)
	}
	args := append([]string{p.script}, p.args...)
	p.cmd = exec.CommandContext(p.ctx, pythonPath, args...)
	p.cmd.Dir = p.workingDir
	p.cmd.Stdout = p.stdout
	p.cmd.Stderr = p.stderr
	p.cmd.Stdin = p.stdin

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("не удалось запустить Python процесс: %w", err)
	}
	p.status = statusRunning
	return nil
}

// Stop останавливает Python процесс
func (p *PyProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != statusRunning {
		return fmt.Errorf("процесс не был запущен")
	}
	p.cancel()
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()
	select {
	case err := <-done:
		// Процесс завершился самостоятельно
		p.status = statusStopped
		return err
	case <-time.After(5 * time.Second):
		// Если процесс не завершился, отправляем SIGTERM (более мягкий сигнал)
		if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("не удалось отправить SIGTERM: %w", err)
		}
		// Даем дополнительное время на обработку SIGTERM
		select {
		case err := <-done:
			p.status = statusStopped
			return err
		case <-time.After(3 * time.Second):
			// 4. Если все еще не завершился, применяем SIGKILL
			if err := p.cmd.Process.Signal(syscall.SIGKILL); err != nil {
				return fmt.Errorf("не удалось отправить SIGKILL: %w", err)
			}
			p.status = statusStopped
			return nil
		}
	}
}

// IsRunning проверяет, работает ли процесс
func (p *PyProcess) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != statusRunning || p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	// Проверяем существование процесса
	process, err := os.FindProcess(p.cmd.Process.Pid)
	if err != nil {
		return false
	}
	// Посылаем нулевой сигнал для проверки работы процесса
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Wait ожидает завершения процесса
func (p *PyProcess) Wait() error {
	p.mu.Lock()
	if p.status != statusRunning {
		p.mu.Unlock()
		return fmt.Errorf("процесс не был запущен")
	}
	cmd := p.cmd
	p.mu.Unlock()
	return cmd.Wait()
}

// PID возвращает ID процесса или -1 если процесс не запущен
func (p *PyProcess) PID() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != statusRunning || p.cmd == nil || p.cmd.Process == nil {
		return -1
	}
	return p.cmd.Process.Pid
}

// SetStdout устанавливает вывод для stdout процесса
func (p *PyProcess) SetStdout(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stdout = w
	if p.cmd != nil {
		p.cmd.Stdout = w
	}
}

// SetStderr устанавливает вывод для stderr процесса
func (p *PyProcess) SetStderr(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stderr = w
	if p.cmd != nil {
		p.cmd.Stderr = w
	}
}

// SetStdin устанавливает ввод для stdin процесса
func (p *PyProcess) SetStdin(r io.Reader) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stdin = r
	if p.cmd != nil {
		p.cmd.Stdin = r
	}
}
