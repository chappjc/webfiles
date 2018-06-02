// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chappjc/webfiles/middleware"
	"github.com/chappjc/webfiles/response"
	"github.com/chappjc/webfiles/server"
)

var listen = flag.String("host", "127.0.0.1:7777", "webfiles listens on host:port")
var signingKey = flag.String("signingkey", "asdf1234", "Signing key for JWT and sessions.")
var maxFileSize = flag.Int64("maxfilesize", 32<<22, "Maximum uploaded file size permitted.")
var logLevel = flag.String("loglevel", "debug", "Logging level (debug, info, warning, error, fatal, panic)")

func init() {
	err := startLogger()
	if err != nil {
		fmt.Printf("Unable to start logger: %v", err)
		os.Exit(1)
	}

	server.UseLog(log)
	middleware.UseLog(log)
	response.UseLog(log)
}

// _main is wrapped by main so that defers will run.
func _main() error {
	defer func() {
		logFILE.Close()
		os.Stdout.Sync()
	}()
	flag.Parse()
	if err := setLogLevel(*logLevel); err != nil {
		return fmt.Errorf("failed to set log level: %v", err)
	}

	// Create the folder for the secure persistent cookie store.
	cookieStore, err := filepath.Abs("cookiestore")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(cookieStore, 0700); err != nil {
		return err
	}

	// Construct the Server and path multiplexer.
	svr := server.NewServer(*signingKey, cookieStore, *maxFileSize)
	webMux := server.NewRouter(svr)

	log.Infof("webfiles is listening on http://%s.", *listen)
	return http.ListenAndServe(*listen, webMux)
}

func main() {
	if err := _main(); err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
