// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/chappjc/webfiles/middleware"
	"github.com/chappjc/webfiles/response"

	"github.com/chappjc/webfiles/server"
)

var listen = flag.String("host", "127.0.0.1:7777", "webfiles listens on host:port")
var signingKey = flag.String("signingkey", "", "Signing key for JWT and sessions.")
var maxFileSize = flag.Int64("maxfilesize", 32<<22, "Maximum uploaded file size permitted.")

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

func _main() error {
	defer func() {
		logFILE.Close()
		os.Stdout.Sync()
	}()
	flag.Parse()

	svr := server.NewServer(*signingKey, *maxFileSize)
	webMux := server.NewRouter(svr)

	return http.ListenAndServe(*listen, webMux)
}

func main() {
	if err := _main(); err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
