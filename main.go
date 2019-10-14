package main

import (
  "net/http"
)

func subdomainName() string { return "default" }
func bucketName() string { return "redding-flex-server.appspot.com" }
func shouldAcceptWebSocket(request *http.Request) bool { return false }

func start() {
}

func handle(writer http.ResponseWriter, request *http.Request, system *System) {
	system.FileServe(request)
	// system.WebServe(request)
}
