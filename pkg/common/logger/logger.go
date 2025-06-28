package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

var (
	debugMode    bool
	infoColor    = color.New(color.FgCyan)
	successColor = color.New(color.FgGreen)
	warningColor = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)
)

// SetDebugMode enables or disables debug logging
func SetDebugMode(enabled bool) {
	debugMode = enabled
}

// Info prints an info message
func Info(format string, a ...interface{}) {
	logMessage(infoColor, "INFO", format, a...)
}

// Success prints a success message
func Success(format string, a ...interface{}) {
	logMessage(successColor, "SUCCESS", format, a...)
}

// Warning prints a warning message
func Warning(format string, a ...interface{}) {
	logMessage(warningColor, "WARNING", format, a...)
}

// Error prints an error message
func Error(format string, a ...interface{}) {
	logMessage(errorColor, "ERROR", format, a...)
}

// Debug prints a debug message if debug mode is enabled
func Debug(format string, a ...interface{}) {
	if debugMode {
		logMessage(infoColor, "DEBUG", format, a...)
	}
}

func logMessage(c *color.Color, level, format string, a ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", timestamp, c.Sprint(level), message)
}
