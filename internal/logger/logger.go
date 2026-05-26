package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lly0010/winphp2025/internal/paths"
)

// Logger 维护循环 buffer + 同步写文件, 前端可通过 Tail() 拉取最近 N 条.
// 同时通过 channel 实时推送给 WailsRuntime (App 侧订阅).

type Entry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

var (
	mu      sync.Mutex
	ring    []Entry
	maxRing = 1000
	logF    *os.File
	subs    []chan Entry
	openErr error
)

func ensureOpen() {
	if logF != nil || openErr != nil {
		return
	}
	p := filepath.Join(paths.LogsDir, "winphp.log")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		openErr = err
		return
	}
	logF, openErr = os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}

func write(level, msg string) {
	mu.Lock()
	defer mu.Unlock()
	ensureOpen()
	e := Entry{
		Time:  time.Now().Format("2006-01-02 15:04:05"),
		Level: level,
		Msg:   msg,
	}
	ring = append(ring, e)
	if len(ring) > maxRing {
		ring = ring[len(ring)-maxRing:]
	}
	if logF != nil {
		fmt.Fprintf(logF, "[%s] [%s] %s\n", e.Time, e.Level, e.Msg)
		_ = logF.Sync()
	}
	// 通知订阅者 (非阻塞)
	for _, ch := range subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func Info(format string, a ...any)  { write("INFO", fmt.Sprintf(format, a...)) }
func Warn(format string, a ...any)  { write("WARN", fmt.Sprintf(format, a...)) }
func Error(format string, a ...any) { write("ERROR", fmt.Sprintf(format, a...)) }

func Tail(n int) []Entry {
	mu.Lock()
	defer mu.Unlock()
	if n <= 0 || n > len(ring) {
		n = len(ring)
	}
	out := make([]Entry, n)
	copy(out, ring[len(ring)-n:])
	return out
}

func Subscribe() chan Entry {
	mu.Lock()
	defer mu.Unlock()
	ch := make(chan Entry, 64)
	subs = append(subs, ch)
	return ch
}

func Unsubscribe(target chan Entry) {
	mu.Lock()
	defer mu.Unlock()
	for i, ch := range subs {
		if ch == target {
			subs = append(subs[:i], subs[i+1:]...)
			close(target)
			return
		}
	}
}
