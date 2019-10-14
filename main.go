package main

import (
	"fmt"
	"io/ioutil"
	"log"
  "net/http"
  "os"
  "strings"
  "strconv"
)

var ROOT_PATH string

var STYLE_STRING = `
<style>
body {
	background-color: #222;
	color: white;
}

a {
	color: rgb(255, 221, 27);
}

</style>
`

func main() {
	ROOT_PATH = os.Args[1]
  http.HandleFunc("/", handle)
  log.Printf("FATAL ERROR: %v", http.ListenAndServe(":8080", nil))
}

func handle(writer http.ResponseWriter, request *http.Request) {
	path := ROOT_PATH + request.URL.Path
	log.Printf("handle(\"%s\")", path)
	file, err := os.Open(path)
	if err != nil {
	  SendError(writer, 404, "File Not Found [a]")
	  return
	}
	defer file.Close()

	fileInfo, err := file.Stat();
	if err != nil {
		SendError(writer, 500, "Internal Server Error [b]")
		return
	}

	isFile := !fileInfo.Mode().IsDir()
	if isFile {
		http.ServeFile(writer, request, path)
		return
	}

	// The path is a directory.
	if !strings.HasSuffix(path, "/") {
		http.Redirect(writer, request, request.URL.Path + "/", http.StatusSeeOther)
		return
	}
  children, err := ioutil.ReadDir(path)
  if err != nil {
  	SendError(writer, 500, "Internal Server Error [c]")
		return
  }
  writer.Header().Set("Content-type", "text/html")
  for _, child := range children {
  	childName := child.Name()
	  if strings.HasPrefix(childName, ".") { continue }

	  file, err := os.Open(path + childName)
	  if err != nil { continue }
	  fileInfo, err = file.Stat();
		if err != nil { continue }
		isFile := !fileInfo.Mode().IsDir()
	  if isFile {
		  writer.Write([]byte("<a href=\"" + childName + "\">"))
		} else {
			writer.Write([]byte("<a href=\"" + childName + "/\">"))
		}
		writer.Write([]byte(childName))
		writer.Write([]byte("</a><br/>"))
	}
	writer.Write([]byte(STYLE_STRING))
}

func SendError(writer http.ResponseWriter, errorCode int, format string, args ...interface{}) {
  errorMessage := "Error " + strconv.Itoa(errorCode) + ": " + fmt.Sprintf(format, args...)
  writer.Header().Set("Content-type", "text/html")
  writer.WriteHeader(errorCode)
  writer.Write([]byte(errorMessage))
  writer.Write([]byte(STYLE_STRING))
  log.Printf(errorMessage)
}

