package gui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Logger provides a thread-safe GUI logger
type Logger struct {
	widget   *widget.Entry
	messages []string
	maxLines int
	mu       sync.Mutex
	enabled  bool
}

// NewLogger creates a new GUI logger
func NewLogger(maxLines int) *Logger {
	entry := widget.NewMultiLineEntry()
	entry.Wrapping = fyne.TextWrapWord
	entry.Disable() // Read-only

	return &Logger{
		widget:   entry,
		messages: make([]string, 0, maxLines),
		maxLines: maxLines,
		enabled:  true,
	}
}

// Log adds a message to the log
func (l *Logger) Log(format string, args ...interface{}) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf("[%s] %s", timestamp, fmt.Sprintf(format, args...))

	// Also print to console for debugging
	fmt.Println(message)

	l.messages = append(l.messages, message)

	// Keep only the last maxLines messages
	if len(l.messages) > l.maxLines {
		l.messages = l.messages[len(l.messages)-l.maxLines:]
	}

	// Update the widget
	l.updateWidget()
}

// LogError logs an error message
func (l *Logger) LogError(format string, args ...interface{}) {
	l.Log("❌ ERROR: "+format, args...)
}

// LogSuccess logs a success message
func (l *Logger) LogSuccess(format string, args ...interface{}) {
	l.Log("✅ "+format, args...)
}

// LogWarning logs a warning message
func (l *Logger) LogWarning(format string, args ...interface{}) {
	l.Log("⚠️  WARNING: "+format, args...)
}

// LogInfo logs an info message
func (l *Logger) LogInfo(format string, args ...interface{}) {
	l.Log("ℹ️  "+format, args...)
}

// Clear clears all log messages
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.messages = make([]string, 0, l.maxLines)
	l.updateWidget()
}

// GetWidget returns the widget for display
func (l *Logger) GetWidget() *widget.Entry {
	return l.widget
}

// updateWidget updates the widget with current messages
func (l *Logger) updateWidget() {
	text := ""
	for _, msg := range l.messages {
		text += msg + "\n"
	}
	l.widget.SetText(text)

	// Scroll to bottom
	l.widget.CursorRow = len(l.messages)
}

// Enable enables logging
func (l *Logger) Enable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = true
}

// Disable disables logging
func (l *Logger) Disable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = false
}
