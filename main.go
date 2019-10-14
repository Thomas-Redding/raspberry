package main

import (
	"log"
	"os"
  "net/http"
)

func bucketRoot(isLocalHost bool) string {
	if isLocalHost {
		pwd, err := os.Getwd()
		if err != nil {
			log.Printf("Error in bucketPath(): %v", err)
			return ""
		}
		return pwd + "/data"
	} else {
		log.Printf("ERROR: DO NOT RUN THIS IN PRODUCTION")
		// Production
		return ""
	}
}

func bucketName() string { return "redding-flex-server.appspot.com" }
func shouldAcceptWebSocket(request *http.Request) bool { return false }

func start() {
}

func handle(writer http.ResponseWriter, request *http.Request, system *System) {
	system.FileServe(request)
	// system.WebServe(request)
}
