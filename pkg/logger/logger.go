// In logger/logger.go

package logger

import (
	"log"
	"os"
)

var (
	infoLogger  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	debugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
)

// Infof logs information messages
func Infof(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}

// Debugf logs debug messages
func Debugf(format string, v ...interface{}) {
	debugLogger.Printf(format, v...)
}

// Errorf logs error messages
func Errorf(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
}
