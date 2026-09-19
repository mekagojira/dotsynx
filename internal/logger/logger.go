package logger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Level string

const (
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
	LevelSync  Level = "SYNC"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
	Raw       string    `json:"raw"`
}

var (
	logMu    sync.Mutex
	logFile  *os.File
	initOnce sync.Once
)

// DefaultLogPath returns ~/.local/share/dotsynx/dotsynx.log
func DefaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "dotsynx", "dotsynx.log")
}

// Init initializes logging to ~/.local/share/dotsynx/dotsynx.log
func Init() error {
	var err error
	initOnce.Do(func() {
		p := DefaultLogPath()
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		logFile, err = os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	})
	return err
}

// Log writes a message to the log file
func Log(level Level, format string, args ...any) {
	_ = Init()

	logMu.Lock()
	defer logMu.Unlock()

	now := time.Now()
	msg := fmt.Sprintf(format, args...)
	entryStr := fmt.Sprintf("%s [%s] %s\n", now.Format("2006-01-02 15:04:05"), level, msg)

	if logFile != nil {
		_, _ = logFile.WriteString(entryStr)
	}
}

func Info(format string, args ...any) {
	Log(LevelInfo, format, args...)
}

func Warn(format string, args ...any) {
	Log(LevelWarn, format, args...)
}

func Error(format string, args ...any) {
	Log(LevelError, format, args...)
}

func Sync(format string, args ...any) {
	Log(LevelSync, format, args...)
}

// ReadEntries reads recent log entries from the log file
func ReadEntries(maxLines int, since time.Duration) ([]LogEntry, error) {
	p := DefaultLogPath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return []LogEntry{}, nil
	}

	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	cutoff := time.Time{}
	if since > 0 {
		cutoff = time.Now().Add(-since)
	}

	var entries []LogEntry
	for _, line := range lines {
		entry := parseLine(line)
		if entry == nil {
			continue
		}
		if !cutoff.IsZero() && entry.Timestamp.Before(cutoff) {
			continue
		}
		entries = append(entries, *entry)
	}

	if maxLines > 0 && len(entries) > maxLines {
		entries = entries[len(entries)-maxLines:]
	}

	// Reverse to descending order (newest first)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// Tail writes existing and incoming log entries to out until ctx is canceled (ascending order for streaming)
func Tail(ctx context.Context, lines int, out io.Writer) error {
	p := DefaultLogPath()
	_ = Init()

	entries, _ := ReadEntries(lines, 0)
	// Output in chronological order for live tailing
	for i := len(entries) - 1; i >= 0; i-- {
		fmt.Fprintln(out, entries[i].Raw)
	}

	file, err := os.Open(p)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to end
	_, _ = file.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(file)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					fmt.Fprint(out, line)
				}
				if err != nil {
					break
				}
			}
		}
	}
}

func parseLine(line string) *LogEntry {
	if len(line) < 26 {
		return nil
	}

	tsStr := line[:19]
	t, err := time.Parse("2006-01-02 15:04:05", tsStr)
	if err != nil {
		return nil
	}

	rest := strings.TrimSpace(line[19:])
	if !strings.HasPrefix(rest, "[") {
		return nil
	}
	endIdx := strings.Index(rest, "]")
	if endIdx == -1 {
		return nil
	}

	level := Level(rest[1:endIdx])
	msg := strings.TrimSpace(rest[endIdx+1:])

	return &LogEntry{
		Timestamp: t,
		Level:     level,
		Message:   msg,
		Raw:       line,
	}
}
