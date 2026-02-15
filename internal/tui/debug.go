package tui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
)

var (
	debugLogger *log.Logger
	debugOnce   sync.Once
)

// debugLog writes a message to the debug log file.
// Only active when FR8_TUI_DEBUG=1. Log file: /tmp/fr8-tui-debug.log
func debugLog(format string, args ...any) {
	debugOnce.Do(func() {
		if os.Getenv("FR8_TUI_DEBUG") != "1" {
			return
		}
		f, err := os.OpenFile("/tmp/fr8-tui-debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return
		}
		debugLogger = log.New(f, "", log.Ltime|log.Lmicroseconds)

		// Dump environment info on first call
		debugLogger.Printf("=== fr8 TUI debug log ===")
		debugLogger.Printf("TERM=%s", os.Getenv("TERM"))
		debugLogger.Printf("TERM_PROGRAM=%s", os.Getenv("TERM_PROGRAM"))
		debugLogger.Printf("TERM_PROGRAM_VERSION=%s", os.Getenv("TERM_PROGRAM_VERSION"))
		debugLogger.Printf("COLORTERM=%s", os.Getenv("COLORTERM"))
		debugLogger.Printf("TERMINAL_EMULATOR=%s", os.Getenv("TERMINAL_EMULATOR"))
		debugLogger.Printf("LANG=%s", os.Getenv("LANG"))

		// Test box-drawing character widths
		chars := []struct {
			name string
			ch   string
		}{
			{"╭", "╭"}, {"╮", "╮"}, {"╰", "╰"}, {"╯", "╯"},
			{"│", "│"}, {"─", "─"}, {"▸", "▸"}, {"▶", "▶"},
			{"●", "●"}, {"✓", "✓"}, {"✗", "✗"}, {"·", "·"},
		}
		for _, c := range chars {
			debugLogger.Printf("char %s: ansi.StringWidth=%d len(bytes)=%d", c.name, ansi.StringWidth(c.ch), len(c.ch))
		}
	})

	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

// debugLogView logs diagnostic info about the final rendered view.
func debugLogView(view string, modelWidth, modelHeight int) {
	if debugLogger == nil {
		return
	}
	lines := strings.Split(view, "\n")
	debugLog("--- View render: model.width=%d model.height=%d lines=%d ---", modelWidth, modelHeight, len(lines))
	for i, line := range lines {
		w := ansi.StringWidth(line)
		marker := ""
		if w != modelWidth {
			marker = fmt.Sprintf(" *** MISMATCH (expected %d)", modelWidth)
		}
		if i < 30 || marker != "" { // Log first 30 lines, plus any mismatches
			debugLog("  line[%2d] width=%3d%s", i, w, marker)
		}
	}
}
