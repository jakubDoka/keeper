package klog

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/jakubDoka/keeper/kcfg"
)

type Level int

const (
	LError Level = iota
	LWarn
	LInfo
	LDebug

	LLast
)

var levelStrings = [...]string{
	"ERROR",
	"WARN",
	"INFO",
	"DEBUG",
}

func (l Level) String() string {
	if l >= LLast {
		return "UNKNOWN"
	}
	return levelStrings[l]
}

type Logger struct {
	level      Level
	stacktrace [LDebug + 1]int
	targets    []Target
	finished   bool
}

func (l *Logger) ApplyConfig(cfg kcfg.Log) {
	l.check()

	slice := cfg.StacktraceDepth.Slice()
	copy(l.stacktrace[:], slice)

	if cfg.LogToConsole {
		l.AddTarget(ConsoleTarget{})
	}

	upper := strings.ToUpper(cfg.Level)
	for i := LError; i <= LDebug; i++ {
		if upper == i.String() {
			l.level = i
		}
	}
}

func (l *Logger) SetLevel(level Level) {
	l.check()
	l.level = level
}

func (l *Logger) AddTarget(target ...Target) {
	l.check()
	l.targets = append(l.targets, target...)
}

func (l *Logger) SetStacktraceDepth(level Level, depth int) {
	l.check()
	l.stacktrace[level] = depth
}

func (l *Logger) check() {
	if l.finished {
		panic("logger is currently in use so config cannot be modified")
	}
}

func (l *Logger) Finish() {
	l.finished = true
}

func (l *Logger) Finished() bool {
	return l.finished
}

func (l *Logger) Debug(message string, args ...interface{}) {
	l.Log(LDebug, message, args...)
}

func (l *Logger) Info(message string, args ...interface{}) {
	l.Log(LInfo, message, args...)
}

func (l *Logger) Warn(message string, args ...interface{}) {
	l.Log(LWarn, message, args...)
}

func (l *Logger) Error(message string, args ...interface{}) {
	l.Log(LError, message, args...)
}

func (l *Logger) Fatal(message string, args ...interface{}) {
	l.Log(LError, message, args...)
	os.Exit(1)
}

func (l *Logger) Log(level Level, message string, args ...interface{}) {
	if l.level >= level {
		var lineInfo string

		callers := make([]uintptr, l.stacktrace[level])
		if len(callers) > 0 {
			amount := runtime.Callers(3, callers)
			frames := runtime.CallersFrames(callers[:amount])
			for {
				frame, more := frames.Next()
				lineInfo += fmt.Sprintf("%s:%d\n", frame.File, frame.Line)
				if !more {
					break
				}
			}
		}

		for _, target := range l.targets {
			target.Write(level, time.Now(), lineInfo, fmt.Sprintf(message, args...))
		}
	}
}

type Target interface {
	Write(level Level, t time.Time, lineInfo, message string)
}

type ConsoleTarget struct{}

func (c ConsoleTarget) Write(level Level, t time.Time, lineInfo, message string) {
	str := "<%s> %s\n%s%s\n"
	if lineInfo == "" {
		str = "<%s> %s%s %s\n"
	}
	fmt.Printf(str, level, t.Format("2006/01/02 15:04:05"), lineInfo, message)
}
