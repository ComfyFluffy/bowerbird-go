package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	c "github.com/WOo0W/bowerbird/cli/color"
)

// Level defines log level of the logger
type Level uint8

// Logging levels
const (
	DEBUG Level = iota
	INFO
	NOTICE
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
	cDebug  = c.SHiBlue("DEBUG")
	cInfo   = c.SHiCyan("INFO")
	cNotice = c.SHiGreen("NOTICE")
	cWarn   = c.SHiYellow("WARN")
	cError  = c.SHiRed("ERROR")
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
	// LineRefreshRate           time.Duration
}

// New return's a new Logger printing colored messages to Stderr
func New() *Logger {
	return &Logger{
		ConsoleOutput: c.Stderr,
		FileOutput:    ioutil.Discard,
		ConsoleLevel:  NOTICE,
		FileLevel:     10, // Will not output anything to FileOutput
		MaxLength:     60,
		// LineRefreshRate: 250 * time.Millisecond,
	}
}

// G defines the default global Logger
var G = New()

// Debug logs debug level messages
func (l *Logger) Debug(a ...interface{}) {
	var timeString string
	if l.ConsoleLevel <= DEBUG || l.FileLevel <= DEBUG {
		timeString = timeNowString()
	}
	if l.ConsoleLevel <= DEBUG {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(timeString), cDebug, fmt.Sprintln(a...))
	}
	if l.FileLevel <= DEBUG {
		fmt.Fprintf(l.FileOutput, logFormat, timeString, "DEBUG", fmt.Sprintln(a...))
	}
}

// Info logs info level messages
func (l *Logger) Info(a ...interface{}) {
	var timeString string
	if l.ConsoleLevel <= INFO || l.FileLevel <= INFO {
		timeString = timeNowString()
	}
	if l.ConsoleLevel <= INFO {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(timeString), cInfo, fmt.Sprintln(a...))
	}
	if l.FileLevel <= INFO {
		fmt.Fprintf(l.FileOutput, logFormat, timeString, "INFO", fmt.Sprintln(a...))
	}
}

// Notice logs notice level messages
func (l *Logger) Notice(a ...interface{}) {
	var timeString string
	if l.ConsoleLevel <= NOTICE || l.FileLevel <= NOTICE {
		timeString = timeNowString()
	}
	if l.ConsoleLevel <= NOTICE {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(timeString), cNotice, fmt.Sprintln(a...))
	}
	if l.FileLevel <= NOTICE {
		fmt.Fprintf(l.FileOutput, logFormat, timeString, "NOTICE", fmt.Sprintln(a...))
	}
}

// Warn logs warn level messages
func (l *Logger) Warn(a ...interface{}) {
	var timeString string
	if l.ConsoleLevel <= WARN || l.FileLevel <= WARN {
		timeString = timeNowString()
	}
	if l.ConsoleLevel <= WARN {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(timeString), cWarn, c.SHiYellow(fmt.Sprintln(a...)))
	}
	if l.FileLevel <= WARN {
		fmt.Fprintf(l.FileOutput, logFormat, timeString, "WARN", fmt.Sprintln(a...))
	}
}

// Error logs error level messages
func (l *Logger) Error(a ...interface{}) {
	var timeString string
	if l.ConsoleLevel <= ERROR || l.FileLevel <= ERROR {
		timeString = timeNowString()
	}
	if l.ConsoleLevel <= ERROR {
		fmt.Fprintf(l.ConsoleOutput, logFormat, c.SHiBlack(timeString), cError, c.SHiRed(fmt.Sprintln(a...)))
	}
	if l.FileLevel <= ERROR {
		fmt.Fprintf(l.FileOutput, logFormat, timeString, "ERROR", fmt.Sprintln(a...))
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
		fmt.Fprint(l.ConsoleOutput, "/r", ss)
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
