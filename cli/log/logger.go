package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	c "github.com/WOo0W/bowerbird/cli/color"
)

// Level defines log level of the logger
type Level int

// Logging levels
const (
	DEBUG Level = iota
	INFO
	// NOTICE
	WARN
	ERROR
	LINE
	PRINT
)

// Default formats
const (
	timeFormat = "01/02 15:04:05"
	logFormat  = "\r%s [%s] %s"
)

// Colored level strings
var (
	cDebug = c.SHiBlue("DEBUG")
	cInfo  = c.SHiGreen("INFO")
	// cNotice = c.SHiGreen("NOTICE")
	cWarn  = c.SHiYellow("WARN")
	cError = c.SHiRed("ERROR")
)

func timeNowString() string {
	t := time.Now()
	return t.Format(timeFormat)
}

// Logger defiles the logging output and level
type Logger struct {
	ConsoleOutput, FileOutput io.Writer
	ConsoleLevel, FileLevel   Level
	MaxLength                 int
	Format                    string
	// LineRefreshRate           time.Duration
}

// New return's a new Logger printing colored messages to Stderr
func New() *Logger {
	return &Logger{
		ConsoleOutput: c.Stderr,
		FileOutput:    ioutil.Discard,
		ConsoleLevel:  INFO,
		FileLevel:     10, // Will not output anything to FileOutput
		MaxLength:     60,
		Format:        logFormat,
		// LineRefreshRate: 250 * time.Millisecond,
	}
}

// G defines the default global Logger
var G = New()

// Debug logs debug level messages
func (l *Logger) Debug(a ...interface{}) {
	var (
		times   string
		message string
	)
	if l.ConsoleLevel <= ERROR || l.FileLevel <= ERROR {
		times = timeNowString()
		message = fmt.Sprintln(a...)
	}
	if l.ConsoleLevel <= DEBUG {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(times), cDebug, message)
	}
	if l.FileLevel <= DEBUG {
		fmt.Fprintf(l.FileOutput, logFormat, times, "DEBUG", message)
	}
}

// Info logs info level messages
func (l *Logger) Info(a ...interface{}) {
	var (
		times   string
		message string
	)
	if l.ConsoleLevel <= INFO || l.FileLevel <= INFO {
		times = timeNowString()
		message = fmt.Sprintln(a...)
	}
	if l.ConsoleLevel <= INFO {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(times), cInfo, message)
	}
	if l.FileLevel <= INFO {
		fmt.Fprintf(l.FileOutput, logFormat, times, "INFO", message)
	}
}

// Warn logs warn level messages
func (l *Logger) Warn(a ...interface{}) {
	var (
		times   string
		message string
	)
	if l.ConsoleLevel <= WARN || l.FileLevel <= WARN {
		times = timeNowString()
		message = fmt.Sprintln(a...)
	}
	if l.ConsoleLevel <= WARN {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(times), cWarn, c.SHiYellow(message))
	}
	if l.FileLevel <= WARN {
		fmt.Fprintf(l.FileOutput, logFormat, times, "WARN", message)
	}
}

// Error logs error level messages
func (l *Logger) Error(a ...interface{}) {
	var (
		times   string
		message string
	)
	if l.ConsoleLevel <= ERROR || l.FileLevel <= ERROR {
		times = timeNowString()
		message = fmt.Sprintln(a...)
	}
	if l.ConsoleLevel <= ERROR {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(times), cError, c.SHiRed(message))
	}
	if l.FileLevel <= ERROR {
		fmt.Fprintf(l.FileOutput, logFormat, times, "ERROR", message)
	}
}

// Line refreshes the latest line in console with message
func (l *Logger) Line(message string) {
	if l.ConsoleLevel <= LINE {
		var ss string
		sr := []rune(message)
		lm := len(message)
		if lm > l.MaxLength-5 {
			ss = "..." + string(sr[lm-l.MaxLength+5:])
		} else {
			ss = message + strings.Repeat(" ", l.MaxLength-2-lm)
		}
		fmt.Fprint(l.ConsoleOutput, "\r", ss)
	}
}

// Print logs print level messages without modified
func (l *Logger) Print(a ...interface{}) {
	if l.ConsoleLevel <= PRINT {
		fmt.Fprint(l.ConsoleOutput, a...)
	}
	if l.FileLevel <= PRINT {
		fmt.Fprint(l.FileOutput, a...)
	}
}

type LoggingRoundTripper struct {
	Logger   *Logger
	Underlay http.RoundTripper
}

func (lr *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	lr.Logger.Debug(req.Method, req.URL)
	return lr.Underlay.RoundTrip(req)
}

func NewLoggingRoundTripper(l *Logger, r http.RoundTripper) *LoggingRoundTripper {
	return &LoggingRoundTripper{
		Logger:   l,
		Underlay: r,
	}
}
