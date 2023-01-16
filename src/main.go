package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
)

var cameras *Cameras

func main() {
	var customFormatter = log.TextFormatter{}
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	var formatter log.Formatter = &customFormatter
	log.SetFormatter(formatter)

	log.SetLevel(log.InfoLevel)
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.ToSlash("/var/log/mp4f-server/mp4f-server.log"),
		MaxSize:    5, // MB
		MaxBackups: 10,
		MaxAge:     30, // days
		Compress:   true,
	}
	gin.DefaultWriter = io.MultiWriter(os.Stdout, lumberjackLogger)
	log.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))
	cameras = loadConfig()
	_ = cameras
	ffmpegFeed(cameras)
	serveHTTP()
}
