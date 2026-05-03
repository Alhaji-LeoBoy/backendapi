package logger

import (
	"log"
	"os"
	"time"
)

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	warnLogger  *log.Logger
}

func New() *Logger {
	return &Logger{
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime),
		warnLogger:  log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime),
	}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		l.infoLogger.Printf(msg+" %v", fields)
	} else {
		l.infoLogger.Println(msg)
	}
}

func (l *Logger) Error(msg string, err error, fields ...interface{}) {
	if err != nil {
		l.errorLogger.Printf("%s: %v %v", msg, err, fields)
	} else {
		l.errorLogger.Printf("%s %v", msg, fields)
	}
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		l.debugLogger.Printf(msg+" %v", fields)
	} else {
		l.debugLogger.Println(msg)
	}
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		l.warnLogger.Printf(msg+" %v", fields)
	} else {
		l.warnLogger.Println(msg)
	}
}

func (l *Logger) Fatal(msg string, err error) {
	if err != nil {
		l.errorLogger.Fatalf("%s: %v", msg, err)
	} else {
		l.errorLogger.Fatalln(msg)
	}
}

func (l *Logger) Request(method, path string, status int, duration time.Duration) {
	l.infoLogger.Printf("[%s] %s - %d (%v)", method, path, status, duration)
}
