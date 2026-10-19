package log

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/crush/internal/event"
	"github.com/charmbracelet/x/term"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	initOnce    sync.Once
	initialized atomic.Bool
	// panicDir là thư mục ghi panic log (thư mục của tệp log); chưa đặt = thư mục tạm.
	panicDir atomic.Value
)

func Setup(logFile string, debug bool, ws ...io.Writer) {
	initOnce.Do(func() {
		panicDir.Store(filepath.Dir(logFile))
		logRotator := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    10,    // Max size in MB
			MaxBackups: 0,     // Number of backups
			MaxAge:     30,    // Days
			Compress:   false, // Enable compression
		}

		level := slog.LevelInfo
		if debug {
			level = slog.LevelDebug
		}

		opts := &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		}

		var handlers []slog.Handler
		handlers = append(handlers, slog.NewJSONHandler(logRotator, opts))

		for _, w := range ws {
			if w == nil {
				continue
			}
			if f, ok := w.(term.File); ok && term.IsTerminal(f.Fd()) {
				handlers = append(handlers, slog.NewTextHandler(w, opts))
			} else {
				handlers = append(handlers, slog.NewJSONHandler(w, opts))
			}
		}

		slog.SetDefault(slog.New(slog.NewMultiHandler(handlers...)))
		initialized.Store(true)
	})
}

func Initialized() bool {
	return initialized.Load()
}

func RecoverPanic(name string, cleanup func()) {
	if r := recover(); r != nil {
		event.Error(r, "panic", true, "name", name)

		// Ghi vào thư mục log, không phải thư mục hiện tại (thường là repo đang
		// làm, dễ bị commit nhầm), quyền 0600.
		dir, _ := panicDir.Load().(string)
		filename := filepath.Join(cmp.Or(dir, os.TempDir()),
			fmt.Sprintf("crush-panic-%s-%s.log", name, time.Now().Format("20060102-150405")))
		if file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600); err == nil {
			fmt.Fprintf(file, "Panic in %s: %v\n\n", name, r)
			fmt.Fprintf(file, "Time: %s\n\n", time.Now().Format(time.RFC3339))
			fmt.Fprintf(file, "Stack Trace:\n%s\n", debug.Stack())
			file.Close()
		}
		// Khôi phục (terminal…) kể cả khi không ghi được tệp.
		if cleanup != nil {
			cleanup()
		}
	}
}
