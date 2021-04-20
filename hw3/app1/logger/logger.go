// Package logger implements functions and structs to logging INFO ERROR WARNING messges to log file and to stdout
//
// The InitLoggers func opens(creates) log file and creates several loggers
//
//
//
package logger

import (
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
)

var (
	//Logger *logrus.Logger
)
// InitLoggers - начальная инициализация - лог файлы/ логгеры
// File - log file
func InitLoggers(File string, debug bool, loglevel int) *logrus.Logger {
	file, err := os.OpenFile(File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("error creating log file %v",err)
		return nil
	}
	Logger := logrus.New()
	Logger.Formatter = new(logrus.TextFormatter)                     //default
	Logger.Formatter.(*logrus.TextFormatter).DisableColors = true    // remove colors
	//logger.Formatter.(*logrus.TextFormatter).DisableTimestamp = true // remove timestamp from test output
	Logger.Formatter.(*logrus.TextFormatter).FullTimestamp = true
	Logger.Formatter.(*logrus.TextFormatter).TimestampFormat = "2006-01-02 15:04:05"

	switch loglevel {
	case 3:
		Logger.Level = logrus.DebugLevel
	case 2:
		Logger.Level = logrus.WarnLevel
	case 1:
		Logger.Level = logrus.InfoLevel
	}

	// if debug flag is set print all to screen and to file
	if debug{
		Logger.Out = io.MultiWriter(os.Stdout, file)
	} else {
		Logger.Out = io.Writer(file)
	}

	return Logger
}