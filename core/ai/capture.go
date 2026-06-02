package ai

import (
	"io"
	"os"
	"sync"
)

// LineBuffer captures the last N lines written to it for failure diagnostics.
type LineBuffer struct {
	mu     sync.Mutex
	lines  []string
	limit  int
}

func newLineBuffer(limit int) *LineBuffer {
	return &LineBuffer{limit: limit}
}

func (b *LineBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	text := string(p)
	for _, line := range splitLines(text) {
		if line == "" {
			continue
		}
		b.lines = append(b.lines, line)
		if len(b.lines) > b.limit {
			b.lines = b.lines[len(b.lines)-b.limit:]
		}
	}
	return len(p), nil
}

func (b *LineBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.lines))
	copy(out, b.lines)
	return out
}

func (b *LineBuffer) Text() string {
	lines := b.Lines()
	if len(lines) == 0 {
		return ""
	}
	var out string
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

// teeWriter writes to stdout/stderr and the line buffer simultaneously.
type teeWriter struct {
	primary io.Writer
	buffer  *LineBuffer
}

func (t *teeWriter) Write(p []byte) (int, error) {
	t.buffer.Write(p)
	return t.primary.Write(p)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, trimCR(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, trimCR(s[start:]))
	}
	return lines
}

func trimCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}

// LiveOutputWriters returns stdout/stderr writers that tee to a rolling line buffer.
func LiveOutputWriters(limit int) (stdout, stderr io.Writer, buffer *LineBuffer) {
	buf := newLineBuffer(limit)
	return &teeWriter{primary: os.Stdout, buffer: buf},
		&teeWriter{primary: os.Stderr, buffer: buf},
		buf
}
