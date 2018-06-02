package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	prefix_fmt "github.com/chappjc/logrus-prefix"
	"github.com/shiena/ansicolor"
	"github.com/sirupsen/logrus"
)

var (
	log     *logrus.Logger
	logFILE *os.File
)

const logFileName = "webfiles.log"

func startLogger() error {
	logFilePath, _ := filepath.Abs(logFileName)
	var err error
	logFILE, err = os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0664)
	if err != nil {
		return fmt.Errorf("Error opening log file: %v", err)
	}

	logrus.SetOutput(io.MultiWriter(logFILE, os.Stdout))
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&prefix_fmt.TextFormatter{ForceColors: true})

	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &prefix_fmt.TextFormatter{
		ForceColors:     true,
		ForceFormatting: true,
		FullTimestamp:   true,
		TimestampFormat: "02 Jan 06 15:04.00 -0700",
	}
	log.Out = ansicolor.NewAnsiColorWriter(io.MultiWriter(logFILE, os.Stdout))
	return nil
}

func setLogLevel(level string) error {
	Level, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level: %v", err)
	}

	logrus.SetLevel(Level)
	log.Level = Level
	return nil
}
