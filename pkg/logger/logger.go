package logger

import (
	"log"
	"os"
)

var (
	infoLogger  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Infof logs information messages
func Infof(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}

// Errorf logs error messages
func Errorf(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
}
