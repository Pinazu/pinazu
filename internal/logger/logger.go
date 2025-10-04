package logger

import (
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// HclogColorWriter wraps an io.Writer to provide colored log output formatting.
type HclogColorWriter struct {
	writer io.Writer
}

// ANSI color codes for terminal output formatting.
const (
	Reset   = "\033[0m"
	White   = "\033[37m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Cyan    = "\033[36m"
	Blue    = "\033[34m"
	Gray    = "\033[90m"
	Magenta = "\033[35m"
	Bold    = "\033[1m"
)

// getLogLevelColor returns the appropriate ANSI color code for a log level.
func getLogLevelColor(level string) string {
	switch level {
	case "INFO":
		return Green
	case "WARN":
		return Yellow
	case "ERROR":
		return Red
	case "DEBUG":
		return Cyan
	default:
		return Gray
	}
}

// NewHclogColorWriter creates a new colored log writer that wraps the given io.Writer.
func NewHclogColorWriter(w io.Writer) *HclogColorWriter {
	return &HclogColorWriter{writer: w}
}

// Write implements io.Writer interface with colored log formatting.
func (hcw *HclogColorWriter) Write(p []byte) (n int, err error) {
	line := strings.TrimSpace(string(p))
	if line == "" {
		return hcw.writer.Write(p)
	}

	var result strings.Builder

	// Updated regex to match the actual format from your terminal
	// Handle format: timestamp [level] location: logger_name message
	// 2025-07-22T09:25:59.391Z [INFO] cli/cli.go:170: agent handler service...
	logRegex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[([A-Z]+)\]\s+(.*)$`)

	matches := logRegex.FindStringSubmatch(line)
	if len(matches) != 4 {
		// If regex doesn't match, just output the line as-is
		return hcw.writer.Write([]byte(line + "\n"))
	}

	timestamp := matches[1]
	level := matches[2]
	locationAndContent := matches[3]

	// Split location and content at the first colon
	parts := strings.SplitN(locationAndContent, ":", 2)
	if len(parts) != 2 {
		// If no colon found, output as-is
		return hcw.writer.Write([]byte(line + "\n"))
	}

	location := strings.TrimSpace(parts[0])
	content := strings.TrimSpace(parts[1])

	// Remove duplicate location from content if it exists
	content = removeDuplicateLocation(content, location)

	// Parse message and key-value pairs from cleaned content
	message, kvPairs := parseMessageAndKeyValues(content)

	// 1. Timestamp (blue)
	result.WriteString(Blue + timestamp + Reset + " ")

	// 2. Log level (colored based on level)
	levelColor := getLogLevelColor(level)
	result.WriteString(levelColor + Bold + "[" + level + "]" + Reset + " ")

	// 3. Location file (magenta/light purple)
	result.WriteString(Magenta + location + Reset + " ")

	// 4. Message (white)
	if message != "" {
		result.WriteString(White + message + Reset)
	}

	// 5. Key-value pairs (key=gray, value=cyan)
	for _, kv := range kvPairs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			result.WriteString(" " + Gray + parts[0] + Reset + "=" + Cyan + parts[1] + Reset)
		}
	}

	result.WriteString("\n")
	return hcw.writer.Write([]byte(result.String()))
}

// removeDuplicateLocation removes duplicate location prefixes from log content.
func removeDuplicateLocation(content, location string) string {
	// Remove duplicate location: pattern from the beginning of content
	locationPattern := location + ": "
	if after, ok := strings.CutPrefix(content, locationPattern); ok {
		content = after
	}

	return content
}

// parseMessageAndKeyValues extracts the main message and key-value pairs from log content.
func parseMessageAndKeyValues(input string) (string, []string) {
	// Find key=value patterns at the end of the string
	// Updated regex to handle quoted values with spaces
	kvRegex := regexp.MustCompile(`\s+(\w+)=("(?:[^"\\]|\\.)*"|[^\s]+)`)
	matches := kvRegex.FindAllStringSubmatchIndex(input, -1)

	var kvPairs []string
	message := input

	if len(matches) > 0 {
		// Extract message (everything before first key=value)
		firstKVStart := matches[0][0]
		message = strings.TrimSpace(input[:firstKVStart])

		// Extract all key=value pairs
		for _, match := range matches {
			keyStart, keyEnd := match[2], match[3]
			valueStart, valueEnd := match[4], match[5]
			key := input[keyStart:keyEnd]
			value := input[valueStart:valueEnd]
			kvPairs = append(kvPairs, key+"="+value)
		}
	}

	// Clean up the message by removing any trailing colon artifacts
	message = strings.TrimSuffix(message, ":")
	message = strings.TrimSpace(message)

	return message, kvPairs
}

// NewCustomColorLog creates a new hclog.Logger with custom color formatting.
func NewCustomColorLog() hclog.Logger {
	return NewCustomColorLogWithLevel(hclog.Info)
}

// NewCustomColorLogWithLevel creates a new hclog.Logger with custom color formatting and specified log level.
func NewCustomColorLogWithLevel(level hclog.Level) hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Level:           level,
		Output:          NewHclogColorWriter(os.Stdout),
		Color:           hclog.ColorOff,
		IncludeLocation: true,
	})
}
