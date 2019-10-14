package main

import (
  "net/http"
)

func bucketRoot() string {
  return "/Users/John/files/to/serve"
}

func start() {}

func handle(writer http.ResponseWriter, request *http.Request, system *System) {
  system.FileServe(request)
}
